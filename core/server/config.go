package server

import (
	"github.com/richd0tcom/funnel/internal/broker"
	"github.com/richd0tcom/funnel/internal/db"
	"github.com/richd0tcom/funnel/internal/domain"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ServerConfig struct {
	MessageQueue broker.MessageQueue
	DataStore    domain.DataStore
	Consumer     domain.DataConsumer
	WorkerCount  int
	BatchSize    int
	Port         string
}

type ConfigOption func(*ServerConfig) error

func WithKafka(brokers, topic string) ConfigOption {
	return func(config *ServerConfig) error {
		mq, err := broker.NewKafkaQueue(brokers, topic)
		if err != nil {
			return err
		}
		config.MessageQueue = mq
		return nil
	}
}


func WithMongoDB(client *mongo.Client, database string) ConfigOption {
	
	return func(config *ServerConfig) error {
		store, err := db.NewMongoTimeSeriesStore(client, database)
		if err != nil {
			return err
		}
		config.DataStore = store
		return nil
	}
}

func WithWorkerConfig(workerCount, batchSize int) ConfigOption {
	return func(config *ServerConfig) error {
		config.WorkerCount = workerCount
		config.BatchSize = batchSize
		return nil
	}
}

func WithPort(port string) ConfigOption {
	return func(config *ServerConfig) error {
		config.Port = port
		return nil
	}
}