package core

import (
	"testing"
	"time"
)

func TestCoarseNow(t *testing.T) {
	StartCoarseClock()
	// Allow the ticker to fire at least once
	time.Sleep(2 * time.Millisecond)

	got := CoarseNow()
	now := time.Now()

	diff := now.Sub(got)
	if diff < 0 {
		diff = -diff
	}

	// The cached time should be within 5ms of real time
	if diff > 5*time.Millisecond {
		t.Errorf("CoarseNow() drifted %v from time.Now()", diff)
	}
}

func TestStartCoarseClockIdempotent(t *testing.T) {
	// Calling multiple times must not panic
	StartCoarseClock()
	StartCoarseClock()
	StartCoarseClock()

	got := CoarseNow()
	if got.IsZero() {
		t.Error("CoarseNow() returned zero time after multiple StartCoarseClock calls")
	}
}
