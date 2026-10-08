// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// NoGoroutineLeaks marks the calling goroutine and returns a check that
// stops the test when a goroutine that the scope started is still running
// when the check is called. The scope is what the calling goroutine runs
// between this call and the check.
//
//	done := assert.NoGoroutineLeaks(t, "the worker stops with its context")
//	defer done()
//
// The call sets a profiler label of the scope on the calling goroutine, and
// a goroutine inherits the labels of the goroutine that starts it, so every
// goroutine that the scope starts has the label, at any depth. A goroutine
// of another test never has it, so a test that uses the check may call
// [testing.T.Parallel]. Call the check on the goroutine that called
// NoGoroutineLeaks, as a deferred call does: the check clears the labels of
// the goroutine that calls it.
//
// The check reads the goroutine profile up to 100 times, 5 ms apart, and
// reports the labelled goroutines that are still running at the last
// reading. A goroutine that returns during that half second is not a leak.
// The wait is real time, because no clock that a test controls affects when
// a goroutine returns. The record's detail leaked states the function that
// each leaked goroutine runs, in ascending order.
//
// # Limits
//
//   - A goroutine that a goroutine started before the call starts on the
//     scope's behalf, such as a worker of a pool, does not have the label,
//     and the check does not report it.
//   - Code that sets labels from a context without the scope's label, such
//     as [runtime/pprof.Do] with a context of its own, removes the label
//     from the goroutines that it starts.
//   - The call replaces the labels that the calling goroutine had, and the
//     check clears them.
//
// # Allocation contract
//
// A call and a check that finds no labelled goroutine allocate at most 175
// times in a process of a few goroutines, the goroutine profile among them.
// The profile grows with the goroutines of the process and with their
// stacks.
func NoGoroutineLeaks(tb TB, msg string) func() {
	tb.Helper()
	return matcher.NoGoroutineLeaks(tb, matcher.Fatal, msg)
}
