// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// Case is one call of a property's body. The body's assertions report to
// the case, and its draws decode the body's inputs from the case's choices.
//
// A Case is an [assert.TB], an [assert.Reporter] and an [assert.Clocked],
// so every assertion of this module works on it. It keeps each record in
// call order, and its first record is the case's failure. A fatal record
// ends the calling goroutine through runtime.Goexit, as testing ends a
// test. Deferred calls run, and a recover returns nil during a Goexit, so a
// body that recovers every panic cannot run on past the end of its case.
//
// A failure's identity is its assertion and its location, and for a panic
// the type of its value and the innermost frame of the caller's code: the
// first frame whose file is a test file or whose function is outside this
// module. Two failures with one identity are one failure, and a message is
// no part of an identity.
//
// When the body ends, however it ends, the case cancels its context and
// runs its cleanups, the last registered first, on the goroutine that ran
// the body. A cleanup is part of the case: a failure in it fails the case,
// and a draw in it is a draw of the case.
//
// # Concurrency
//
// Every method is safe for concurrent use, so an assertion may report to a
// case from any goroutine. A fatal record, a rejection and a draw that ends
// the case end the goroutine that makes them, and the case fails or is
// rejected when its body returns. Call Draw on the goroutine that runs the
// body: draws from other goroutines record their choices in the order the
// scheduler gives, and a replay of those choices decodes other values.
//
// A goroutine that reports to the case must end before the body returns.
// The run reuses the storage of a case whose body has returned for a later
// case, which keeps any report made after that return.
//
// # Allocation contract
//
// Helper, Clock, Assume, Classify of a counted label, Target of a scored
// label and Rand allocate nothing. Observe and Cleanup allocate only to grow
// the case's record, and Logf allocates the message it formats as well.
// History allocates the history on its first call. A recording Report
// allocates twice, for the record's sentence and the message formatted
// from it. Errorf allocates five times: the message, the frames searched
// for its location, and the same two. Context allocates the case's context
// on its first call. A draw allocates the record of its value and what its
// generator decodes.
type Case engine.Case

var (
	_ assert.TB       = (*Case)(nil)
	_ assert.Reporter = (*Case)(nil)
	_ assert.Clocked  = (*Case)(nil)
)

// Helper marks the calling function as a helper. A record states the frame
// that its assertion or message states, so the mark moves no record.
func (c *Case) Helper() {
	(*engine.Case)(c).Helper()
}

// Fatalf keeps a record without an assertion, whose contract is the message
// and whose location is the innermost frame of the caller's code, and ends
// the calling goroutine.
func (c *Case) Fatalf(format string, args ...any) {
	(*engine.Case)(c).Fatalf(format, args...)
}

// Errorf keeps a record without an assertion, whose contract is the message
// and whose location is the innermost frame of the caller's code. The body
// runs on, and the case fails when it returns.
func (c *Case) Errorf(format string, args ...any) {
	(*engine.Case)(c).Errorf(format, args...)
}

// Report keeps an assertion's record f. An aborting record ends the calling
// goroutine. A recording record lets the body run on, and the case fails
// when it returns.
func (c *Case) Report(f assert.Failure, aborting bool) {
	(*engine.Case)(c).Report(f, aborting)
}

// Clock returns the clock of the property's seat, so every case of a
// property runs under the test's clock, and [assert.System] for a seat
// without one.
func (c *Case) Clock() assert.Clock {
	return (*engine.Case)(c).Clock()
}

// Assume rejects the case when condition is false, which ends the calling
// goroutine. A rejected case is not counted, not shrunk, and does not fail
// the property. A run that rejects more than ten cases for every valid one
// fails as [Rejected].
func (c *Case) Assume(condition bool) {
	(*engine.Case)(c).Assume(condition)
}

// Classify counts the case under label, for the coverage requirements that
// [Require] states. A label counted twice in one case counts once.
func (c *Case) Classify(label string) {
	(*engine.Case)(c).Classify(label)
}

// Logf formats its arguments as fmt.Sprintf does and attaches the message
// to the case. Only a failing case reports its messages, after its
// counterexample.
func (c *Case) Logf(format string, args ...any) {
	(*engine.Case)(c).Note(fmt.Sprintf(format, args...))
}

// Rand returns a source of random values whose every value is an integer
// choice of the case over the whole unsigned 64-bit range, so code written
// against math/rand/v2 replays and shrinks without change. A value of the
// source requests an input, as a draw does.
func (c *Case) Rand() rand.Source {
	return (*engine.Case)(c).Rand()
}

// Observe records a fingerprint of the subject's state at this point. A
// replay of the case compares its fingerprints with the recorded ones, and
// a run whose replay records another fingerprint, or another number of
// them, ends as [Flaky].
func (c *Case) Observe(fingerprint uint64) {
	(*engine.Case)(c).Observe(fingerprint)
}

// History returns the case's history, which records the calls that the body
// makes to a subject, for a check of package history or a machine of
// package stateful. It is empty when the case starts.
func (c *Case) History() *history.History {
	return (*engine.Case)(c).History()
}

// Target records score as a score that the case achieved under label. A
// case that records two scores under one label keeps the higher. A campaign
// explores near the cases with the highest score of each label, and every
// other run records the score and generates as if it were absent.
func (c *Case) Target(label string, score float64) {
	(*engine.Case)(c).Target(label, score)
}

// Cleanup registers f to run when the case ends: after its body returns,
// fails, panics, rejects the case or stops at a draw. The case runs its
// cleanups on the goroutine that ran the body, the last registered first.
// A failure in a cleanup fails the case as one in the body does, and the
// later cleanups still run. A cleanup that a cleanup registers runs
// before the case ends, and a draw in a cleanup is a draw of the case.
func (c *Case) Cleanup(f func()) {
	(*engine.Case)(c).Cleanup(f)
}

// Context returns the case's context, for the code under test that takes
// one. It derives from the context of the property's seat, such as the one
// a *testing.T returns, and from context.Background() for a seat without
// one. The case cancels it when the body ends, before the cleanups run.
// A second call returns the same context.
func (c *Case) Context() context.Context {
	return (*engine.Case)(c).Context()
}

// Draw returns a value of g and records it under label for the
// counterexample. Two draws may share a label. A draw ends the calling
// goroutine when the case repeats a tested case, when the body requested
// other choices after the same values in an earlier case, and when the
// case passes its cap on choices.
func (c *Case) Draw[T any](g Generator[T], label string) T {
	return engine.Draw((*engine.Case)(c), engine.Generator[T](g), label)
}

// bodyOf returns the engine's body that calls body with the engine's case
// as a Case, whose context derives from ctx.
func bodyOf(ctx context.Context, body func(*Case)) engine.Body {
	return engine.WithContext(ctx, func(c *engine.Case) { body((*Case)(c)) })
}
