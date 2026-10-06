// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// NoGoroutineLeaks marks the calling goroutine and returns a check that
// records a failure, and lets the test continue, when a goroutine that the
// scope started is still running when the check is called. The scope is
// what the calling goroutine runs between this call and the check.
//
//	done := expect.NoGoroutineLeaks(t, "the worker stops with its context")
//	defer done()
//
// The call sets a profiler label of the scope on the calling goroutine, and
// every goroutine that the scope starts inherits it, so a goroutine of
// another test is never reported. Call the check on the goroutine that
// called NoGoroutineLeaks, as a deferred call does. See
// [go.dokimi.dev/assert.NoGoroutineLeaks] for the wait of the check, its
// detail and its limits.
//
// # Allocation contract
//
// A call and a check that finds no labelled goroutine allocate at most 175
// times in a process of a few goroutines, the goroutine profile among them.
// The profile grows with the goroutines of the process and with their
// stacks.
func NoGoroutineLeaks(tb assert.TB, msg string) func() {
	tb.Helper()
	return matcher.NoGoroutineLeaks(tb, matcher.Soft, msg)
}
