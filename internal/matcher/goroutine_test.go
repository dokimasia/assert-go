// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// leakMsg is the caller's message of a leak check.
const leakMsg = "the worker stops"

// TestGoroutine runs the shared cases of the leak check, in parallel. See
// [matchertest.RunNoGoroutineLeaks].
func TestGoroutine(t *testing.T) {
	t.Parallel()

	t.Run("NoGoroutineLeaks", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNoGoroutineLeaks(t, func(s *matchertest.Seat, msg string) func() {
			return matcher.NoGoroutineLeaks(s, matcher.Fatal, msg)
		})

		t.Run("writes the record of a passing check", func(t *testing.T) {
			t.Parallel()
			checkPassRecord(t, "no-task-leaks", func(seat matcher.Seat) {
				matcher.NoGoroutineLeaks(seat, matcher.Fatal, leakMsg)()
			})
		})
	})
}

// TestGoroutineAllocs checks the allocation ceiling of a passing call of
// each function of goroutine.go. A leak check allocates more for more
// goroutines, so the test checks the ceilings in a child process, which
// runs no other test.
func TestGoroutineAllocs(t *testing.T) {
	if !childtest.InChild(t) {
		runChild(t)
		return
	}
	checkAllocs(t, goroutineCases())
}

// BenchmarkGoroutine measures a passing call of each function of
// goroutine.go.
func BenchmarkGoroutine(b *testing.B) {
	benchAllocs(b, goroutineCases())
}

// goroutineCases returns a passing call of each function of goroutine.go,
// with its allocation ceiling, measured: a leak check that finds no
// labelled goroutine.
func goroutineCases() []allocCase {
	return []allocCase{
		{name: "NoGoroutineLeaks", allocs: 163, call: func(seat matcher.Seat) {
			matcher.NoGoroutineLeaks(seat, matcher.Fatal, leakMsg)()
		}},
	}
}
