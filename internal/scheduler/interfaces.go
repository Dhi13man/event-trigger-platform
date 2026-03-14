package scheduler

import (
	"context"

	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/storage"
)

// ScheduleStore defines DB operations required by the scheduler engine.
type ScheduleStore interface {
	GetDueSchedules(ctx context.Context, limit int) ([]storage.ScheduleWithTrigger, error)
	UpdateScheduleStatus(ctx context.Context, scheduleID string, status models.ScheduleStatus) error
	RevertScheduleToPending(ctx context.Context, scheduleID string) error
	CreateNextSchedule(ctx context.Context, schedule *models.TriggerSchedule) error
	DeactivateTrigger(ctx context.Context, triggerID string) error
}

// EventFirer abstracts event firing from the scheduler.
type EventFirer interface {
	FireTrigger(ctx context.Context, trigger *models.Trigger, source models.EventSource, payload map[string]interface{}, isTestRun bool) (string, error)
}
