package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/dhima/event-trigger-platform/internal/testutil/fakes"
	"github.com/dhima/event-trigger-platform/pkg/clock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestProcessSchedule_TimeScheduled_SuccessDeactivates(t *testing.T) {
	store := fakes.NewFakeScheduleStore()
	trig := models.Trigger{ID: uuid.New().String(), Name: "one", Type: models.TriggerTypeTimeScheduled, Status: models.TriggerStatusActive, Config: []byte(`{"endpoint":"https://e","http_method":"POST"}`)}
	store.Triggers[trig.ID] = trig
	sch := models.TriggerSchedule{ID: uuid.New().String(), TriggerID: trig.ID, FireAt: time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC), Status: models.ScheduleStatusPending}
	store.Schedules[sch.ID] = sch

	eng := NewEngineWithClock(1*time.Second, store, &fakes.FakeEventFirer{Fail: false}, zap.NewNop(), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 1, 0, time.UTC)))
	err := eng.processSchedule(context.Background(), storage.ScheduleWithTrigger{Schedule: sch, Trigger: trig})
	assert.NoError(t, err)

	// schedule completed
	got := store.Schedules[sch.ID]
	assert.Equal(t, models.ScheduleStatusCompleted, got.Status)
	// trigger deactivated
	assert.Equal(t, models.TriggerStatusInactive, store.Triggers[trig.ID].Status)
}

func TestProcessSchedule_Cron_SuccessCreatesNext(t *testing.T) {
	store := fakes.NewFakeScheduleStore()
	trig := models.Trigger{ID: uuid.New().String(), Name: "cron", Type: models.TriggerTypeCronScheduled, Status: models.TriggerStatusActive, Config: []byte(`{"cron":"*/5 * * * *","timezone":"UTC"}`)}
	store.Triggers[trig.ID] = trig
	sch := models.TriggerSchedule{ID: uuid.New().String(), TriggerID: trig.ID, FireAt: time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC), Status: models.ScheduleStatusPending}
	store.Schedules[sch.ID] = sch

	eng := NewEngineWithClock(1*time.Second, store, &fakes.FakeEventFirer{Fail: false}, zap.NewNop(), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 1, 0, time.UTC)))
	err := eng.processSchedule(context.Background(), storage.ScheduleWithTrigger{Schedule: sch, Trigger: trig})
	assert.NoError(t, err)

	// Next schedule exists (count > 1)
	foundNext := false
	for id, s := range store.Schedules {
		if id != sch.ID && s.TriggerID == trig.ID {
			foundNext = true
		}
	}
	assert.True(t, foundNext, "next schedule not created")
}

func TestProcessSchedule_Failure_RetriesAndReverts(t *testing.T) {
	store := fakes.NewFakeScheduleStore()
	trig := models.Trigger{ID: uuid.New().String(), Name: "cron", Type: models.TriggerTypeCronScheduled, Status: models.TriggerStatusActive, Config: []byte(`{"cron":"*/5 * * * *","timezone":"UTC"}`)}
	store.Triggers[trig.ID] = trig
	sch := models.TriggerSchedule{ID: uuid.New().String(), TriggerID: trig.ID, FireAt: time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC), Status: models.ScheduleStatusPending}
	store.Schedules[sch.ID] = sch

	eng := NewEngineWithClock(1*time.Second, store, &fakes.FakeEventFirer{Fail: true}, zap.NewNop(), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 1, 0, time.UTC)))
	err := eng.processSchedule(context.Background(), storage.ScheduleWithTrigger{Schedule: sch, Trigger: trig})
	assert.Error(t, err)

	got := store.Schedules[sch.ID]
	assert.Equal(t, models.ScheduleStatusPending, got.Status)
	assert.Equal(t, 1, got.AttemptCount)
}

