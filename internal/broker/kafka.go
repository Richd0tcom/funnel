package broker

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaQueue struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
	topic    string
}
type KafkaTopicConfig struct {
	NumPartitions     int
	ReplicationFactor int
	Topic             string
	BootstrapServers  string
}

func NewAdminClient(spec KafkaTopicConfig) {
	a, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": spec.BootstrapServers})
	if err != nil {
		fmt.Printf("Failed to create Admin client: %s\n", err)
		os.Exit(1)
	}

	// Contexts are used to abort or limit the amount of time
	// the Admin call blocks waiting for a result.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create topics on cluster.
	// Set Admin options to wait for the operation to finish (or at most 60s)
	maxDur, err := time.ParseDuration("60s")
	if err != nil {
		panic("ParseDuration(60s)")
	}
	results, err := a.CreateTopics(
		ctx,

		[]kafka.TopicSpecification{{
			Topic: spec.Topic,
			NumPartitions: spec.NumPartitions,
		}},
		// Admin options
		kafka.SetAdminOperationTimeout(maxDur))
	if err != nil {
		// Check if the error is due to topic already existing
		if kErr, ok := err.(kafka.Error); ok && kErr.Code() == kafka.ErrTopicAlreadyExists {
			log.Printf("Topic already exists!!")
			return
		}
		fmt.Printf("Failed to create topic: %v\n", err)
		os.Exit(1)
	}

	// Print results
	for _, result := range results {
		fmt.Printf("%s\n", result)
	}

	a.Close()
}

func NewKafkaQueue(brokers, topic string) (*KafkaQueue, error) {
	NewAdminClient(KafkaTopicConfig{
		Topic: topic,
		BootstrapServers: brokers,
		NumPartitions: 1, //TODO: set from env
	})
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"acks":              "all",
		"retries":           3,
		"batch.size":        16384,
		"linger.ms":         5,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          "sensor-aggregator", //TODO: change this to get from env
		"auto.offset.reset": "latest",
	})
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &KafkaQueue{
		producer: producer,
		consumer: consumer,
		topic:    topic,
	}, nil
}

func (k *KafkaQueue) Publish(ctx context.Context, data []byte) error {
	deliveryChan := make(chan kafka.Event)

	defer close(deliveryChan)

	err := k.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &k.topic,
			Partition: kafka.PartitionAny,
		},
		Value: data,
	}, deliveryChan)

	if err != nil {
		return err
	}

	select {
	case e := <-deliveryChan:
		if msg, ok := e.(*kafka.Message); ok {
			if msg.TopicPartition.Error != nil {
				return msg.TopicPartition.Error
			}
		}

	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (k *KafkaQueue) Subscribe(ctx context.Context, handler func([]byte) error) error {

	err := k.consumer.Subscribe(k.topic, nil)

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := k.consumer.ReadMessage(100 * time.Millisecond)

			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				return err
			}

			if err = handler(msg.Value); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}

}

func (k *KafkaQueue) Close() error {
	k.producer.Close()
	return k.consumer.Close()
}
