// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"math"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// MaxAllocsInvoke calls a surface's allocation ceiling.
type MaxAllocsInvoke func(seat *Seat, fn func(), ceiling uint64, msg string)

// MaxAllocsWithSetupInvoke calls a surface's allocation ceiling with a
// setup, whose inputs are byte slices.
type MaxAllocsWithSetupInvoke func(seat *Seat, setup func() *[]byte, fn func(*[]byte), ceiling uint64, msg string)

// warmAndCounted is the number of times one count of an allocation ceiling
// calls its callable: once to warm it, and 100 times to count.
const warmAndCounted = 101

// threeCounts is the number of times an allocation ceiling calls a callable
// that exceeds it in every count: three counts of warmAndCounted calls.
const threeCounts = 3 * warmAndCounted

// allocated keeps what allocateOnce built, so escape analysis cannot
// remove the allocation.
var allocated []byte

// allocateOnce makes one heap allocation per call.
func allocateOnce() { allocated = make([]byte, 64) }

// fresh returns a new input for a setup: a slice of 64 bytes, behind a
// pointer, built on the heap.
func fresh() *[]byte {
	input := make([]byte, 64)
	return &input
}

// OverCeiling returns the outcome of the assertion with the ceiling want on
// a callable whose every count allocates total times, which exceeds want. In
// a build that counts allocations, the outcome is a failure that states
// want, got as the rounded average, and total. In a build that does not, it
// is a pass.
func OverCeiling(assertion string, counted bool, want, got, total uint64) Case {
	if !counted {
		return Case{}
	}
	return Case{
		Fails: true, Assertion: assertion,
		Detail: map[string]any{"want": want, "got": got, "total": total},
	}
}

// position returns the place of call in its count: 0 for the call that
// warms the callable, and 1 to 100 for the counted calls.
func position(call int) int { return call % warmAndCounted }

// allocatesTimes returns a callable that allocates on each call as many
// times as times reports for the call's number, which counts from 0 for the
// call that warms it.
func allocatesTimes(times func(call int) int) func() {
	call := 0
	return func() {
		for range times(call) {
			allocateOnce()
		}
		call++
	}
}

// halfTheCalls allocates on every even counted call of each count: 50 of
// the 100, whose average of a half rounds up to 1.
func halfTheCalls(call int) int {
	if p := position(call); p > 0 && p%2 == 0 {
		return 1
	}
	return 0
}

// fortyNineCalls allocates on the first 49 counted calls of each count.
func fortyNineCalls(call int) int {
	if p := position(call); p >= 1 && p <= 49 {
		return 1
	}
	return 0
}

// oneCall allocates on the first counted call of each count.
func oneCall(call int) int {
	if position(call) == 1 {
		return 1
	}
	return 0
}

// twoCalls allocates on the first two counted calls of each count.
func twoCalls(call int) int {
	if p := position(call); p == 1 || p == 2 {
		return 1
	}
	return 0
}

// sevenThenOne allocates on the first 7 counted calls of the first count,
// and on the first counted call of each later count, as a slice does that
// doubles while it grows.
func sevenThenOne(call int) int {
	p := position(call)
	if call < warmAndCounted {
		if p >= 1 && p <= 7 {
			return 1
		}
		return 0
	}
	if p == 1 {
		return 1
	}
	return 0
}

// twiceOn returns the allocations of a call that allocates once on every
// counted call, and once more on the first extra counted calls of each
// count.
func twiceOn(extra int) func(call int) int {
	return func(call int) int {
		switch p := position(call); {
		case p == 0:
			return 0
		case p <= extra:
			return 2
		default:
			return 1
		}
	}
}

// RunMaxAllocs drives invoke against every case an allocation ceiling
// must produce. A case over its ceiling fails only in a build whose
// allocation counts describe the code, which matcher.AllocationsCounted
// reports.
//
// The count covers the whole process, so the cases do not run in parallel,
// and the test that calls RunMaxAllocs does not call t.Parallel either.
func RunMaxAllocs(t *testing.T, invoke MaxAllocsInvoke) {
	t.Helper()
	counted := matcher.AllocationsCounted()

	t.Run("a callable that allocates on 50 of the 100 counted calls fails a ceiling of 0", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocatesTimes(halfTheCalls), 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs", counted, 0, 1, 50))
	})

	t.Run("a callable that allocates on 49 of the 100 counted calls fails a ceiling of 0", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocatesTimes(fortyNineCalls), 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs", counted, 0, 0, 49))
	})

	t.Run("a callable that allocates once in every count passes a ceiling of 0", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocatesTimes(oneCall), 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a callable that allocates twice in every count fails a ceiling of 0", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocatesTimes(twoCalls), 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs", counted, 0, 0, 2))
	})

	t.Run("a callable whose allocations fall to one in its second count passes a ceiling of 0", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocatesTimes(sevenThenOne), 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a callable that allocates 149 times in every count passes a ceiling of 1", func(t *testing.T) {
		allocated = nil

		seat := &Seat{}
		invoke(seat, allocatesTimes(twiceOn(49)), 1, contractMsg)
		checkOutcome(t, seat, Case{})

		// A fixture that allocates nothing meets every ceiling, so the
		// case requires the allocation.
		if allocated == nil {
			t.Fatal("the fixture allocated nothing, so the case checked no ceiling")
		}
	})

	t.Run("a callable that allocates 150 times in every count fails a ceiling of 1", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocatesTimes(twiceOn(50)), 1, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs", counted, 1, 2, 150))
	})

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

	t.Run("a callable over its ceiling reports the ceiling, the count and the total", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, allocateOnce, 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs", counted, 0, 1, 100))
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

	t.Run("it calls a callable that meets its ceiling 101 times in every build", func(t *testing.T) {
		calls := 0

		seat := &Seat{}
		invoke(seat, func() { calls++ }, 0, contractMsg)

		if calls != warmAndCounted {
			t.Fatalf("called the callable %d times, want %d", calls, warmAndCounted)
		}
	})

	t.Run("it counts a callable over its ceiling three times in a build that counts allocations", func(t *testing.T) {
		calls := 0
		over := func() {
			allocateOnce()
			calls++
		}

		seat := &Seat{}
		invoke(seat, over, 0, contractMsg)

		want := warmAndCounted
		if counted {
			want = threeCounts
		}
		if calls != want {
			t.Fatalf("called the callable over its ceiling %d times, want %d", calls, want)
		}
	})
}

