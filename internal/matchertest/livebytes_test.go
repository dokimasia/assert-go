// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestLiveBytesTwins runs TestLiveBytesTwinsProcess in a child process, and
// requires the failures of RunMaxLiveBytes for two broken twins.
func TestLiveBytesTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestLiveBytesTwinsProcess",
		"the fixture copied no entry, so the case read no heap",
		"the fixture built no cache, so the case read no warm call",
		"the faults are [], want the fault of a callable that called the probe 0 times",
		"called the callable 0 times, want 2",
		"called the callable over its ceiling 0 times, want 11",
		"got is 0 bytes, want more than the ceiling of 65536")
}

// TestLiveBytesTwinsProcess runs only in the child process of
// TestLiveBytesTwins.
func TestLiveBytesTwinsProcess(t *testing.T) {
	inChild(t)

	t.Run("RunMaxLiveBytes of a twin that never calls the callable", func(t *testing.T) {
		matchertest.RunMaxLiveBytes(t, func(*matchertest.Seat, func(func()), uint64, string) {})
	})

	t.Run("RunMaxLiveBytes of a twin that fails with no excess", func(t *testing.T) {
		matchertest.RunMaxLiveBytes(t, func(s *matchertest.Seat, fn func(func()), ceiling uint64, msg string) {
			probes := uint64(0)
			fn(func() { probes++ })
			matcher.Fail(s, matcher.Fatal, "max-live-bytes", msg,
				map[string]any{"want": ceiling, "got": uint64(0), "probes": probes})
		})
	})
}
