package broker

import "context"

type MessageQueue interface {
	Publish(ctx context.Context, data []byte) error
	Subscribe(ctx context.Context, handler func([]byte)error) error
	Close() error
}