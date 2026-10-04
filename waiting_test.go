// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestWaiting runs the shared cases of the retrying assertions, and their
// cases on a seat's clock.
func TestWaiting(t *testing.T) {
	t.Parallel()

	t.Run("Eventually", func(t *testing.T) {
		t.Parallel()
		matchertest.RunEventually(t, func(s *matchertest.Seat, timeout, interval time.Duration,
			failing func() bool, msg string,
		) {
			assert.Eventually(s, timeout, interval, func(tb assert.TB) {
				if failing() {
					assert.True(tb, false, matchertest.InnerReason)
				}
			}, msg)
		})

		t.Run("gives up on a body that never passes without real waiting on a seat's clock", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder().WithClock(assert.NewControlled(epoch))

			started := time.Now()
			assert.Eventually(r, time.Hour, time.Minute,
				func(tb assert.TB) { tb.Errorf("never settles") },
				"the body settles")
			elapsed := time.Since(started)

			if !r.Failed() {
				t.Fatal("a body that never passes reports")
			}
			// An hour of controlled time costs no real time. Against the
			// runtime clock, this call takes an hour.
			if elapsed > 5*time.Second {
				t.Fatalf("spent %v of real time against a controlled clock", elapsed)
			}
		})

		t.Run("passes a body that settles on a seat's clock", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder().WithClock(assert.NewControlled(epoch))

			attempts := 0
			assert.Eventually(r, time.Hour, time.Minute, func(tb assert.TB) {
				attempts++
				if attempts < 3 {
					tb.Errorf("not yet")
				}
			}, "the body settles by the third attempt")

			if r.Failed() {
				t.Fatalf("a body that settles reports: %s", r.Message())
			}
			if attempts != 3 {
				t.Fatalf("ran %d attempts, want 3", attempts)
			}
		})
	})

	t.Run("EventuallyTrue", func(t *testing.T) {
		t.Parallel()
		matchertest.RunEventuallyTrue(t, func(s *matchertest.Seat, timeout time.Duration,
			pred func() bool, msg string,
		) {
			assert.EventuallyTrue(s, timeout, pred, msg)
		})
	})
}

// TestWaitingAllocs checks the allocation ceiling of a passing call of
// each retrying assertion.
func TestWaitingAllocs(t *testing.T) {
	alloctest.Check(t, waitingCases())
}

// BenchmarkWaiting measures a passing call of each retrying assertion.
func BenchmarkWaiting(b *testing.B) {
	for _, c := range waitingCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// waitingCases returns a call of each retrying assertion that passes at its
// first attempt, with its allocation ceiling, measured.
func waitingCases() []alloctest.Case {
	settled := func(assert.TB) {}
	ready := func() bool { return true }
	return []alloctest.Case{
		{Name: "Eventually", Allocs: 5, Call: func(tb assert.TB) {
			assert.Eventually(tb, time.Second, time.Millisecond, settled, allocContract)
		}},
		{
			Name: "EventuallyTrue",
			Call: func(tb assert.TB) { assert.EventuallyTrue(tb, time.Second, ready, allocContract) },
		},
	}
}
