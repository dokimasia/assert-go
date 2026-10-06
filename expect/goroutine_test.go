// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestGoroutine runs the shared cases of the leak check, in parallel. See
// [matchertest.RunNoGoroutineLeaks].
func TestGoroutine(t *testing.T) {
	t.Parallel()

	t.Run("NoGoroutineLeaks", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNoGoroutineLeaks(t, func(s *matchertest.Seat, msg string) func() {
			return expect.NoGoroutineLeaks(s, msg)
		})
	})
}

// TestGoroutineAllocs checks the allocation ceiling of a passing leak
// check. A leak check allocates more for more goroutines, so the test
// checks the ceiling in a child process, which runs no other test.
func TestGoroutineAllocs(t *testing.T) {
	if childtest.InChild(t) {
		alloctest.Check(t, goroutineCases())
		return
	}
	out, err := childtest.Run(t, t.Name())
	expect.Contains(t, out, "--- PASS: "+t.Name()+" ", "the child process passes the test")
	expect.NoError(t, err, "the child process exits")
}

// BenchmarkGoroutine measures a passing leak check.
func BenchmarkGoroutine(b *testing.B) {
	for _, c := range goroutineCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// goroutineCases returns a call of NoGoroutineLeaks and of its check, which
// finds no new goroutine, with its allocation ceiling, measured.
func goroutineCases() []alloctest.Case {
	return []alloctest.Case{
		{
			Name:   "NoGoroutineLeaks",
			Call:   func(tb assert.TB) { expect.NoGoroutineLeaks(tb, allocContract)() },
			Allocs: 174,
		},
	}
}
