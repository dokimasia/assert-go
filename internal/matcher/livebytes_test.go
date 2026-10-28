// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestMaxLiveBytes does not run in parallel: its readings cover the heap of
// the whole process.
func TestMaxLiveBytes(t *testing.T) {
	matchertest.RunMaxLiveBytes(t, func(s *matchertest.Seat, fn func(func()), ceiling uint64, msg string) {
		matcher.MaxLiveBytes(s, matcher.Fatal, fn, ceiling, msg)
	})

	t.Run("writes the record of a passing call", func(t *testing.T) {
		checkPassRecord(t, "max-live-bytes", func(seat matcher.Seat) {
			matcher.MaxLiveBytes(seat, matcher.Fatal, func(probe func()) { probe() }, 1<<20, allocContract)
		})
	})
}
