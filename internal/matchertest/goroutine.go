// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"slices"
	"testing"
	"time"
)

// LeakInvoke calls a surface's leak assertion, returning the check it
// hands back.
type LeakInvoke func(seat *Seat, msg string) func()

// RunNoGoroutineLeaks drives invoke against every case a leak
// assertion must produce. The cases run in parallel, as the tests of a
// package do, so a case fails when the check attributes the goroutines of
// another case to its scope.
func RunNoGoroutineLeaks(t *testing.T, invoke LeakInvoke) {
	t.Helper()

	t.Run("reports nothing when nothing was started", func(t *testing.T) {
		t.Parallel()
		seat := &Seat{}
		invoke(seat, contractMsg)()
		checkOutcome(t, seat, Case{})
	})

	t.Run("reports nothing for a goroutine that finished", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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

	// The profile lists the group of more goroutines first, so a check that
	// skipped the sort would state the function of the second group first.
	t.Run("reports the function of every goroutine still running, in ascending order", func(t *testing.T) {
		t.Parallel()
		seat := &Seat{}
		check := invoke(seat, contractMsg)

		release := make(chan struct{})
		for range firstRunning {
			go func() { <-release }()
		}
		for range secondRunning {
			go func() { <-release }()
		}

		check()
		close(release)

		checkOutcome(t, seat, Case{Fails: true, Assertion: "no-task-leaks"})
		leaked, _ := seat.Records()[0].Detail["leaked"].([]string)
		if len(leaked) != firstRunning+secondRunning || !slices.IsSorted(leaked) ||
			len(slices.Compact(slices.Clone(leaked))) != 2 {
			t.Fatalf("leaked %v, want the functions of the %d and %d goroutines in ascending order",
				seat.Records()[0].Detail["leaked"], firstRunning, secondRunning)
		}
	})

	t.Run("reports a goroutine that a goroutine of the scope starts", func(t *testing.T) {
		t.Parallel()
		seat := &Seat{}
		check := invoke(seat, contractMsg)

		release, started := make(chan struct{}), make(chan struct{})
		go func() {
			go func() {
				close(started)
				<-release
			}()
		}()
		<-started

		check()
		close(release)

		checkOutcome(t, seat, Case{Fails: true, Assertion: "no-task-leaks"})
	})

	t.Run("reports nothing for a goroutine that a goroutine started before the call starts", func(t *testing.T) {
		t.Parallel()
		spawn, started, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
		go func() {
			<-spawn
			go func() {
				close(started)
				<-release
			}()
		}()

		seat := &Seat{}
		check := invoke(seat, contractMsg)
		close(spawn)
		<-started
		check()
		close(release)

		checkOutcome(t, seat, Case{})
	})
}

// The goroutines that the case of a leak starts and leaves running: two
// groups, each of one function, the second the larger.
const (
	firstRunning  = 3
	secondRunning = 5
)
