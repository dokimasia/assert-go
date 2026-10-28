// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestMaxLiveBytes does not run in parallel: its readings cover the heap of
// the whole process.
func TestMaxLiveBytes(t *testing.T) {
	matchertest.RunMaxLiveBytes(t, func(s *matchertest.Seat, fn func(func()), ceiling uint64, msg string) {
		expect.MaxLiveBytes(s, fn, ceiling, msg)
	})
}
