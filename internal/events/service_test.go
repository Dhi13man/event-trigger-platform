package events

import (
	"context"
	"testing"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/testutil/fakes"
	"github.com/dhima/event-trigger-platform/pkg/clock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestFireTrigger_Success(t *testing.T) {
	store := fakes.NewFakeEventLogStore()
	pub := &fakes.FakePublisher{}
	svc := NewServiceWithClock(store, pub, newTestZap(t), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC)))

	tr := &models.Trigger{ID: uuid.New().String(), Type: models.TriggerTypeCronScheduled}
	payload := map[string]any{"k": "v"}
	id, err := svc.FireTrigger(context.Background(), tr, models.EventSourceScheduler, payload, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	// Published once
	assert.Len(t, pub.Events, 1)

	// Stored event has success
	ev, _ := store.GetEventLog(context.Background(), id)
	if assert.NotNil(t, ev) {
		assert.Equal(t, models.ExecutionStatusSuccess, ev.ExecutionStatus)
		assert.Equal(t, models.EventSourceScheduler, ev.Source)
		assert.False(t, ev.IsTestRun)
	}
}

func TestFireTrigger_PublishFailMarksFailure(t *testing.T) {
	store := fakes.NewFakeEventLogStore()
	pub := &fakes.FakePublisher{FailNext: true}
	svc := NewServiceWithClock(store, pub, newTestZap(t), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC)))

	tr := &models.Trigger{ID: uuid.New().String(), Type: models.TriggerTypeCronScheduled}
	id, err := svc.FireTrigger(context.Background(), tr, models.EventSourceScheduler, nil, true)
	assert.Error(t, err)
	assert.NotEmpty(t, id)

	ev, _ := store.GetEventLog(context.Background(), id)
	if assert.NotNil(t, ev) {
		assert.Equal(t, models.ExecutionStatusFailure, ev.ExecutionStatus)
		assert.True(t, ev.IsTestRun)
	}
}

func TestQueryEvents_WithFilters(t *testing.T) {
	// Arrange
	store := fakes.NewFakeEventLogStore()
	pub := &fakes.FakePublisher{}
	svc := NewServiceWithClock(store, pub, newTestZap(t), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC)))

	tr1ID := uuid.New().String()
	tr2ID := uuid.New().String()

	// Create multiple events with different attributes
	ev1ID, _ := svc.FireTrigger(context.Background(), &models.Trigger{ID: tr1ID, Type: models.TriggerTypeWebhook}, models.EventSourceWebhook, map[string]any{"k": "v1"}, false)
	ev2ID, _ := svc.FireTrigger(context.Background(), &models.Trigger{ID: tr2ID, Type: models.TriggerTypeCronScheduled}, models.EventSourceScheduler, map[string]any{"k": "v2"}, true)

	tests := []struct {
		name          string
		query         models.ListEventsQuery
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "filter by trigger_id",
			query:         models.ListEventsQuery{TriggerID: tr1ID, Page: 1, Limit: 10},
			expectedCount: 1,
			expectedIDs:   []string{ev1ID},
		},
		{
			name:          "filter by source",
			query:         models.ListEventsQuery{Source: string(models.EventSourceScheduler), Page: 1, Limit: 10},
			expectedCount: 1,
			expectedIDs:   []string{ev2ID},
		},
		{
			name:          "filter by retention_status",
			query:         models.ListEventsQuery{RetentionStatus: string(models.RetentionStatusActive), Page: 1, Limit: 10},
			expectedCount: 2,
			expectedIDs:   []string{ev1ID, ev2ID},
		},
		{
			name:          "filter by execution_status",
			query:         models.ListEventsQuery{ExecutionStatus: string(models.ExecutionStatusSuccess), Page: 1, Limit: 10},
			expectedCount: 2,
			expectedIDs:   []string{ev1ID, ev2ID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			events, pagination, err := svc.QueryEvents(context.Background(), tt.query)

			// Assert
			assert.NoError(t, err)
			assert.Len(t, events, tt.expectedCount)
			assert.Equal(t, int64(tt.expectedCount), pagination.TotalRecords)
		})
	}
}

func TestQueryEvents_Pagination(t *testing.T) {
	// Arrange
	store := fakes.NewFakeEventLogStore()
	pub := &fakes.FakePublisher{}
	svc := NewServiceWithClock(store, pub, newTestZap(t), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC)))

	// Create 5 events
	for i := 0; i < 5; i++ {
		_, _ = svc.FireTrigger(context.Background(), &models.Trigger{ID: uuid.New().String(), Type: models.TriggerTypeWebhook}, models.EventSourceWebhook, map[string]any{"k": i}, false)
	}

	// Act
	events, pagination, err := svc.QueryEvents(context.Background(), models.ListEventsQuery{
		Page:  1,
		Limit: 2,
	})

	// Assert
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, 1, pagination.CurrentPage)
	assert.Equal(t, 2, pagination.PageSize)
	assert.Equal(t, int64(5), pagination.TotalRecords)
	assert.Equal(t, 3, pagination.TotalPages)
}

func TestGetEvent_Success(t *testing.T) {
	// Arrange
	store := fakes.NewFakeEventLogStore()
	pub := &fakes.FakePublisher{}
	svc := NewServiceWithClock(store, pub, newTestZap(t), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC)))

	tr := &models.Trigger{ID: uuid.New().String(), Type: models.TriggerTypeWebhook}
	eventID, _ := svc.FireTrigger(context.Background(), tr, models.EventSourceWebhook, map[string]any{"k": "v"}, false)

	// Act
	event, err := svc.GetEvent(context.Background(), eventID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, eventID, event.ID)
	assert.Equal(t, models.EventSourceWebhook, event.Source)
}

func TestGetEvent_NotFound(t *testing.T) {
	// Arrange
	store := fakes.NewFakeEventLogStore()
	pub := &fakes.FakePublisher{}
	svc := NewServiceWithClock(store, pub, newTestZap(t), clock.NewFixed(time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC)))

	// Act
	event, err := svc.GetEvent(context.Background(), "nonexistent")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, event)
}

func newTestZap(t *testing.T) *zap.Logger { return zap.NewNop() }
