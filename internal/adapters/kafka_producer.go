package adapters

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
	"time"
	"validator-function/internal/logger"
)

type ConfluentKafkaProducer struct {
	producer   *kafka.Producer
	maxRetries int
	retryDelay time.Duration
}

func NewConfluentKafkaProducer(config *kafka.ConfigMap, maxRetries int, retryDelay time.Duration) (*ConfluentKafkaProducer, error) {
	logger.Info("Creating Kafka producer",
		zap.Any("config", config),
		zap.Int("maxRetries", maxRetries),
		zap.Duration("retryDelay", retryDelay))

	producer, err := kafka.NewProducer(config)
	if err != nil {
		logger.Error("Failed to create Kafka producer", zap.Error(err))
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	// Handle events in a goroutine
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logger.Error("Message delivery failed",
						zap.Error(ev.TopicPartition.Error),
						zap.String("topic", *ev.TopicPartition.Topic))
				}
			case *kafka.Error:
				logger.Error("Kafka error",
					zap.String("code", ev.Code().String()),
					zap.Error(ev))
			default:
				logger.Info("Kafka event",
					zap.String("type", fmt.Sprintf("%T", e)),
					zap.Any("event", e))
			}
		}
	}()

	logger.Info("Kafka producer created successfully")
	return &ConfluentKafkaProducer{
		producer:   producer,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}, nil
}

func (p *ConfluentKafkaProducer) PublishMessage(ctx context.Context, topic string, key string, value []byte, headers map[string]string) error {
	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying message publication",
				zap.Int("attempt", attempt),
				zap.Int("maxRetries", p.maxRetries),
				zap.Error(lastErr))
			time.Sleep(p.retryDelay * time.Duration(attempt))
		}

		if err := p.publishWithTimeout(ctx, topic, key, value, headers); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to publish message after %d retries: %w", p.maxRetries, lastErr)
}

func (p *ConfluentKafkaProducer) publishWithTimeout(ctx context.Context, topic string, key string, value []byte, headers map[string]string) error {
	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          value,
		Headers:        kafkaHeaders,
	}

	if err := p.producer.Produce(message, deliveryChan); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("message delivery timed out")
	}
}

func (p *ConfluentKafkaProducer) Close() error {
	p.producer.Close()
	return nil
}

func (p *ConfluentKafkaProducer) GetUnderlyingProducer() (*kafka.Producer, error) {
	if p.producer == nil {
		return nil, fmt.Errorf("kafka producer not initialized")
	}
	return p.producer, nil
}
