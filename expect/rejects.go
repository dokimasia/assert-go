// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Rejects runs fn against an implementation that fn is meant to reject,
// and records a failure and lets the test continue when fn passes. It
// returns the failure records of fn, in call order, and none when fn
// reported none. Its failure is a record of rejects whose contract is msg,
// which states the rejection that did not happen.
//
//	got := expect.Rejects(t, "an unbounded pool fails the check",
//	    func(tb expect.TB) { handsOutEveryItem(tb, unboundedPool{}) })
//	expect.Length(t, got, 1, "the check fails once")
//
// fn receives the seat of [assert.Rejects], which ends fn at its first
// failure that stops, and the call records of fn are recorded under the
// call of Rejects, as its run 1.
//
// # Concurrency
//
// fn runs on a goroutine of its own and this call blocks until it
// finishes. A panic inside fn is not recovered: it stops the process.
//
// # Allocation contract
//
// A passing call of a check that fails at a call of [True] allocates 8
// times, the check's failure and the returned copy of its records
// included.
func Rejects(tb assert.TB, msg string, fn func(tb assert.TB)) []Failure {
	tb.Helper()
	return matcher.Rejects(tb, matcher.Soft, msg, func(s matcher.Seat) { fn(s) })
}
