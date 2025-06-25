package broker

import "context"

type MessageQueue interface {
	Publish(ctx context.Context, data []byte) error
	Consume(ctx context.Context, handler func([]byte)error) error
	Subscribe() error
	Close() error
}