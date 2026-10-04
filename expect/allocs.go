// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// MaxAllocs calls fn once to warm it, counts the heap allocations of the
// next 100 calls, and records a failure and lets the test continue when
// their average, rounded down, exceeds ceiling. The failure names the
// ceiling and the count.
//
//	expect.MaxAllocs(t, func() { _, _ = store.Get(ctx, id) }, 0,
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
// In a build with the race detector, msan or asan, and in one whose
// -gcflags turn off optimisation or inlining, it calls fn as an ordinary
// build does and checks no ceiling. Those builds allocate differently
// from an ordinary build.
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
func MaxAllocs(tb assert.TB, fn func(), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocs(tb, matcher.Soft, fn, ceiling, msg)
}
