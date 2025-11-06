package fakes

import (
	"context"
	"sync"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
)

// FakeEventLogStore is an in-memory EventLogStore.
type FakeEventLogStore struct {
	mu     sync.Mutex
	events map[string]models.EventLog
}

func NewFakeEventLogStore() *FakeEventLogStore {
	return &FakeEventLogStore{events: make(map[string]models.EventLog)}
}

func (f *FakeEventLogStore) CreateEventLog(_ context.Context, e *models.EventLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	ev := *e
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	f.events[ev.ID] = ev
	return nil
}

func (f *FakeEventLogStore) UpdateEventLogStatus(_ context.Context, eventID string, status models.ExecutionStatus, errorMessage *string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	ev := f.events[eventID]
	ev.ExecutionStatus = status
	ev.ErrorMessage = errorMessage
	f.events[eventID] = ev
	return nil
}

func (f *FakeEventLogStore) GetEventLog(_ context.Context, eventID string) (*models.EventLog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ev, ok := f.events[eventID]
	if !ok {
		return nil, ErrNotFound
	}
	cpy := ev
	return &cpy, nil
}

func (f *FakeEventLogStore) ListEventLogs(_ context.Context, q models.ListEventsQuery) ([]models.EventLog, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]models.EventLog, 0)
	for _, ev := range f.events {
		if q.RetentionStatus != "" && string(ev.RetentionStatus) != q.RetentionStatus {
			continue
		}
		if q.TriggerID != "" {
			if ev.TriggerID == nil || *ev.TriggerID != q.TriggerID {
				continue
			}
		}
		if q.ExecutionStatus != "" && string(ev.ExecutionStatus) != q.ExecutionStatus {
			continue
		}
		if q.Source != "" && string(ev.Source) != q.Source {
			continue
		}
		out = append(out, ev)
	}
	total := int64(len(out))
	// naive pagination
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	start := (q.Page - 1) * q.Limit
	if start > len(out) {
		return []models.EventLog{}, total, nil
	}
	end := start + q.Limit
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], total, nil
}
