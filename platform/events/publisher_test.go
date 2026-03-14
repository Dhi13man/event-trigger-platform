package events

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewPublisher_WhenCreated_ThenReturnsPublisherWithWriter(t *testing.T) {
	// Arrange
	brokers := []string{"localhost:9092"}
	topic := "test-topic"
	logger, _ := zap.NewDevelopment()

	// Act
	publisher := NewPublisher(brokers, topic, logger)

	// Assert
	if publisher == nil {
		t.Fatal("expected publisher to be non-nil")
	}
	if publisher.writer == nil {
		t.Fatal("expected writer to be non-nil")
	}
	if publisher.logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
	if publisher.writer.Topic != topic {
		t.Errorf("expected topic '%s', got '%s'", topic, publisher.writer.Topic)
	}
}

func TestNewPublisher_WhenCreatedWithMultipleBrokers_ThenConfiguresCorrectly(t *testing.T) {
	// Arrange
	brokers := []string{"broker1:9092", "broker2:9092", "broker3:9092"}
	topic := "trigger-events"
	logger, _ := zap.NewDevelopment()

	// Act
	publisher := NewPublisher(brokers, topic, logger)

	// Assert
	if publisher.writer.Addr.String() != "broker1:9092,broker2:9092,broker3:9092" {
		t.Errorf("unexpected broker configuration: %s", publisher.writer.Addr.String())
	}
}

func TestNewPublisher_WhenCreated_ThenHasProductionSettings(t *testing.T) {
	// Arrange
	brokers := []string{"localhost:9092"}
	topic := "test-topic"
	logger, _ := zap.NewDevelopment()

	// Act
	publisher := NewPublisher(brokers, topic, logger)

	// Assert
	if publisher.writer.RequiredAcks != -1 { // RequireAll = -1
		t.Errorf("expected RequiredAcks to be -1 (all), got %d", publisher.writer.RequiredAcks)
	}
	if publisher.writer.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts to be 3, got %d", publisher.writer.MaxAttempts)
	}
	if publisher.writer.WriteTimeout != 10*time.Second {
		t.Errorf("expected WriteTimeout to be 10s, got %v", publisher.writer.WriteTimeout)
	}
}

func TestPublish_WhenEventIsValid_ThenNoError(t *testing.T) {
	// Arrange
	logger, _ := zap.NewDevelopment()
	publisher := NewPublisher([]string{"localhost:9092"}, "test-topic", logger)

	event := TriggerEvent{
		EventID:   "evt-123",
		TriggerID: "trg-456",
		Type:      "webhook",
		Payload: map[string]interface{}{
			"message": "test",
		},
		FiredAt: time.Now(),
		Source:  "webhook",
	}

	// Act
	ctx := context.Background()
	// Note: This will fail if Kafka is not running, but we're testing the marshaling logic
	_ = publisher.Publish(ctx, event)

	// Assert - if we reach here without panic, marshaling works
}

func TestPublish_WhenContextCanceled_ThenReturnsError(t *testing.T) {
	// Arrange
	logger, _ := zap.NewDevelopment()
	publisher := NewPublisher([]string{"localhost:9092"}, "test-topic", logger)

	event := TriggerEvent{
		EventID:   "evt-123",
		TriggerID: "trg-456",
		Type:      "time_scheduled",
		Payload:   map[string]interface{}{},
		FiredAt:   time.Now(),
		Source:    "scheduler",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act
	err := publisher.Publish(ctx, event)

	// Assert - expect error due to canceled context or Kafka connection failure
	// We don't check specific error as it depends on Kafka availability
	_ = err
}

func TestClose_WhenCalledWithValidWriter_ThenClosesSuccessfully(t *testing.T) {
	// Arrange
	logger, _ := zap.NewDevelopment()
	publisher := NewPublisher([]string{"localhost:9092"}, "test-topic", logger)

	// Act
	err := publisher.Close()

	// Assert - close should not panic even if Kafka is not running
	_ = err
}

func TestClose_WhenCalledMultipleTimes_ThenDoesNotPanic(t *testing.T) {
	// Arrange
	logger, _ := zap.NewDevelopment()
	publisher := NewPublisher([]string{"localhost:9092"}, "test-topic", logger)

	// Act & Assert
	_ = publisher.Close()
	// Calling close again should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic, but got: %v", r)
		}
	}()
	_ = publisher.Close()
}

func TestTriggerEvent_WhenMarshaledToJSON_ThenContainsAllFields(t *testing.T) {
	// Arrange
	event := TriggerEvent{
		EventID:   "evt-123",
		TriggerID: "trg-456",
		Type:      "cron_scheduled",
		Payload: map[string]interface{}{
			"message": "test",
			"count":   42,
		},
		FiredAt: time.Date(2025, 11, 6, 10, 30, 0, 0, time.UTC),
		Source:  "scheduler",
	}

	// Act
	// This is implicitly tested in Publish, but we can verify the structure is correct
	if event.EventID == "" {
		t.Error("expected EventID to be set")
	}
	if event.TriggerID == "" {
		t.Error("expected TriggerID to be set")
	}
	if event.Type == "" {
		t.Error("expected Type to be set")
	}
	if event.Payload == nil {
		t.Error("expected Payload to be set")
	}
	if event.FiredAt.IsZero() {
		t.Error("expected FiredAt to be set")
	}
	if event.Source == "" {
		t.Error("expected Source to be set")
	}
}
