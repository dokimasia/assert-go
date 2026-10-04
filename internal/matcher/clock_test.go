// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
)

// bareSeat implements neither Clocked nor Reporter, like a seat that a test
// framework supplies.
type bareSeat struct{}

func (bareSeat) Helper()               {}
func (bareSeat) Fatalf(string, ...any) {}
func (bareSeat) Errorf(string, ...any) {}

// The results of the allocation cases, which keep each call.
var (
	clockOf    matcher.Clock
	controlled *matcher.Controlled
	instant    time.Time
)

// TestClockOf checks the clock that a seat supplies.
func TestClockOf(t *testing.T) {
	t.Parallel()

	t.Run("returns the runtime clock for a seat without a clock", func(t *testing.T) {
		t.Parallel()

		if got := matcher.ClockOf(&bareSeat{}); got == nil {
			t.Fatal("ClockOf() = nil, want the runtime clock")
		}
	})
}

// TestSystem checks the runtime clock.
func TestSystem(t *testing.T) {
	t.Parallel()

	t.Run("Now", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a later instant after a wait", func(t *testing.T) {
			t.Parallel()

			runtime := matcher.System{}
			first := runtime.Now()
			time.Sleep(2 * time.Millisecond)
			if !runtime.Now().After(first) {
				t.Fatal("Now() did not move, want the runtime clock to advance")
			}
		})
	})
}

// TestControlledClock checks a clock that a test advances.
func TestControlledClock(t *testing.T) {
	t.Parallel()

	t.Run("Advance", func(t *testing.T) {
		t.Parallel()

		t.Run("does not move time backwards", func(t *testing.T) {
			t.Parallel()

			c := matcher.NewControlled(clockEpoch)
			c.Advance(-time.Hour)

			if got := c.Now(); !got.Equal(clockEpoch) {
				t.Fatalf("Now() = %v after a negative advance, want %v", got, clockEpoch)
			}
		})
	})

	t.Run("Sleep", func(t *testing.T) {
		t.Parallel()

		t.Run("returns at once for a duration that is not positive", func(t *testing.T) {
			t.Parallel()

			// A positive duration would block until something advanced
			// the clock, and nothing here will.
			matcher.NewControlled(clockEpoch).Sleep(0)
		})

		// Sleep measures from the instant it reads, so an advance made
		// before it read does not count. The test advances until Sleep
		// returns, a second at a time.
		t.Run("blocks until another goroutine advances the clock past the duration", func(t *testing.T) {
			t.Parallel()

			c := matcher.NewControlled(clockEpoch)
			woke := make(chan struct{})
			go func() {
				c.Sleep(time.Second)
				close(woke)
			}()

			select {
			case <-woke:
				t.Fatal("Sleep returned before the clock moved")
			case <-time.After(20 * time.Millisecond):
			}

			for range 500 {
				c.Advance(time.Second)
				select {
				case <-woke:
					return
				case <-time.After(10 * time.Millisecond):
				}
			}
			t.Fatal("Sleep did not return after 500 advances of a second")
		})
	})
}

// TestClockAllocs checks the allocation ceiling of each function and
// method of clock.go.
func TestClockAllocs(t *testing.T) {
	checkAllocs(t, clockCases())
}

// BenchmarkClock measures each function and method of clock.go.
func BenchmarkClock(b *testing.B) {
	benchAllocs(b, clockCases())
}

// clockCases returns a call of each function and method of clock.go, with
// its allocation ceiling, measured. Sleep sleeps for no time, so it
// returns at once.
func clockCases() []allocCase {
	c := matcher.NewControlled(clockEpoch)
	return []allocCase{
		{name: "ClockOf", call: func(seat matcher.Seat) { clockOf = matcher.ClockOf(seat) }},
		{name: "NewControlled", call: func(matcher.Seat) { controlled = matcher.NewControlled(clockEpoch) }, allocs: 2},
		{name: "Controlled.Now", call: func(matcher.Seat) { instant = c.Now() }},
		{name: "Controlled.Advance", call: func(matcher.Seat) { c.Advance(time.Second) }},
		{name: "Controlled.Sleep", call: func(matcher.Seat) { c.Sleep(0) }},
		{name: "System.Now", call: func(matcher.Seat) { instant = matcher.System{}.Now() }},
		{name: "System.Sleep", call: func(matcher.Seat) { matcher.System{}.Sleep(0) }},
	}
}
