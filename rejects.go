// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

// Rejects runs fn against an implementation fn is meant to reject, and
// stops the test when fn passes. It returns fn's failure message,
// empty when there was none.
//
// This is the assertion that an assertion can fail. A check whose
// every statement is [NoError] passes against a subject whose methods
// do nothing and return nil: it reads as coverage and establishes
// nothing. Rejects names the wrong implementation, drives the check
// against it, and reads the rejection.
//
//	assert.Rejects(t, "a store that overwrites fails the check",
//	    func(tb assert.TB) { refusesADuplicate(tb, overwritingStore{}) })
//
// Assert on the returned message. A subject that panics before
// reaching the assertion satisfies a bare call while the check's own
// assertion never ran, which is the defect this exists to catch, one
// level up:
//
//	got := assert.Rejects(t, "an unbounded pool fails the check",
//	    func(tb assert.TB) { handsOutWhatItHolds(tb, unboundedPool{}) })
//	assert.Contains(t, got, "the pool is then empty",
//	    "and fails for the reason the check is about")
//
// # Concurrency
//
// fn runs on a goroutine of its own and this call blocks until it
// finishes. The seat fn receives stops it at its first failure by
// ending that goroutine, which is what keeps a check from running past
// an assertion it already failed. The goroutine does not outlive the
// call, so a leak check around this stays quiet.
//
// A panic inside fn is not recovered. It crosses the goroutine
// boundary and stops the process, which is right: a check that panics
// is a defect in the check or in the stand-in, and reporting it as a
// rejection would hide it.
func Rejects(tb TB, msg string, fn func(tb TB)) string {
	tb.Helper()

	r := NewRecorder().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(r)
	}()
	<-done

	if !r.Failed() {
		tb.Fatalf("%s: the check passed against an implementation it must reject", msg)
	}
	return r.Msg()
}
