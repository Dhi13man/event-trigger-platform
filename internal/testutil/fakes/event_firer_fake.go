package fakes

import (
	"context"
	"errors"

	"github.com/dhima/event-trigger-platform/internal/models"
)

// FakeEventFirer simulates event firing outcome.
type FakeEventFirer struct {
	Fail bool
}

func (f *FakeEventFirer) FireTrigger(_ context.Context, _ *models.Trigger, _ models.EventSource, _ map[string]interface{}, _ bool) (string, error) {
	if f.Fail {
		return "", errors.New("fire failed")
	}
	return "event-123", nil
}
