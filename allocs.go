// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// MaxAllocs calls fn once to warm it, counts the heap allocations of the
// next 100 calls, and stops the test when their average, rounded down,
// exceeds ceiling. The failure names the ceiling and the count.
//
//	assert.MaxAllocs(t, func() { _, _ = store.Get(ctx, id) }, 0,
//	    "Get averages under one allocation per call once the store is warm")
//
// The rounding passes a ceiling of 0 for a function that allocates on
// 99 of the 100 calls.
//
// It is the test form of the benchmark ceiling
// [go.dokimi.dev/assert/bench.Contract.MaxAllocs], so the ordinary test
// run checks it.
//
// # Builds that allocate differently
//
// In a build with the race detector, msan or asan, in one whose -gcflags
// turn off optimisation or inlining, and in a test binary that a mutation
// run instrumented, which runs with DOKIMI_MUTATE_MUTANT in its
// environment, it calls fn as an ordinary build does and checks no
// ceiling. Those builds allocate differently from an ordinary build.
//
// # Parallel tests
//
// It counts through [testing.AllocsPerRun], which panics while a
// parallel test runs, so the test that calls MaxAllocs does not call
// t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the 101 calls of fn
// allocate.
func MaxAllocs(tb TB, fn func(), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocs(tb, matcher.Fatal, fn, ceiling, msg)
}

// MaxAllocsWithSetup calls setup, and fn on the input that setup returns,
// once to warm both. It then counts the heap allocations of the next 100
// calls of fn, each on an input that a call of setup builds outside the
// count, and stops the test when their average, rounded down, exceeds
// ceiling. The failure names the ceiling and the count.
//
//	assert.MaxAllocsWithSetup(t, freshStore, (*Store).Settle, 4,
//	    "settling a store allocates at most four times")
//
// It states a ceiling on a call that consumes its input, which
// [MaxAllocs] would count together with the input's build. A setup can
// also empty a cache before each call: two runs of [runtime.GC] empty
// every [sync.Pool].
//
// # Builds that allocate differently
//
// In the builds where [MaxAllocs] checks no ceiling, it calls setup and fn
// as an ordinary build does and checks no ceiling either.
//
// # Parallel tests
//
// It counts the allocations of the whole process, with GOMAXPROCS at 1, so
// the test that calls MaxAllocsWithSetup does not call t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the 101 calls of setup and
// of fn allocate.
func MaxAllocsWithSetup[T any](tb TB, setup func() T, fn func(T), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocsWithSetup(tb, matcher.Fatal, setup, fn, ceiling, msg)
}
