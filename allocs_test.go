// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestMaxAllocs does not run in parallel: testing.AllocsPerRun panics
// while a parallel test runs.
func TestMaxAllocs(t *testing.T) {
	matchertest.RunMaxAllocs(t, func(s *matchertest.Seat, fn func(), ceiling uint64, msg string) {
		assert.MaxAllocs(s, fn, ceiling, msg)
	})
}
