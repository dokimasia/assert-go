// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestWaiting(t *testing.T) {
	t.Parallel()

	t.Run("RunEventually", func(t *testing.T) {
		t.Parallel()

		matchertest.RunEventually(t, func(s *matchertest.Seat, timeout, interval time.Duration,
			failing func() bool, msg string,
		) {
			deadline := time.Now().Add(timeout)
			for attempt := 0; ; attempt++ {
				if !failing() {
					return
				}
				if time.Now().After(deadline) {
					s.Fatalf("%s: still failing after %v and %d attempts: %s",
						msg, timeout, attempt+1, matchertest.InnerReason)
					return
				}
				time.Sleep(interval)
			}
		})
	})

	t.Run("RunEventuallyTrue", func(t *testing.T) {
		t.Parallel()

		matchertest.RunEventuallyTrue(t, func(s *matchertest.Seat, timeout time.Duration,
			pred func() bool, msg string,
		) {
			deadline := time.Now().Add(timeout)
			for attempt := 0; ; attempt++ {
				if pred() {
					return
				}
				if time.Now().After(deadline) {
					s.Fatalf("%s: still false after %v and %d attempts", msg, timeout, attempt+1)
					return
				}
				time.Sleep(time.Millisecond)
			}
		})
	})
}
