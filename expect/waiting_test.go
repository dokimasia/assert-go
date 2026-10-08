// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestWaiting runs the shared cases of the retrying assertions.
func TestWaiting(t *testing.T) {
	t.Parallel()

	t.Run("Eventually", func(t *testing.T) {
		t.Parallel()
		matchertest.RunEventually(t, func(s *matchertest.Seat, timeout, interval time.Duration,
			failing func() bool, msg string,
		) {
			expect.Eventually(s, timeout, interval, func(tb assert.TB) {
				if failing() {
					expect.True(tb, false, matchertest.InnerReason)
				}
			}, msg)
		})
	})

	t.Run("EventuallyTrue", func(t *testing.T) {
		t.Parallel()
		matchertest.RunEventuallyTrue(t, func(s *matchertest.Seat, timeout time.Duration,
			pred func() bool, msg string,
		) {
			expect.EventuallyTrue(s, timeout, pred, msg)
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
// first attempt, with its allocation ceiling.
func waitingCases() []alloctest.Case {
	settled := func(assert.TB) {}
	ready := func() bool { return true }
	return []alloctest.Case{
		{Name: "Eventually", Allocs: 7, Call: func(tb assert.TB) {
			expect.Eventually(tb, time.Second, time.Millisecond, settled, allocContract)
		}},
		{
			Name: "EventuallyTrue",
			Call: func(tb assert.TB) { expect.EventuallyTrue(tb, time.Second, ready, allocContract) },
		},
	}
}
