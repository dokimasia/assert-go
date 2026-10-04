// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestOverCeiling(t *testing.T) {
	t.Parallel()

	t.Run("returns a failure of the ceiling and the count in a build that counts allocations", func(t *testing.T) {
		t.Parallel()

		got := matchertest.OverCeiling(true)
		if !got.Fails || got.Assertion != "max-allocs" ||
			got.Detail["want"] != uint64(0) || got.Detail["got"] != uint64(1) {
			t.Fatalf("OverCeiling(true) = %+v, want a failure of max-allocs with want 0 and got 1", got)
		}
	})

	t.Run("returns a pass in a build that does not count allocations", func(t *testing.T) {
		t.Parallel()

		if got := matchertest.OverCeiling(false); got.Fails || got.Assertion != "" || got.Detail != nil {
			t.Fatalf("OverCeiling(false) = %+v, want a pass", got)
		}
	})
}

// TestAllocsTwins runs TestAllocsTwinsChild in a child process, and
// requires the failures of RunMaxAllocs for a twin that never calls the
// callable.
func TestAllocsTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestAllocsTwinsChild",
		"the fixture allocated nothing, so the case checked no ceiling",
		"called the callable 0 times, want 101")
}

// TestAllocsTwinsChild runs only in the child process of TestAllocsTwins.
func TestAllocsTwinsChild(t *testing.T) {
	inChild(t)

	t.Run("RunMaxAllocs of a twin that never calls the callable", func(t *testing.T) {
		matchertest.RunMaxAllocs(t, func(*matchertest.Seat, func(), uint64, string) {})
	})
}
