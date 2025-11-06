package clock

import (
	"testing"
	"time"
)

func TestRealClock_Now_WhenCalled_ThenReturnsCurrentTime(t *testing.T) {
	// Arrange
	realClock := RealClock{}
	beforeCall := time.Now()

	// Act
	result := realClock.Now()

	// Assert
	afterCall := time.Now()
	if result.Before(beforeCall) || result.After(afterCall) {
		t.Errorf("expected time between %v and %v, got %v", beforeCall, afterCall, result)
	}
}

func TestFixedClock_Now_WhenCalled_ThenReturnsFixedTime(t *testing.T) {
	// Arrange
	fixedTime := time.Date(2025, 11, 6, 10, 30, 0, 0, time.UTC)
	fixedClock := NewFixed(fixedTime)

	// Act
	result1 := fixedClock.Now()
	time.Sleep(10 * time.Millisecond)
	result2 := fixedClock.Now()

	// Assert
	if !result1.Equal(fixedTime) {
		t.Errorf("expected first call to return %v, got %v", fixedTime, result1)
	}
	if !result2.Equal(fixedTime) {
		t.Errorf("expected second call to return %v, got %v", fixedTime, result2)
	}
	if !result1.Equal(result2) {
		t.Errorf("expected both calls to return same time, got %v and %v", result1, result2)
	}
}

func TestNewFixed_WhenCreatedWithTime_ThenReturnsFixedClockWithThatTime(t *testing.T) {
	// Arrange
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	clock := NewFixed(testTime)

	// Assert
	if !clock.Now().Equal(testTime) {
		t.Errorf("expected Now() to return %v, got %v", testTime, clock.Now())
	}
}

func TestFixedClock_Now_WhenZeroTime_ThenReturnsZeroTime(t *testing.T) {
	// Arrange
	zeroTime := time.Time{}
	fixedClock := NewFixed(zeroTime)

	// Act
	result := fixedClock.Now()

	// Assert
	if !result.IsZero() {
		t.Errorf("expected zero time, got %v", result)
	}
}
