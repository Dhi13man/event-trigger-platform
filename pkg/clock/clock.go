package clock

import "time"

// Clock provides an abstraction over time retrieval for deterministic testing.
type Clock interface {
	Now() time.Time
}

// RealClock returns the real current time.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// FixedClock always returns a fixed time. Useful for tests.
type FixedClock struct{ t time.Time }

func NewFixed(t time.Time) FixedClock { return FixedClock{t: t} }

func (f FixedClock) Now() time.Time { return f.t }
