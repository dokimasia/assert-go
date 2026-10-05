// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/record"
)

// Rejects runs fn against an implementation that fn is meant to reject,
// and stops the test when fn passes. It returns the failure records of fn,
// in call order, and none when fn reported none. Its failure is a record of
// rejects whose contract is msg, which states the rejection that did not
// happen.
//
// Rejects is the assertion that an assertion can fail. A check whose
// every statement is [NoError] passes against a subject whose methods do
// nothing and return nil, so the check counts as coverage and verifies
// nothing. The caller names the wrong implementation, and Rejects runs
// the check against it and reads the rejection.
//
//	assert.Rejects(t, "a store that overwrites fails the check",
//	    func(tb assert.TB) { refusesADuplicate(tb, overwritingStore{}) })
//
// Assert on the returned records. A check can fail for a reason other
// than the one it is about, such as a subject that panics inside
// [NotPanics] before the check's own assertion runs. A bare call of
// Rejects passes then, and a record names the assertion that failed, with
// its contract and its detail:
//
//	got := assert.Rejects(t, "an unbounded pool fails the check",
//	    func(tb assert.TB) { handsOutEveryItem(tb, unboundedPool{}) })
//	assert.Length(t, got, 1, "the check fails once")
//	assert.Equal(t, got[0].Contract, "the pool is then empty",
//	    "and fails for the reason the check is about")
//
// A message that fn passes to Fatalf or Errorf of its seat, and a fault of
// this module, fail fn without a record. Rejects passes then, and returns
// the records of fn's assertions alone.
//
// The call records of the assertions that fn calls are recorded under
// the call of Rejects, as its run 1.
//
// # Concurrency
//
// fn runs on a goroutine of its own and this call blocks until it
// finishes. The seat fn receives ends that goroutine at fn's first
// failure, so a check does not run past an assertion it already failed.
// The goroutine ends before the call returns, so a leak check around this
// call reports nothing.
//
// A panic inside fn is not recovered. It crosses the goroutine boundary
// and stops the process. A check that panics is a defect in the check or
// in the wrong implementation, and reporting it as a rejection would hide
// it.
//
// # Allocation contract
//
// A passing call of a check that fails at a call of [True] allocates 9
// times, the check's failure and the returned copy of its records
// included.
func Rejects(tb TB, msg string, fn func(tb TB)) []Failure {
	tb.Helper()

	run := matcher.Begin(tb)
	r := NewRecorder().WithGoexit()
	record.Run(&r.calls, run.Slot(), nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(r)
	}()
	<-done
	run.Slot().Take(&r.calls, record.NoPhase)

	if !r.Failed() {
		run.Fail(matcher.Fatal, "rejects", msg, nil)
		return r.Failures()
	}
	run.Pass(matcher.Fatal, "rejects", msg)
	return r.Failures()
}
