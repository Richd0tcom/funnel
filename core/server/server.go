package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richd0tcom/funnel/core/consumer"
	"github.com/richd0tcom/funnel/internal/domain"
	"github.com/richd0tcom/funnel/internal/worker"
)

type Server struct {
	config    *ServerConfig
	worker *worker.Worker
	// metrics   *Metrics
	router    *gin.Engine
}

func NewServer(options ...ConfigOption) (*Server, error) {
	config := &ServerConfig{
		WorkerCount: 4,
		BatchSize:   100,
		Port:        "8080",
	}

	for _, option := range options {
		if err := option(config); err != nil {
			return nil, err
		}
	}

	// Set default consumer if not provided
	if config.Consumer == nil {
		config.Consumer = consumer.NewMockDataConsumer("DefaultConsumer")
	}

	// metrics := NewMetrics()
	processor := worker.NewWorker(config.DataStore, config.Consumer, config.WorkerCount, config.BatchSize)

	server := &Server{
		config:    config,
		worker: processor,
		router:    gin.Default(),
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Metrics endpoint
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	api := s.router.Group("/api/v1")
	{
		api.POST("/ingest", s.handleIngest)
		api.POST("/aggregates", s.handleGetAggregates)
	}
}

func (s *Server) handleIngest(c *gin.Context) {
	var bulkData domain.BulkSensorData
	if err := c.ShouldBindJSON(&bulkData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate data
	if len(bulkData.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no data provided"})
		return
	}

	// Publish to message queue
	data, err := json.Marshal(bulkData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize data"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := s.config.MessageQueue.Publish(ctx, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish data"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "data accepted for processing",
		"count":   len(bulkData.Data),
	})
}

func (s *Server) handleGetAggregates(c *gin.Context) {
	var query domain.AggregateQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate query
	if query.StartTime.IsZero() || query.EndTime.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time and end_time are required"})
		return
	}

	if query.Granularity == "" {
		query.Granularity = "hour"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	results, err := s.config.DataStore.GetAggregates(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get aggregates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

func (s *Server) Start(ctx context.Context) error {
	// Start batch processor
	go func() {
		if err := s.worker.Start(ctx, s.config.MessageQueue); err != nil {
			log.Printf("Batch processor error: %v", err)
		}
	}()

	// Start HTTP server
	server := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("Server starting on port %s", s.config.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Close() error {
	if s.config.MessageQueue != nil {
		s.config.MessageQueue.Close()
	}
	if s.config.DataStore != nil {
		s.config.DataStore.Close()
	}
	return nil
}