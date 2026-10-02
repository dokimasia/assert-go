// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// MaxAllocsInvoke calls a surface's allocation ceiling.
type MaxAllocsInvoke func(seat *Seat, fn func(), ceiling uint64, msg string)

// warmAndCounted is the number of times an allocation ceiling calls its
// callable: once to warm it, and 100 times to count.
const warmAndCounted = 101

// allocated keeps what allocateOnce built, so escape analysis cannot
// remove the allocation.
var allocated []byte

// allocateOnce makes one heap allocation per call.
func allocateOnce() { allocated = make([]byte, 64) }

// OverCeiling returns the outcome of a ceiling of 0 on a callable that
// allocates once per call: a failure that states the ceiling and the
// count in a build that counts allocations, and none in a build that
// does not.
func OverCeiling(counted bool) Case {
	if !counted {
		return Case{}
	}
	return Case{
		Fails: true, Assertion: "max-allocs",
		Detail: map[string]any{"want": uint64(0), "got": uint64(1)},
	}
}

// RunMaxAllocs drives invoke against every case an allocation ceiling
// must produce. A case over its ceiling fails only in a build whose
// allocation counts describe the code, which matcher.AllocationsCounted
// reports.
//
// The cases do not run in parallel, because testing.AllocsPerRun panics
// while a parallel test runs. The test that calls RunMaxAllocs does not
// call t.Parallel either.
func RunMaxAllocs(t *testing.T, invoke MaxAllocsInvoke) {
	t.Helper()

	t.Run("a callable that allocates nothing passes", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, func() {}, 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a callable at its ceiling passes", func(t *testing.T) {
		allocated = nil

		seat := &Seat{}
		invoke(seat, allocateOnce, 1, contractMsg)
		checkOutcome(t, seat, Case{})

		// A fixture that allocates nothing meets every ceiling, so the
		// case requires the allocation.
		if allocated == nil {
			t.Fatal("the fixture allocated nothing, so the case checked no ceiling")
		}
	})

	t.Run("a callable over its ceiling reports the ceiling and the count", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocateOnce, 0, contractMsg)
		checkOutcome(t, seat, OverCeiling(matcher.AllocationsCounted()))
	})

	t.Run("the call that warms the callable is not counted", func(t *testing.T) {
		var cache []byte
		warmsOnce := func() {
			if cache == nil {
				cache = make([]byte, 64)
			}
		}

		seat := &Seat{}
		invoke(seat, warmsOnce, 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("it calls the callable in every build", func(t *testing.T) {
		calls := 0

		seat := &Seat{}
		invoke(seat, func() { calls++ }, 0, contractMsg)

		if calls != warmAndCounted {
			t.Fatalf("called the callable %d times, want %d", calls, warmAndCounted)
		}
	})
}
