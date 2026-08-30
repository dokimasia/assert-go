// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// Not parallel, and neither are its cases: the reading is over the
// whole process. See [matchertest.RunNoGoroutineLeaks].
func TestGoroutine(t *testing.T) {
	t.Run("NoGoroutineLeaks", func(t *testing.T) {
		matchertest.RunNoGoroutineLeaks(t, func(s *matchertest.Seat, msg string) func() {
			return matcher.NoGoroutineLeaks(s, matcher.Fatal, msg)
		})
	})
}
