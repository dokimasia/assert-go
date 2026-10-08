// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// MaxAllocs calls fn once to warm it and counts the heap allocations of the
// next 100 calls. It stops the test when their average, rounded to the
// nearest whole number, exceeds ceiling. The failure names the ceiling and
// the count.
//
//	assert.MaxAllocs(t, func() { _, _ = store.Get(ctx, id) }, 0,
//	    "Get allocates nothing per call once the store is warm")
//
// A half rounds up, so a function that allocates on 50 of the 100 calls
// fails a ceiling of 0. A value that a pool's stock serves on one call and
// that leaks on every other fails it too.
//
// It is the test form of the benchmark ceiling
// [go.dokimi.dev/assert/bench.Contract.MaxAllocs], so the ordinary test
// run checks it.
//
// # Builds that allocate differently
//
// In a build with the race detector, msan or asan, in one whose -gcflags
// turn off optimisation or inlining, and in a test binary that a mutation
// run instrumented, which runs with DOKIMI_MUTATE_INSTRUMENTED in its
// environment, it calls fn as an ordinary build does and checks no
// ceiling. Those builds allocate differently from an ordinary build. The
// ordinary build of one mutant, which a mutation run builds to confirm a
// survivor, checks the ceiling.
//
// It checks no ceiling either in a run that writes the test log of go test,
// which go test passes to every run whose result it can cache. The log
// allocates in some calls of package os, such as an Open through an
// os.Root. There it writes a note into the test's log that a run with
// -count=1 checks the ceiling. A run with -count=1, -covermode or -bench
// turns the cache off and writes no test log.
//
// # Parallel tests
//
// It counts the allocations of the whole process, with GOMAXPROCS at 1, so
// the test that calls MaxAllocs does not call t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the 101 calls of fn
// allocate, and the note in a run that writes the test log.
func MaxAllocs(tb TB, fn func(), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocs(tb, matcher.Fatal, fn, ceiling, msg)
}

// MaxAllocsWithSetup calls setup, and fn on the input that setup returns,
// once to warm both. It then counts the heap allocations of the next 100
// calls of fn, each on an input that a call of setup builds outside the
// count. It stops the test when their average, rounded to the nearest whole
// number with a half rounded up, exceeds ceiling. The failure names the
// ceiling and the count.
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
// In the builds and the runs where [MaxAllocs] checks no ceiling, it calls
// setup and fn as an ordinary build does and checks no ceiling either. It
// writes the note of MaxAllocs in a run that writes the test log.
//
// # Parallel tests
//
// It counts the allocations of the whole process, with GOMAXPROCS at 1, so
// the test that calls MaxAllocsWithSetup does not call t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the 101 calls of setup and
// of fn allocate, and the note in a run that writes the test log.
func MaxAllocsWithSetup[T any](tb TB, setup func() T, fn func(T), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxAllocsWithSetup(tb, matcher.Fatal, setup, fn, ceiling, msg)
}
