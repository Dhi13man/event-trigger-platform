package events

import (
	"context"

	"github.com/dhima/event-trigger-platform/internal/models"
	platformEvents "github.com/dhima/event-trigger-platform/platform/events"
)

// EventLogStore defines persistence required by the Event Service.
type EventLogStore interface {
	CreateEventLog(ctx context.Context, eventLog *models.EventLog) error
	UpdateEventLogStatus(ctx context.Context, eventID string, status models.ExecutionStatus, errorMessage *string) error
	ListEventLogs(ctx context.Context, query models.ListEventsQuery) ([]models.EventLog, int64, error)
	GetEventLog(ctx context.Context, eventID string) (*models.EventLog, error)
}

// EventPublisher abstracts the Kafka publisher for testability.
type EventPublisher interface {
	Publish(ctx context.Context, event platformEvents.TriggerEvent) error
}