func TestProcessSchedule_MaxRetriesReached(t *testing.T) {
	// Arrange
	store := fakes.NewFakeScheduleStore()
	trig := models.Trigger{ID: uuid.New().String(), Name: "retry", Type: models.TriggerTypeCronScheduled, Status: models.TriggerStatusActive, Config: []byte(`{"cron":"*/5 * * * *","timezone":"UTC","endpoint":"https://e","http_method":"POST"}`)}
	store.Triggers[trig.ID] = trig
	sch := models.TriggerSchedule{
		ID:           uuid.New().String(),
		TriggerID:    trig.ID,
		FireAt:       time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC),
		Status:       models.ScheduleStatusPending,
		AttemptCount: 4, // Already at 4, next attempt will be 5 (max)
	}
	store.Schedules[sch.ID] = sch

	eng := NewEngineWithClock(1*time.Second, store, &fakes.FakeEventFirer{Fail: true}, zap.NewNop(), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 1, 0, time.UTC)))

	// Act
	err := eng.processSchedule(context.Background(), storage.ScheduleWithTrigger{Schedule: sch, Trigger: trig})

	// Assert
	assert.Error(t, err)
	got := store.Schedules[sch.ID]
	// AttemptCount stays at 4 because RevertScheduleToPending is only called for retries
	// When max retries are reached, we go directly to cancelled without incrementing
	assert.Equal(t, 4, got.AttemptCount)
	// At max retries, schedule is cancelled instead of pending
	assert.Equal(t, models.ScheduleStatusCancelled, got.Status)
}

func TestProcessSchedule_CancelledTrigger(t *testing.T) {
	// Arrange
	store := fakes.NewFakeScheduleStore()
	trig := models.Trigger{ID: uuid.New().String(), Name: "cancelled", Type: models.TriggerTypeCronScheduled, Status: models.TriggerStatusInactive, Config: []byte(`{"cron":"*/5 * * * *","timezone":"UTC","endpoint":"https://e","http_method":"POST"}`)}
	store.Triggers[trig.ID] = trig
	sch := models.TriggerSchedule{ID: uuid.New().String(), TriggerID: trig.ID, FireAt: time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC), Status: models.ScheduleStatusPending}
	store.Schedules[sch.ID] = sch

	eng := NewEngineWithClock(1*time.Second, store, &fakes.FakeEventFirer{Fail: false}, zap.NewNop(), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 1, 0, time.UTC)))

	// Act
	err := eng.processSchedule(context.Background(), storage.ScheduleWithTrigger{Schedule: sch, Trigger: trig})

	// Assert
	assert.NoError(t, err)
	got := store.Schedules[sch.ID]
	assert.Equal(t, models.ScheduleStatusCompleted, got.Status)
	// Should NOT create next schedule for inactive trigger
	foundNext := false
	for id, s := range store.Schedules {
		if id != sch.ID && s.TriggerID == trig.ID && s.Status == models.ScheduleStatusPending {
			foundNext = true
		}
	}
	assert.False(t, foundNext, "next schedule should not be created for inactive trigger")
}

func TestProcessSchedules_BatchProcessing(t *testing.T) {
	// Arrange
	store := fakes.NewFakeScheduleStore()
	firer := &fakes.FakeEventFirer{Fail: false}
	eng := NewEngineWithClock(1*time.Second, store, firer, zap.NewNop(), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 1, 0, time.UTC)))

	// Create 3 due schedules
	for i := 0; i < 3; i++ {
		trigID := uuid.New().String()
		trig := models.Trigger{ID: trigID, Name: "batch", Type: models.TriggerTypeTimeScheduled, Status: models.TriggerStatusActive, Config: []byte(`{"endpoint":"https://e","http_method":"POST"}`)}
		store.Triggers[trigID] = trig
		sch := models.TriggerSchedule{ID: uuid.New().String(), TriggerID: trigID, FireAt: time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC), Status: models.ScheduleStatusPending}
		store.Schedules[sch.ID] = sch
	}

	// Act
	eng.processSchedules(context.Background())

	// Assert
	// All 3 schedules should be completed
	completedCount := 0
	for _, s := range store.Schedules {
		if s.Status == models.ScheduleStatusCompleted {
			completedCount++
		}
	}
	assert.Equal(t, 3, completedCount)
}

