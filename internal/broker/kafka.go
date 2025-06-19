package broker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaQueue struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
	topic    string
}

func NewKafkaQueue(brokers, topic string) (*KafkaQueue, error) {
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

	consumer, err:= kafka.NewConsumer(&kafka.ConfigMap{
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
		topic: topic,
	}, nil
}

func (k *KafkaQueue) Publish(ctx context.Context, data []byte) error {
	deliveryChan := make(chan kafka.Event)

	defer close(deliveryChan)

	err:= k.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic: &k.topic,
			Partition: kafka.PartitionAny,
		},
		Value: data,
	}, deliveryChan)

	if err != nil {
		return err
	}

	select {
	case e:= <-deliveryChan:
		if msg, ok:= e.(*kafka.Message); ok {
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
	
	err:= k.consumer.Subscribe(k.topic, nil)

	if err != nil {
		return err
	}
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err:= k.consumer.ReadMessage(100 * time.Millisecond)

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