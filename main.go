package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/richd0tcom/funnel/core/server"
	"github.com/richd0tcom/funnel/internal/db"
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

	client, err:= db.NewMongoConnection(mongoURI)
	

	switch messageQueueType {
	case "kafka":
		brokers := os.Getenv("KAFKA_BROKERS")
		fmt.Println("-----------KAFKA BROKER FROM ENV")
		if brokers == "" {
			brokers = "localhost:9092"
		}

		srv, err = server.NewServer(
			server.WithKafka(brokers, "sensor-data"),
			server.WithMongoDB(client, "sensor_db"),
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

	defer client.Disconnect(ctx)

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	srv.Close()
	log.Println("Server shutdown complete")
}
