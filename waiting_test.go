// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

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

func TestWaitingReadsTheSeatsClock(t *testing.T) {
	t.Parallel()

	t.Run("Eventually", func(t *testing.T) {
		t.Parallel()

		t.Run("a body that never passes gives up without real waiting", func(t *testing.T) {
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
			// An hour of controlled time costs no real time, which is the
			// whole point: against the runtime clock this would take an
			// hour.
			if elapsed > 5*time.Second {
				t.Fatalf("spent %v of real time against a controlled clock", elapsed)
			}
		})

		t.Run("a body that settles passes", func(t *testing.T) {
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
}
