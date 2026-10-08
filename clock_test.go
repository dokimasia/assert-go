// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
)

// controlled keeps the clock that a call of NewControlled returns.
var controlled *assert.Controlled

// TestControlled checks a clock that a test advances.
func TestControlled(t *testing.T) {
	t.Parallel()

	t.Run("Now", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the start until it is advanced", func(t *testing.T) {
			t.Parallel()

			c := assert.NewControlled(epoch)
			if got := c.Now(); !got.Equal(epoch) {
				t.Fatalf("Now() = %v, want %v", got, epoch)
			}

			c.Advance(time.Hour)
			if got, want := c.Now(), epoch.Add(time.Hour); !got.Equal(want) {
				t.Fatalf("Now() = %v, want %v", got, want)
			}
		})

		t.Run("returns the start after a negative advance", func(t *testing.T) {
			t.Parallel()

			c := assert.NewControlled(epoch)
			c.Advance(-time.Hour)

			if got := c.Now(); !got.Equal(epoch) {
				t.Fatalf("Now() = %v after a negative advance, want %v", got, epoch)
			}
		})
	})

	t.Run("Sleep", func(t *testing.T) {
		t.Parallel()

		t.Run("returns once another goroutine advances past it", func(t *testing.T) {
			t.Parallel()

			c := assert.NewControlled(epoch)
			done := make(chan struct{})
			go func() {
				c.Sleep(time.Minute)
				close(done)
			}()

			// The sleeper is blocked until the clock passes a minute, so an
			// advance of half a minute leaves it blocked.
			c.Advance(30 * time.Second)
			select {
			case <-done:
				t.Fatal("Sleep returned before the clock passed the duration")
			case <-time.After(20 * time.Millisecond):
			}

			// An advance of an hour releases the sleeper whichever side of
			// the first advance it started on, so the case does not depend
			// on the scheduling of the goroutines.
			c.Advance(time.Hour)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Sleep did not return once the clock passed the duration")
			}
		})
	})
}

// TestClockAllocs checks the allocation ceiling of NewControlled.
func TestClockAllocs(t *testing.T) {
	alloctest.Check(t, clockCases())
}

// BenchmarkClock measures NewControlled.
func BenchmarkClock(b *testing.B) {
	for _, c := range clockCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// clockCases returns a call of NewControlled, with its allocation ceiling,
// measured.
func clockCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "NewControlled", Call: func(assert.TB) { controlled = assert.NewControlled(epoch) }, Allocs: 2},
	}
}
