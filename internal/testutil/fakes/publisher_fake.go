package fakes

import (
	"context"
	"errors"
	"sync"

	platformEvents "github.com/dhima/event-trigger-platform/platform/events"
)

// FakePublisher captures published events and can simulate failures.
type FakePublisher struct {
	mu        sync.Mutex
	Events    []platformEvents.TriggerEvent
	FailNext  bool
	FailError error
}

func (p *FakePublisher) Publish(_ context.Context, e platformEvents.TriggerEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.FailNext {
		p.FailNext = false
		if p.FailError == nil {
			p.FailError = errors.New("publish failed")
		}
		return p.FailError
	}
	p.Events = append(p.Events, e)
	return nil
}
