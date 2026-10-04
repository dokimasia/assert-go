// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// NoGoroutineLeaks records which goroutines are running and returns a
// check that stops the test when a goroutine started after this call is
// still running when the check is called.
//
//	done := assert.NoGoroutineLeaks(t, "the worker stops with its context")
//	defer done()
//
// The check compares goroutine ids, not counts, so a goroutine that was
// already running is never reported.
//
// The check reads the running goroutines up to 100 times, 5 ms apart, and
// reports the new goroutines that are still running at the last reading.
// A goroutine that returns during that half second is not a leak. The
// wait is real time, because no clock that a test controls affects when a
// goroutine returns.
//
// Each reading dumps the stacks of every goroutine into a buffer of at
// most 64 MiB. A dump that does not fit leaves the set of new goroutines
// unknown, so the check stops the test with a fault and states no
// verdict.
//
// # Parallel tests
//
// A goroutine that a parallel test starts between the two readings is
// new, so the check reports it as a leak. Do not call [testing.T.Parallel]
// in a test that uses this check. A package whose other tests are
// parallel can still produce a false report, because each reading covers
// the whole process.
//
// # Allocation contract
//
// A call and a check that finds no new goroutine allocate 7 times in a
// process of a few goroutines, the 1 MiB buffer of the dumps among them.
// The sets of goroutine ids grow with the goroutines of the process, so a
// process of more goroutines allocates more.
func NoGoroutineLeaks(tb TB, msg string) func() {
	tb.Helper()
	return matcher.NoGoroutineLeaks(tb, matcher.Fatal, msg)
}
