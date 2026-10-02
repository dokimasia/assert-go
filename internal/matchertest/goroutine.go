// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"testing"
	"time"
)

// LeakInvoke calls a surface's leak assertion, returning the check it
// hands back.
type LeakInvoke func(seat *Seat, msg string) func()

// RunNoGoroutineLeaks drives invoke against every case a leak
// assertion must produce.
//
// These cases do not run in parallel, and a caller must not make them.
// The reading is over the whole process, so a goroutine a parallel
// neighbour starts between the two readings is indistinguishable from
// a leak, and the suite would report noise.
func RunNoGoroutineLeaks(t *testing.T, invoke LeakInvoke) {
	t.Helper()

	t.Run("reports nothing when nothing was started", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, contractMsg)()
		checkOutcome(t, seat, Case{})
	})

	t.Run("reports nothing for a goroutine that finished", func(t *testing.T) {
		seat := &Seat{}
		check := invoke(seat, contractMsg)

		done := make(chan struct{})
		go close(done)
		<-done

		check()
		checkOutcome(t, seat, Case{})
	})

	// The goroutine ends 50 milliseconds into the check, well inside the
	// half second that a check waits for goroutines on their way out.
	t.Run("reports nothing for a goroutine that ends while the check waits", func(t *testing.T) {
		seat := &Seat{}
		check := invoke(seat, contractMsg)

		ended := make(chan struct{})
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(ended)
		}()

		check()
		<-ended
		checkOutcome(t, seat, Case{})
	})

	t.Run("reports a goroutine still running", func(t *testing.T) {
		seat := &Seat{}
		check := invoke(seat, contractMsg)

		release := make(chan struct{})
		go func() { <-release }()

		check()
		close(release)

		checkOutcome(t, seat, Case{Fails: true, Assertion: "no-task-leaks"})
	})
}
