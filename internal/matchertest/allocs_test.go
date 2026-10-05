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

		got := matchertest.OverCeiling("max-allocs-with-setup", true)
		if !got.Fails || got.Assertion != "max-allocs-with-setup" ||
			got.Detail["want"] != uint64(0) || got.Detail["got"] != uint64(1) {
			t.Fatalf("OverCeiling(true) = %+v, want a failure of max-allocs-with-setup with want 0 and got 1", got)
		}
	})

	t.Run("returns a pass in a build that does not count allocations", func(t *testing.T) {
		t.Parallel()

		if got := matchertest.OverCeiling("max-allocs", false); got.Fails || got.Assertion != "" || got.Detail != nil {
			t.Fatalf("OverCeiling(false) = %+v, want a pass", got)
		}
	})
}

// TestAllocsTwins runs TestAllocsTwinsChild in a child process, and
// requires the failures of RunMaxAllocs and RunMaxAllocsWithSetup for a
// twin that never calls the callable.
func TestAllocsTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestAllocsTwinsChild",
		"the fixture allocated nothing, so the case checked no ceiling",
		"called the callable 0 times, want 101",
		"called the setup 0 times and the callable 0 times, want 101 of each",
		"call 0 took another input than the setup before it built")
}

// TestAllocsTwinsChild runs only in the child process of TestAllocsTwins.
func TestAllocsTwinsChild(t *testing.T) {
	inChild(t)

	t.Run("RunMaxAllocs of a twin that never calls the callable", func(t *testing.T) {
		matchertest.RunMaxAllocs(t, func(*matchertest.Seat, func(), uint64, string) {})
	})

	t.Run("RunMaxAllocsWithSetup of a twin that never calls the setup or the callable", func(t *testing.T) {
		matchertest.RunMaxAllocsWithSetup(t, func(*matchertest.Seat, func() *[]byte, func(*[]byte), uint64, string) {})
	})

	t.Run("RunMaxAllocsWithSetup of a twin that hands the callable inputs of its own", func(t *testing.T) {
		matchertest.RunMaxAllocsWithSetup(t,
			func(_ *matchertest.Seat, setup func() *[]byte, fn func(*[]byte), _ uint64, _ string) {
				for range 101 {
					setup()
					fn(new([]byte))
				}
			})
	})
}
