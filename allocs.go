// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// MaxAllocs calls fn once to warm it, counts the heap allocations of the
// next 100 calls, and stops the test when their average, rounded down,
// exceeds ceiling. The failure names the ceiling and the count.
//
//	assert.MaxAllocs(t, func() { _, _ = store.Get(ctx, id) }, 0,
//	    "Get allocates nothing once the store is warm")
//
// It is the test form of the benchmark ceiling
// [go.dokimi.dev/assert/bench.Contract.MaxAllocs], so the ordinary test
// run checks it.
//
// # Builds that allocate differently
//
// In a build with the race detector, msan or asan, and in one whose
// -gcflags turn off optimisation or inlining, it calls fn as an ordinary
// build does and checks no ceiling. Those builds allocate differently
// from the one that ships.
//
// # Parallel tests
//
// It counts through [testing.AllocsPerRun], which panics while a
// parallel test runs, so the test that calls MaxAllocs does not call
// t.Parallel.
func MaxAllocs(tb TB, fn func(), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocs(tb, matcher.Fatal, fn, ceiling, msg)
}
