// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"context"
	"strconv"
	"time"

	"go.dokimi.dev/assert"
)

// SubjectDriver calls one assertion with a built behaviour, and with the
// options of a case's relaxations when the assertion takes relaxations.
//
// The assertions taking a callable differ in shape, so each driver states
// how its assertion is called, and the corpus runner calls the driver.
type SubjectDriver func(tb assert.TB, held *Subject, msg string, opts []assert.Option)

// The timeout and the interval of the retrying assertions that a subject
// case drives. The corpus runner hands them a controlled clock, which each
// wait moves forward, so an hour passes without real time.
const (
	retryTimeout  = time.Hour
	retryInterval = time.Minute
)

// SubjectDrivers states how each subject-taking assertion is called, by
// surface and canonical id.
var SubjectDrivers = map[string]map[string]SubjectDriver{
	"check":  drivers(abortingFunctions),
	"expect": drivers(recordingFunctions),
}

// drivers returns the driver of each subject-taking assertion of the
// surface whose functions are f. Each driver passes the subject's Input
// and operands to the assertion, and converts each integer that the
// subject returns to an int.
func drivers(f functions) map[string]SubjectDriver {
	return map[string]SubjectDriver{
		"throws":     func(tb assert.TB, s *Subject, m string, _ []assert.Option) { f.Panics(tb, s.raise, m) },
		"not-throws": func(tb assert.TB, s *Subject, m string, _ []assert.Option) { f.NotPanics(tb, s.raise, m) },
		"honours-cancellation": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.HonoursCancellation(tb, s.ctx, m)
		},
		"honours-deadline": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.HonoursDeadline(tb, s.ctx, m)
		},
		"nil-context-safe": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.NilContextSafe(tb, s.ctx, m)
		},
		"pure": func(tb assert.TB, s *Subject, m string, o []assert.Option) { f.Pure(tb, s.Observe, s.call, m, o...) },
		"not-pure": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.NotPure(tb, s.Observe, s.call, m, o...)
		},
		"eventually": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.Eventually(tb, retryTimeout, retryInterval, s.Seated, m)
		},
		"eventually-true": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.EventuallyTrue(tb, retryTimeout, s.settled, m)
		},
		"idempotent": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.Idempotent(tb, s.Call, s.Input, s.Observe, m, o...)
		},
		"accumulates": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.Accumulates(tb, s.Call, s.Input, s.Observe, m)
		},
		"deterministic": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.Deterministic(tb, s.compute, s.Input, m, o...)
		},
		"commutative": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.Commutative(tb, s.combine, s.A, s.B, m, o...)
		},
		"associative": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.Associative(tb, s.combine, s.A, s.B, s.C, m, o...)
		},
		"round-trip": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.RoundTrip(tb, s.render, strconv.Atoi, s.Input, m, o...)
		},
		"stable-order": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.StableOrder(tb, s.Iterate, m, o...)
		},
		"no-duplicates": func(tb assert.TB, s *Subject, m string, o []assert.Option) {
			f.NoDuplicates(tb, s.Iterate, m, o...)
		},
		"monotonic": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.Monotonic(tb, s.Observe, s.Advance, s.Steps, m)
		},
		"total": func(tb assert.TB, s *Subject, m string, _ []assert.Option) { f.Total(tb, s.callOf, s.Domain, m) },
		"after-close": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.FailsAfterClose(tb, s.Closer, s.Use, s.Sentinel, m)
		},
		"poisoned": func(tb assert.TB, s *Subject, m string, _ []assert.Option) {
			f.Poisoned(tb, s.Induce, s.Read, m)
		},
	}
}

// raise calls Raise with the subject's input.
func (s *Subject) raise() { s.Raise(s.Input) }

// ctx calls Ctx with the handle ctx and the subject's input.
func (s *Subject) ctx(ctx context.Context) error { return s.Ctx(ctx, s.Input) }

// call calls Call with the subject's input, and leaves out its failure.
func (s *Subject) call() { _ = s.Call(s.Input) }

// callOf calls Call with the integer x.
func (s *Subject) callOf(x int) error { return s.Call(x) }

// compute returns what Compute returns for the integer x, as an int.
func (s *Subject) compute(x int) (int, error) { return int(signedOf(s.Compute(x))), nil }

// combine returns what Combine returns for the integers a and b, as an int.
func (s *Subject) combine(a, b int) int { return int(signedOf(s.Combine(a, b))) }

// render returns what Render returns for the integer x.
func (s *Subject) render(x int) (string, error) { return s.Render(x), nil }

// settled reports whether an attempt of the subject's seated shape passes,
// so one behaviour serves both retrying assertions.
func (s *Subject) settled() bool {
	trial := assert.NewRecorder()
	s.Seated(trial)
	return !trial.Failed()
}

// RunSubject drives one subject case with the options of its relaxations,
// and reports whether it ran.
//
// It reports false when this language builds no such behaviour or the
// assertion takes none. The corpus runner fails such a case, because only
// a skip that the definition states excuses one.
func RunSubject(surface, assertion, kind string, tb assert.TB, msg string, opts ...assert.Option) bool {
	build, buildable := Subjects[kind]
	drive, driven := SubjectDrivers[surface][assertion]
	if !buildable || !driven {
		return false
	}
	drive(tb, build(), msg, opts)
	return true
}
