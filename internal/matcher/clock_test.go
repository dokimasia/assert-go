// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"context"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// clockEpoch is the instant a controlled clock starts at, chosen so a
// reading cannot pass by accident against a real clock.
var clockEpoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// bareSeat carries neither a clock nor a record, which is what a seat
// supplied by a framework looks like.
type bareSeat struct{}

func (bareSeat) Helper()               {}
func (bareSeat) Fatalf(string, ...any) {}
func (bareSeat) Errorf(string, ...any) {}

func TestClockOf(t *testing.T) {
	t.Parallel()

	t.Run("answers the runtime clock for a seat that carries none", func(t *testing.T) {
		t.Parallel()

		if got := matcher.ClockOf(&bareSeat{}); got == nil {
			t.Fatal("ClockOf() = nil, want the runtime clock")
		}
	})
}

func TestSystem(t *testing.T) {
	t.Parallel()

	t.Run("Now", func(t *testing.T) {
		t.Parallel()

		t.Run("moves on its own", func(t *testing.T) {
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

// clockedSeat is a seat with a clock, which records what was reported.
type clockedSeat struct {
	matchertest.Seat

	clock matcher.Clock
}

// Clock returns the seat's clock.
func (s *clockedSeat) Clock() matcher.Clock { return s.clock }

// TestHonoursDeadlineAgainstASuppliedClock pins that the assertion does
// not read the deadline off the seat's clock.
//
// A context decides expiry against the runtime clock and takes no
// other. Building the deadline from a seat clock reading ahead of the
// runtime hands the subject a deadline that has not passed, and a
// subject that correctly reports nothing is then reported as wrong.
func TestHonoursDeadlineAgainstASuppliedClock(t *testing.T) {
	t.Parallel()

	// Honours its deadline: it answers whatever the context says.
	honours := func(ctx context.Context) error { return ctx.Err() }

	for _, tc := range []struct {
		name  string
		start time.Time
	}{
		{name: "a clock behind the runtime's", start: clockEpoch},
		{name: "a clock ahead of the runtime's", start: time.Now().Add(time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seat := &clockedSeat{clock: matcher.NewControlled(tc.start)}
			matcher.HonoursDeadline(seat, matcher.Fatal, honours,
				"the subject reports why it stopped")

			if seat.Failed() {
				t.Error("reported a subject that honoured its deadline")
			}
		})
	}
}
