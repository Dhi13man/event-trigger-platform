package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// TriggerEvent represents the event structure published to Kafka.
type TriggerEvent struct {
	EventID   string                 `json:"event_id"`
	TriggerID string                 `json:"trigger_id"`
	Type      string                 `json:"type"` // webhook, time_scheduled, cron_scheduled
	Payload   map[string]interface{} `json:"payload"`
	FiredAt   time.Time              `json:"fired_at"`
	Source    string                 `json:"source"` // webhook, scheduler, manual-test
}

// Publisher emits trigger execution jobs to Kafka.
type Publisher struct {
	writer *kafka.Writer
	logger *zap.Logger
}

// NewPublisher creates a Kafka publisher with production-ready configuration.
func NewPublisher(brokers []string, topic string, logger *zap.Logger) *Publisher {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		// Production settings for durability
		RequiredAcks: kafka.RequireAll, // acks=all - wait for all in-sync replicas
		Compression:  kafka.Snappy,     // Compression for network efficiency
		MaxAttempts:  3,                // Retry up to 3 times
		BatchSize:    1,                // Low latency - publish immediately
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	return &Publisher{
		writer: writer,
		logger: logger,
	}
}

// Publish sends a trigger event to the Kafka topic.
// This method is idempotent - the same event_id can be published multiple times.
// Consumer-level deduplication is the responsibility of external consumers.
func (p *Publisher) Publish(ctx context.Context, event TriggerEvent) error {
	// Serialize event to JSON
	messageBytes, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("failed to marshal trigger event",
			zap.String("event_id", event.EventID),
			zap.Error(err))
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message
	msg := kafka.Message{
		Key:   []byte(event.TriggerID), // Key by trigger_id for partition ordering
		Value: messageBytes,
		Time:  time.Now(),
	}

	// Publish with context timeout
	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = p.writer.WriteMessages(publishCtx, msg)
	if err != nil {
		p.logger.Error("failed to publish trigger event to Kafka",
			zap.String("event_id", event.EventID),
			zap.String("trigger_id", event.TriggerID),
			zap.String("topic", p.writer.Topic),
			zap.Error(err))
		return fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	p.logger.Info("trigger event published to Kafka",
		zap.String("event_id", event.EventID),
		zap.String("trigger_id", event.TriggerID),
		zap.String("type", event.Type),
		zap.String("source", event.Source),
		zap.Time("fired_at", event.FiredAt))

	return nil
}

// Close gracefully shuts down the Kafka writer.
func (p *Publisher) Close() error {
	if p.writer != nil {
		err := p.writer.Close()
		if err != nil {
			p.logger.Error("failed to close Kafka writer", zap.Error(err))
			return fmt.Errorf("failed to close Kafka writer: %w", err)
		}
		p.logger.Info("Kafka publisher closed successfully")
	}
	return nil
}
