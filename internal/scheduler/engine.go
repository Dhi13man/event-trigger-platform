package scheduler

import (
	"context"
	"log"
	"time"
)

// Engine periodically scans for triggers that are due to fire and enqueues them.
type Engine struct {
	tick time.Duration
}

// NewEngine constructs a scheduler with the provided polling cadence.
func NewEngine(tick time.Duration) *Engine {
	return &Engine{tick: tick}
}

// Run begins the polling loop; integrate with storage and queue once ready.
func (e *Engine) Run(ctx context.Context) error {
	ticker := time.NewTicker(e.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Printf("scheduler tick (stub): %s", time.Now().Format(time.RFC3339))
			// TODO: fetch due triggers from storage and publish to the queue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
