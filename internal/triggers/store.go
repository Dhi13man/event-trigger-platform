package triggers

import (
	"context"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
)

// TriggerStore defines the storage methods required by the trigger service.
type TriggerStore interface {
	CreateTrigger(ctx context.Context, trigger *models.Trigger, schedule *models.TriggerSchedule) error
	GetTrigger(ctx context.Context, triggerID string) (*models.Trigger, *time.Time, error)
	ListTriggers(ctx context.Context, query models.ListTriggersQuery) ([]models.Trigger, []*time.Time, int64, error)
	UpdateTrigger(ctx context.Context, triggerID string, updates map[string]interface{}) error
	DeleteTrigger(ctx context.Context, triggerID string) error
	UpsertTriggerSchedule(ctx context.Context, triggerID string, schedule *models.TriggerSchedule) error
}
