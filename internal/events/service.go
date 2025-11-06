package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/dhima/event-trigger-platform/platform/events"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service provides business logic for event handling and firing triggers.
type Service struct {
	db        *storage.MySQLClient
	publisher *events.Publisher
	logger    *zap.Logger
}

// NewService creates a new EventService instance.
func NewService(db *storage.MySQLClient, publisher *events.Publisher, logger *zap.Logger) *Service {
	return &Service{
		db:        db,
		publisher: publisher,
		logger:    logger,
	}
}

// FireTrigger creates an event log entry and publishes the trigger event to Kafka.
// This method implements at-least-once semantics:
// 1. Write event_log to database (inside transaction)
// 2. Publish to Kafka (outside transaction)
// 3. If Kafka fails, update event_log status to 'failure'
func (s *Service) FireTrigger(ctx context.Context, trigger *models.Trigger, source models.EventSource, payload map[string]interface{}, isTestRun bool) (string, error) {
	// Generate unique event ID
	eventID := uuid.New().String()

	// Prepare payload JSON
	var payloadBytes json.RawMessage
	if payload != nil {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			s.logger.Error("failed to marshal payload",
				zap.String("trigger_id", trigger.ID),
				zap.Error(err))
			return "", fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	// Create event log entry with 'success' status initially
	eventLog := &models.EventLog{
		ID:              eventID,
		TriggerID:       &trigger.ID,
		TriggerType:     trigger.Type,
		FiredAt:         time.Now().UTC(),
		Payload:         payloadBytes,
		Source:          source,
		ExecutionStatus: models.ExecutionStatusSuccess,
		RetentionStatus: models.RetentionStatusActive,
		IsTestRun:       isTestRun,
		CreatedAt:       time.Now().UTC(),
	}

	// For pure manual test runs without persisted trigger, trigger_id can be nil
	if trigger.ID == "" {
		eventLog.TriggerID = nil
	}

	// Insert event log into database
	err := s.db.CreateEventLog(ctx, eventLog)
	if err != nil {
		s.logger.Error("failed to create event log",
			zap.String("event_id", eventID),
			zap.String("trigger_id", trigger.ID),
			zap.Error(err))
		return "", fmt.Errorf("failed to create event log: %w", err)
	}

	s.logger.Info("event log created successfully",
		zap.String("event_id", eventID),
		zap.String("trigger_id", trigger.ID),
		zap.String("source", string(source)),
		zap.Bool("is_test_run", isTestRun))

	// Publish to Kafka (outside transaction for at-least-once semantics)
	triggerEvent := events.TriggerEvent{
		EventID:   eventID,
		TriggerID: trigger.ID,
		Type:      string(trigger.Type),
		Payload:   payload,
		FiredAt:   eventLog.FiredAt,
		Source:    string(source),
	}

	err = s.publisher.Publish(ctx, triggerEvent)
	if err != nil {
		// Kafka publish failed - update event log status to 'failure'
		s.logger.Error("failed to publish event to Kafka, marking as failed",
			zap.String("event_id", eventID),
			zap.String("trigger_id", trigger.ID),
			zap.Error(err))

		errorMsg := fmt.Sprintf("Kafka publish failed: %s", err.Error())
		updateErr := s.db.UpdateEventLogStatus(ctx, eventID, models.ExecutionStatusFailure, &errorMsg)
		if updateErr != nil {
			s.logger.Error("failed to update event log status after Kafka failure",
				zap.String("event_id", eventID),
				zap.Error(updateErr))
		}

		return eventID, fmt.Errorf("failed to publish event to Kafka: %w", err)
	}

	s.logger.Info("trigger fired successfully",
		zap.String("event_id", eventID),
		zap.String("trigger_id", trigger.ID),
		zap.String("type", string(trigger.Type)),
		zap.String("source", string(source)))

	return eventID, nil
}

// QueryEvents retrieves event logs with filtering and pagination.
func (s *Service) QueryEvents(ctx context.Context, query models.ListEventsQuery) ([]models.EventLog, models.Pagination, error) {
	events, totalCount, err := s.db.ListEventLogs(ctx, query)
	if err != nil {
		s.logger.Error("failed to query event logs",
			zap.String("trigger_id", query.TriggerID),
			zap.String("retention_status", query.RetentionStatus),
			zap.Error(err))
		return nil, models.Pagination{}, fmt.Errorf("failed to query event logs: %w", err)
	}

	// Calculate pagination
	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	totalPages := int(totalCount) / limit
	if int(totalCount)%limit != 0 {
		totalPages++
	}

	pagination := models.Pagination{
		CurrentPage:  page,
		PageSize:     limit,
		TotalPages:   totalPages,
		TotalRecords: totalCount,
	}

	s.logger.Info("queried event logs successfully",
		zap.Int("count", len(events)),
		zap.Int64("total", totalCount),
		zap.Int("page", page))

	return events, pagination, nil
}

// GetEvent retrieves a single event log by ID.
func (s *Service) GetEvent(ctx context.Context, eventID string) (*models.EventLog, error) {
	eventLog, err := s.db.GetEventLog(ctx, eventID)
	if err != nil {
		s.logger.Error("failed to get event log",
			zap.String("event_id", eventID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get event log: %w", err)
	}

	if eventLog == nil {
		s.logger.Info("event log not found",
			zap.String("event_id", eventID))
		return nil, nil
	}

	s.logger.Info("retrieved event log successfully",
		zap.String("event_id", eventID))

	return eventLog, nil
}
