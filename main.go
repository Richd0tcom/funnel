package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/richd0tcom/funnel/core/server"
)

func main() {
	messageQueueType := os.Getenv("MESSAGE_QUEUE_TYPE") // kafka, rabbitmq, channels
	if messageQueueType == "" {
		messageQueueType = "kafka"
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	var srv *server.Server
	var err error

	switch messageQueueType {
	case "kafka":
		brokers := os.Getenv("KAFKA_BROKERS")
		if brokers == "" {
			brokers = "localhost:9092"
		}
		srv, err = server.NewServer(
			server.WithKafka(brokers, "sensor-data"),
			server.WithMongoDB(mongoURI, "sensor_db"),
			server.WithWorkerConfig(8, 200),
			server.WithPort("8080"),
		)
	}

	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received...")
		cancel()
	}()

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	srv.Close()
	log.Println("Server shutdown complete")
}
