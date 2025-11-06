package fakes

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
)

var ErrNotFound = errors.New("not found")

// FakeTriggerStore is an in-memory implementation of the TriggerStore interface.
type FakeTriggerStore struct {
	mu        sync.Mutex
	triggers  map[string]models.Trigger
	schedules map[string][]models.TriggerSchedule // by triggerID
}

func NewFakeTriggerStore() *FakeTriggerStore {
	return &FakeTriggerStore{
		triggers:  make(map[string]models.Trigger),
		schedules: make(map[string][]models.TriggerSchedule),
	}
}

func (f *FakeTriggerStore) CreateTrigger(_ context.Context, trigger *models.Trigger, schedule *models.TriggerSchedule) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := *trigger
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	t.UpdatedAt = t.CreatedAt
	f.triggers[t.ID] = t
	if schedule != nil {
		s := *schedule
		if s.CreatedAt.IsZero() {
			s.CreatedAt = time.Now().UTC()
		}
		s.UpdatedAt = s.CreatedAt
		f.schedules[t.ID] = append(f.schedules[t.ID], s)
	}
	return nil
}

func (f *FakeTriggerStore) GetTrigger(_ context.Context, triggerID string) (*models.Trigger, *time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.triggers[triggerID]
	if !ok {
		return nil, nil, ErrNotFound
	}
	var next *time.Time
	for i := range f.schedules[triggerID] {
		s := f.schedules[triggerID][i]
		if s.Status == models.ScheduleStatusPending || s.Status == models.ScheduleStatusProcessing {
			if next == nil || s.FireAt.Before(*next) {
				t := s.FireAt
				next = &t
			}
		}
	}
	return &t, next, nil
}

func (f *FakeTriggerStore) ListTriggers(_ context.Context, query models.ListTriggersQuery) ([]models.Trigger, []*time.Time, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := make([]models.Trigger, 0, len(f.triggers))
	for _, t := range f.triggers {
		if query.Type != "" && string(t.Type) != query.Type {
			continue
		}
		if query.Status != "" && string(t.Status) != query.Status {
			continue
		}
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool { return strings.Compare(list[i].ID, list[j].ID) < 0 })
	total := int64(len(list))
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	start := (query.Page - 1) * query.Limit
	if start > len(list) {
		return []models.Trigger{}, []*time.Time{}, total, nil
	}
	end := start + query.Limit
	if end > len(list) {
		end = len(list)
	}
	page := list[start:end]
	nexts := make([]*time.Time, 0, len(page))
	for i := range page {
		var next *time.Time
		for _, s := range f.schedules[page[i].ID] {
			if s.Status == models.ScheduleStatusPending || s.Status == models.ScheduleStatusProcessing {
				if next == nil || s.FireAt.Before(*next) {
					t := s.FireAt
					next = &t
				}
			}
		}
		nexts = append(nexts, next)
	}
	return page, nexts, total, nil
}

func (f *FakeTriggerStore) UpdateTrigger(_ context.Context, triggerID string, updates map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.triggers[triggerID]
	if !ok {
		return ErrNotFound
	}
	for k, v := range updates {
		switch k {
		case "name":
			t.Name = v.(string)
		case "status":
			t.Status = v.(models.TriggerStatus)
		case "config":
			t.Config = []byte(v.(string))
		}
	}
	t.UpdatedAt = time.Now().UTC()
	f.triggers[triggerID] = t
	return nil
}

func (f *FakeTriggerStore) DeleteTrigger(_ context.Context, triggerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.triggers[triggerID]; !ok {
		return ErrNotFound
	}
	delete(f.triggers, triggerID)
	delete(f.schedules, triggerID)
	return nil
}

func (f *FakeTriggerStore) UpsertTriggerSchedule(_ context.Context, triggerID string, schedule *models.TriggerSchedule) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.triggers[triggerID]; !ok {
		return ErrNotFound
	}
	// cancel existing pending/processing
	arr := f.schedules[triggerID]
	for i := range arr {
		if arr[i].Status == models.ScheduleStatusPending || arr[i].Status == models.ScheduleStatusProcessing {
			arr[i].Status = models.ScheduleStatusCancelled
			arr[i].UpdatedAt = time.Now().UTC()
		}
	}
	f.schedules[triggerID] = arr
	if schedule != nil {
		s := *schedule
		if s.CreatedAt.IsZero() {
			s.CreatedAt = time.Now().UTC()
		}
		s.UpdatedAt = s.CreatedAt
		f.schedules[triggerID] = append(f.schedules[triggerID], s)
	}
	return nil
}
