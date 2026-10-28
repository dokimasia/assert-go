// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestMaxLiveBytes does not run in parallel: its readings cover the heap of
// the whole process.
func TestMaxLiveBytes(t *testing.T) {
	matchertest.RunMaxLiveBytes(t, func(s *matchertest.Seat, fn func(func()), ceiling uint64, msg string) {
		assert.MaxLiveBytes(s, fn, ceiling, msg)
	})
}
