package ports

import (
	"context"
)

type KafkaProducer interface {
	PublishMessage(ctx context.Context, topic string, key string, value []byte, headers map[string]string) error
	Close() error
}
