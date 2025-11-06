package triggers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/testutil/fakes"
	"github.com/dhima/event-trigger-platform/pkg/clock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func fixed() time.Time { return time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC) }

func TestCreateTrigger_Webhook_NormalizesMethod(t *testing.T) {
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	req := models.CreateTriggerRequest{
		Name: "wh",
		Type: models.TriggerTypeWebhook,
		Config: mustJSON(map[string]any{
			"endpoint": "https://example.com/hook",
			// http_method omitted -> should default to POST
		}),
	}

	resp, err := svc.CreateTrigger(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, models.TriggerTypeWebhook, resp.Type)

	var cfg map[string]any
	_ = json.Unmarshal(resp.Config, &cfg)
	assert.Equal(t, "POST", cfg["http_method"]) // defaulted
}

func TestCreateTrigger_TimeScheduled_Validation(t *testing.T) {
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	// Missing endpoint
	req := models.CreateTriggerRequest{
		Name: "ts",
		Type: models.TriggerTypeTimeScheduled,
		Config: mustJSON(map[string]any{
			"run_at":   "2025-01-02T03:05:00Z",
			"timezone": "UTC",
		}),
	}
	_, err := svc.CreateTrigger(context.Background(), req)
	var vErr ValidationError
	assert.True(t, errors.As(err, &vErr))

	// Past run_at
	req2 := models.CreateTriggerRequest{
		Name: "ts2",
		Type: models.TriggerTypeTimeScheduled,
		Config: mustJSON(map[string]any{
			"run_at":      "2025-01-02T02:58:00Z", // More than 1 min before fixed() → invalid
			"endpoint":    "https://example.com",
			"http_method": "POST",
			"timezone":    "UTC",
		}),
	}
	_, err = svc.CreateTrigger(context.Background(), req2)
	assert.True(t, errors.As(err, &vErr))
}

func TestCreateTrigger_TimeScheduled_SchedulesNext(t *testing.T) {
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	runAt := fixed().Add(2 * time.Minute).Format(time.RFC3339)
	req := models.CreateTriggerRequest{
		Name: "ts",
		Type: models.TriggerTypeTimeScheduled,
		Config: mustJSON(map[string]any{
			"run_at":      runAt,
			"endpoint":    "https://example.com",
			"http_method": "POST",
			"timezone":    "UTC",
		}),
	}
	resp, err := svc.CreateTrigger(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp.NextScheduledRun)
	assert.Equal(t, runAt, resp.NextScheduledRun.UTC().Format(time.RFC3339))
}

func TestCreateTrigger_Cron_SchedulesNext(t *testing.T) {
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	req := models.CreateTriggerRequest{
		Name: "cron",
		Type: models.TriggerTypeCronScheduled,
		Config: mustJSON(map[string]any{
			"cron":        "*/5 * * * *",
			"endpoint":    "https://example.com",
			"http_method": "POST",
			"timezone":    "UTC",
		}),
	}
	resp, err := svc.CreateTrigger(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp.NextScheduledRun)
	// fixed=03:00 → next 03:05
	assert.Equal(t, time.Date(2025, 1, 2, 3, 5, 0, 0, time.UTC), *resp.NextScheduledRun)
}

func TestUpdateTrigger_ConfigChangeReschedules(t *testing.T) {
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	// create cron
	trID := uuid.New().String()
	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     trID,
		Name:   "c",
		Type:   models.TriggerTypeCronScheduled,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"cron": "*/5 * * * *", "endpoint": "https://e"}),
	}, &models.TriggerSchedule{ID: uuid.New().String(), TriggerID: trID, FireAt: fixed().Add(5 * time.Minute), Status: models.ScheduleStatusPending})

	// update cron to */10
	newCfg := mustJSON(map[string]any{"cron": "*/10 * * * *", "endpoint": "https://e"})
	resp, err := svc.UpdateTrigger(context.Background(), trID, models.UpdateTriggerRequest{Config: newCfg})
	assert.NoError(t, err)
	assert.NotNil(t, resp.NextScheduledRun)
}

func TestDeleteTrigger_Success(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))
	trID := uuid.New().String()
	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     trID,
		Name:   "test",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	// Act
	err := svc.DeleteTrigger(context.Background(), trID)

	// Assert
	assert.NoError(t, err)
	_, _, getErr := store.GetTrigger(context.Background(), trID)
	assert.Error(t, getErr) // Should be not found
}

func TestDeleteTrigger_NotFound(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	// Act
	err := svc.DeleteTrigger(context.Background(), "nonexistent")

	// Assert
	assert.Error(t, err)
}

func TestGetTrigger_Success(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))
	trID := uuid.New().String()
	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     trID,
		Name:   "test",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	// Act
	resp, err := svc.GetTrigger(context.Background(), trID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, trID, resp.ID)
	assert.Equal(t, "test", resp.Name)
}

