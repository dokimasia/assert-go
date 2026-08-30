// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// NoGoroutineLeaks records which goroutines are running and returns a
// check that records a failure and lets the test continue if any of them are still running when it
// is called.
//
//	done := assert.NoGoroutineLeaks(t, "the worker stops with its context")
//	defer done()
//
// Identity, not count: only goroutines started after this call are
// reported, so goroutines already running are never blamed.
//
// The check waits up to half a second, polling, before reporting. A
// goroutine on its way out is not a leak, and without that grace every
// test starting one would be flaky. Real time, and necessarily: no
// clock a test controls governs when a goroutine returns.
//
// # What it cannot tell apart
//
// A goroutine started by a neighbouring test between the two readings
// looks exactly like a leak, because both are simply new. Do not call
// [testing.T.Parallel] in a test using this, and be aware that a
// package whose other tests are parallel can still produce a false
// report. The reading is over the whole process; nothing scopes it to
// one test.
func NoGoroutineLeaks(tb assert.TB, msg string) func() {
	tb.Helper()
	return matcher.NoGoroutineLeaks(tb, matcher.Soft, msg)
}
