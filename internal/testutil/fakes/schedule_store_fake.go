package fakes

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/storage"
)

// FakeScheduleStore is an in-memory scheduler store.
type FakeScheduleStore struct {
	mu        sync.Mutex
	Triggers  map[string]models.Trigger
	Schedules map[string]models.TriggerSchedule // by scheduleID
}

func NewFakeScheduleStore() *FakeScheduleStore {
	return &FakeScheduleStore{
		Triggers:  make(map[string]models.Trigger),
		Schedules: make(map[string]models.TriggerSchedule),
	}
}

func (f *FakeScheduleStore) GetDueSchedules(_ context.Context, limit int) ([]storage.ScheduleWithTrigger, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	out := make([]storage.ScheduleWithTrigger, 0)
	for _, s := range f.Schedules {
		if s.Status == models.ScheduleStatusPending && !s.FireAt.After(now) {
			t := f.Triggers[s.TriggerID]
			out = append(out, storage.ScheduleWithTrigger{Schedule: s, Trigger: t})
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (f *FakeScheduleStore) UpdateScheduleStatus(_ context.Context, scheduleID string, status models.ScheduleStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.Schedules[scheduleID]
	if !ok {
		return errors.New("schedule not found")
	}
	s.Status = status
	s.UpdatedAt = time.Now().UTC()
	f.Schedules[scheduleID] = s
	return nil
}

func (f *FakeScheduleStore) RevertScheduleToPending(_ context.Context, scheduleID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.Schedules[scheduleID]
	if !ok {
		return errors.New("schedule not found")
	}
	s.Status = models.ScheduleStatusPending
	s.AttemptCount++
	now := time.Now().UTC()
	s.LastAttemptAt = &now
	s.UpdatedAt = now
	f.Schedules[scheduleID] = s
	return nil
}

func (f *FakeScheduleStore) CreateNextSchedule(_ context.Context, schedule *models.TriggerSchedule) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := *schedule
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	s.UpdatedAt = s.CreatedAt
	f.Schedules[s.ID] = s
	return nil
}

func (f *FakeScheduleStore) DeactivateTrigger(_ context.Context, triggerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.Triggers[triggerID]
	t.Status = models.TriggerStatusInactive
	t.UpdatedAt = time.Now().UTC()
	f.Triggers[triggerID] = t
	return nil
}
