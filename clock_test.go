// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
)

// epoch is the instant a controlled clock starts at, chosen so a test
// reading it back cannot pass by accident against a real clock.
var epoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

func TestControlled(t *testing.T) {
	t.Parallel()

	t.Run("Now", func(t *testing.T) {
		t.Parallel()

		t.Run("answers the start until it is advanced", func(t *testing.T) {
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

		t.Run("does not move backwards", func(t *testing.T) {
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

			// The sleeper is blocked until the clock passes a minute,
			// which is the property under test: a shorter advance must
			// not release it.
			c.Advance(30 * time.Second)
			select {
			case <-done:
				t.Fatal("Sleep returned before the clock reached the duration")
			case <-time.After(20 * time.Millisecond):
			}

			// Advancing well past the duration releases the sleeper
			// whichever side of the first advance it started on, which
			// keeps this from turning on goroutine scheduling.
			c.Advance(time.Hour)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Sleep did not return once the clock passed the duration")
			}
		})
	})
}