// RunMaxAllocsWithSetup drives invoke against every case an allocation
// ceiling with a setup must produce. A case over its ceiling fails only in
// a build whose allocation counts describe the code, which
// matcher.AllocationsCounted reports.
//
// The count covers the whole process, so the cases do not run in parallel,
// and the test that calls RunMaxAllocsWithSetup does not call t.Parallel
// either.
func RunMaxAllocsWithSetup(t *testing.T, invoke MaxAllocsWithSetupInvoke) {
	t.Helper()
	counted := matcher.AllocationsCounted()

	t.Run("a callable that allocates on 50 of the 100 counted calls fails a ceiling of 0", func(t *testing.T) {
		allocating := allocatesTimes(halfTheCalls)
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocating() }, 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs-with-setup", counted, 0, 1, 50))
	})

	t.Run("a callable that allocates on 49 of the 100 counted calls fails a ceiling of 0", func(t *testing.T) {
		allocating := allocatesTimes(fortyNineCalls)
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocating() }, 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs-with-setup", counted, 0, 0, 49))
	})

	t.Run("a callable that allocates once in every count passes a ceiling of 0", func(t *testing.T) {
		allocating := allocatesTimes(oneCall)
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocating() }, 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a callable that allocates twice in every count fails a ceiling of 0", func(t *testing.T) {
		allocating := allocatesTimes(twoCalls)
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocating() }, 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs-with-setup", counted, 0, 0, 2))
	})

	t.Run("a callable whose allocations fall to one in its second count passes a ceiling of 0", func(t *testing.T) {
		allocating := allocatesTimes(sevenThenOne)
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocating() }, 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a callable that allocates 149 times in every count passes a ceiling of 1", func(t *testing.T) {
		allocated = nil

		allocating := allocatesTimes(twiceOn(49))
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocating() }, 1, contractMsg)
		checkOutcome(t, seat, Case{})

		// A fixture that allocates nothing meets every ceiling, so the
		// case requires the allocation.
		if allocated == nil {
			t.Fatal("the fixture allocated nothing, so the case checked no ceiling")
		}
	})

	t.Run("a callable that allocates 150 times in every count fails a ceiling of 1", func(t *testing.T) {
		allocating := allocatesTimes(twiceOn(50))
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocating() }, 1, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs-with-setup", counted, 1, 2, 150))
	})

	t.Run("a setup that allocates leaves a callable that allocates nothing within a ceiling of 0", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) {}, 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a callable at its ceiling passes", func(t *testing.T) {
		allocated = nil

		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocateOnce() }, 1, contractMsg)
		checkOutcome(t, seat, Case{})

		// A fixture that allocates nothing meets every ceiling, so the
		// case requires the allocation.
		if allocated == nil {
			t.Fatal("the fixture allocated nothing, so the case checked no ceiling")
		}
	})

	t.Run("a callable over its ceiling reports the ceiling, the count and the total", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, fresh, func(*[]byte) { allocateOnce() }, 0, contractMsg)
		checkOutcome(t, seat, OverCeiling("max-allocs-with-setup", counted, 0, 1, 100))
	})

	t.Run("the call that warms the callable is not counted", func(t *testing.T) {
		var cache []byte
		warmsOnce := func(*[]byte) {
			if cache == nil {
				cache = make([]byte, 64)
			}
		}

		seat := &Seat{}
		invoke(seat, fresh, warmsOnce, 0, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("it calls the callable on the input that the setup before it built, in every build", func(t *testing.T) {
		var built, called []*[]byte
		setup := func() *[]byte {
			input := fresh()
			built = append(built, input)
			return input
		}
		fn := func(input *[]byte) { called = append(called, input) }

		seat := &Seat{}
		invoke(seat, setup, fn, math.MaxUint64, contractMsg)

		if len(built) != warmAndCounted || len(called) != warmAndCounted {
			t.Fatalf("called the setup %d times and the callable %d times, want %d of each",
				len(built), len(called), warmAndCounted)
		}
		for i := range called {
			if called[i] != built[i] {
				t.Fatalf("call %d took another input than the setup before it built", i)
			}
		}
	})

	t.Run("it counts a callable over its ceiling three times in a build that counts allocations", func(t *testing.T) {
		calls := 0
		over := func(*[]byte) {
			allocateOnce()
			calls++
		}

		seat := &Seat{}
		invoke(seat, fresh, over, 0, contractMsg)

		want := warmAndCounted
		if counted {
			want = threeCounts
		}
		if calls != want {
			t.Fatalf("called the callable over its ceiling %d times, want %d", calls, want)
		}
	})
}