func TestGetTrigger_NotFound(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	// Act
	_, err := svc.GetTrigger(context.Background(), "nonexistent")

	// Assert
	assert.Error(t, err)
}

func TestListTriggers_Pagination(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	// Create 5 triggers
	for i := 0; i < 5; i++ {
		_ = store.CreateTrigger(context.Background(), &models.Trigger{
			ID:     uuid.New().String(),
			Name:   "test",
			Type:   models.TriggerTypeWebhook,
			Status: models.TriggerStatusActive,
			Config: mustJSON(map[string]any{"endpoint": "https://e"}),
		}, nil)
	}

	// Act - Page 1, limit 2
	resp1, err1 := svc.ListTriggers(context.Background(), models.ListTriggersQuery{Page: 1, Limit: 2})

	// Assert
	assert.NoError(t, err1)
	assert.Len(t, resp1.Triggers, 2)
	assert.Equal(t, 1, resp1.Pagination.CurrentPage)
	assert.Equal(t, 2, resp1.Pagination.PageSize)
	assert.Equal(t, int64(5), resp1.Pagination.TotalRecords)
	assert.Equal(t, 3, resp1.Pagination.TotalPages)
}

func TestListTriggers_FilterByType(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     uuid.New().String(),
		Name:   "webhook1",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     uuid.New().String(),
		Name:   "cron1",
		Type:   models.TriggerTypeCronScheduled,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"cron": "*/5 * * * *", "endpoint": "https://e"}),
	}, nil)

	// Act
	resp, err := svc.ListTriggers(context.Background(), models.ListTriggersQuery{
		Type:  string(models.TriggerTypeWebhook),
		Page:  1,
		Limit: 10,
	})

	// Assert
	assert.NoError(t, err)
	assert.Len(t, resp.Triggers, 1)
	assert.Equal(t, models.TriggerTypeWebhook, resp.Triggers[0].Type)
}

func TestListTriggers_FilterByStatus(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     uuid.New().String(),
		Name:   "active",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	trID := uuid.New().String()
	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     trID,
		Name:   "inactive",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusInactive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	// Act
	resp, err := svc.ListTriggers(context.Background(), models.ListTriggersQuery{
		Status: string(models.TriggerStatusInactive),
		Page:   1,
		Limit:  10,
	})

	// Assert
	assert.NoError(t, err)
	assert.Len(t, resp.Triggers, 1)
	assert.Equal(t, models.TriggerStatusInactive, resp.Triggers[0].Status)
}

func TestUpdateTrigger_NameOnly(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))
	trID := uuid.New().String()
	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     trID,
		Name:   "old-name",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	// Act
	newName := "new-name"
	resp, err := svc.UpdateTrigger(context.Background(), trID, models.UpdateTriggerRequest{
		Name: &newName,
	})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "new-name", resp.Name)
}

func TestUpdateTrigger_EmptyNameValidation(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))
	trID := uuid.New().String()
	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     trID,
		Name:   "old-name",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	// Act
	emptyName := "   "
	_, err := svc.UpdateTrigger(context.Background(), trID, models.UpdateTriggerRequest{
		Name: &emptyName,
	})

	// Assert
	var vErr ValidationError
	assert.True(t, errors.As(err, &vErr))
}

func TestUpdateTrigger_StatusChange(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))
	trID := uuid.New().String()
	_ = store.CreateTrigger(context.Background(), &models.Trigger{
		ID:     trID,
		Name:   "test",
		Type:   models.TriggerTypeWebhook,
		Status: models.TriggerStatusActive,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	}, nil)

	// Act
	newStatus := models.TriggerStatusInactive
	resp, err := svc.UpdateTrigger(context.Background(), trID, models.UpdateTriggerRequest{
		Status: &newStatus,
	})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, models.TriggerStatusInactive, resp.Status)
}

func TestCreateTrigger_InvalidType(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	// Act
	_, err := svc.CreateTrigger(context.Background(), models.CreateTriggerRequest{
		Name:   "test",
		Type:   "invalid_type",
		Config: mustJSON(map[string]any{}),
	})

	// Assert
	var vErr ValidationError
	assert.True(t, errors.As(err, &vErr))
}

func TestCreateTrigger_EmptyName(t *testing.T) {
	// Arrange
	store := fakes.NewFakeTriggerStore()
	svc := NewServiceWithClock(store, clock.NewFixed(fixed()))

	// Act
	_, err := svc.CreateTrigger(context.Background(), models.CreateTriggerRequest{
		Name:   "   ",
		Type:   models.TriggerTypeWebhook,
		Config: mustJSON(map[string]any{"endpoint": "https://e"}),
	})

	// Assert
	var vErr ValidationError
	assert.True(t, errors.As(err, &vErr))
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
