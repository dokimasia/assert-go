// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
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
	})
}
