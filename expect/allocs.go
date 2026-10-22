// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// MaxAllocs calls fn once to warm it and counts the heap allocations of the
// next 100 calls. A ceiling of 0 admits one allocation in all. Any other
// ceiling admits an average per call that rounds to at most ceiling, and a
// half rounds up. When a count exceeds ceiling, MaxAllocs counts again,
// ten times at most. It records a failure and lets the test continue when
// no count meets ceiling. The failure has the ceiling as want, the rounded
// average of the count with the lowest total as got, and the allocations of
// that count as total.
//
//	expect.MaxAllocs(t, func() { _, _ = store.Get(ctx, id) }, 0,
//	    "Get allocates nothing per call once the store is warm")
//
// A function that allocates on 2 of the 100 calls fails a ceiling of 0. The
// one allocation that a ceiling of 0 admits covers an allocation that the
// runtime makes while a count runs. A function that grows a slice in the
// first count passes at the second count when the slice no longer grows. A
// function that asserts types for the first time passes once the runtime
// has built the caches of its type assertions, which it builds in about one
// of 1,024 calls of a call site that misses its cache.
//
// It is the test form of the benchmark ceiling
// [go.dokimi.dev/assert/bench.Contract.MaxAllocs], so the ordinary test
// run checks it. The two forms differ at a ceiling of 0. The benchmark
// rounds the average per iteration, so it passes a body that allocates on
// fewer than half of its iterations.
//
// # Builds that allocate differently
//
// In a build with the race detector, msan or asan, in one whose -gcflags
// turn off optimisation or inlining, and in a test binary that a mutation
// run instrumented, which runs with DOKIMI_MUTATE_INSTRUMENTED in its
// environment, it calls fn 101 times and checks no ceiling. Those builds
// allocate differently from an ordinary build. The ordinary build of one
// mutant, which a mutation run builds to confirm a survivor, checks the
// ceiling.
//
// It checks no ceiling either in a run that writes the test log of go test,
// which go test passes to every run whose result it can cache. The log
// allocates in some calls of package os, such as an Open through an
// os.Root. There it writes a note into the test's log that a run with
// -count=1 checks the ceiling. A run with -count=1, -covermode or -bench
// turns the cache off and writes no test log. [AllocsChecked] reports
// whether a run checks the ceiling, and returns the kind of build or run
// that stops the check.
//
// # Parallel tests
//
// It counts the allocations of the whole process, with GOMAXPROCS at 1, so
// the test that calls MaxAllocs does not call t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the calls of fn allocate,
// and the note in a run that writes the test log.
func MaxAllocs(tb assert.TB, fn func(), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocs(tb, matcher.Soft, fn, ceiling, msg)
}

// MaxAllocsWithSetup calls setup, and fn on the input that setup returns,
// once to warm both. It then counts the heap allocations of the next 100
// calls of fn, each on an input that a call of setup builds outside the
// count. It checks each count as [MaxAllocs] does. It records a failure and
// lets the test continue when no count meets ceiling. The failure has the
// fields of a failure of MaxAllocs.
//
//	expect.MaxAllocsWithSetup(t, freshStore, (*Store).Settle, 4,
//	    "settling a store allocates at most four times")
//
// It states a ceiling on a call that consumes its input, which
// [MaxAllocs] would count together with the input's build. A setup can
// also empty a cache before each call: two runs of [runtime.GC] empty
// every [sync.Pool].
//
// # Builds that allocate differently
//
// In the builds and the runs where [MaxAllocs] checks no ceiling, it calls
// setup and fn 101 times each and checks no ceiling either. It writes the
// note of MaxAllocs in a run that writes the test log.
//
// # Parallel tests
//
// It counts the allocations of the whole process, with GOMAXPROCS at 1, so
// the test that calls MaxAllocsWithSetup does not call t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the calls of setup and of
// fn allocate, and the note in a run that writes the test log.
func MaxAllocsWithSetup[T any](tb assert.TB, setup func() T, fn func(T), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocsWithSetup(tb, matcher.Soft, setup, fn, ceiling, msg)
}

// AllocsChecked reports whether [MaxAllocs], [MaxAllocsWithSetup], their
// property forms and the allocation ceilings of a benchmark contract check
// their ceilings in this run, as [go.dokimi.dev/assert.AllocsChecked] does.
// Where they do not, they pass without a count, and reason is the kind of
// build or run that stops the check. Where they do, reason is empty.
//
//	if checked, reason := expect.AllocsChecked(); !checked {
//	    t.Skipf("the run does not check allocation ceilings: %s", reason)
//	}
//
// The result is the same for every call in one run.
//
// # Allocation contract
//
// AllocsChecked allocates nothing after its first call.
func AllocsChecked() (checked bool, reason string) {
	return assert.AllocsChecked()
}
