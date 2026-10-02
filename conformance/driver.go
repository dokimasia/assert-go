// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

// SubjectDriver calls one assertion with a built behaviour.
//
// The assertions taking a callable differ in shape, so each driver states
// how its assertion is called, and the corpus runner calls the driver.
type SubjectDriver func(tb assert.TB, held *Subject, msg string)

// The timeout and the interval of the retrying assertions that a subject
// case drives. The corpus runner hands them a controlled clock, which each
// wait moves forward, so an hour passes without real time.
const (
	retryTimeout  = time.Hour
	retryInterval = time.Minute
)

// SubjectDrivers states how each subject-taking assertion is called, by
// canonical id and surface.
var SubjectDrivers = map[string]map[string]SubjectDriver{
	"check": {
		"throws":               func(tb assert.TB, s *Subject, m string) { assert.Panics(tb, s.Bare, m) },
		"not-throws":           func(tb assert.TB, s *Subject, m string) { assert.NotPanics(tb, s.Bare, m) },
		"honours-cancellation": func(tb assert.TB, s *Subject, m string) { assert.HonoursCancellation(tb, s.Ctx, m) },
		"honours-deadline":     func(tb assert.TB, s *Subject, m string) { assert.HonoursDeadline(tb, s.Ctx, m) },
		"nil-context-safe":     func(tb assert.TB, s *Subject, m string) { assert.NilContextSafe(tb, s.Ctx, m) },
		"pure":                 func(tb assert.TB, s *Subject, m string) { assert.Pure(tb, s.Observe, s.Bare, m) },
		"eventually": func(tb assert.TB, s *Subject, m string) {
			assert.Eventually(tb, retryTimeout, retryInterval, s.Seated, m)
		},
		"eventually-true": func(tb assert.TB, s *Subject, m string) {
			assert.EventuallyTrue(tb, retryTimeout, flips(s), m)
		},
		"idempotent": func(tb assert.TB, s *Subject, m string) {
			assert.Idempotent(tb, s.Call, s.Input, s.Observe, m)
		},
		"accumulates": func(tb assert.TB, s *Subject, m string) {
			assert.Accumulates(tb, s.Call, s.Input, s.Observe, m)
		},
		"deterministic": func(tb assert.TB, s *Subject, m string) { assert.Deterministic(tb, s.Compute, s.Input, m) },
		"commutative":   func(tb assert.TB, s *Subject, m string) { assert.Commutative(tb, s.Combine, s.A, s.B, m) },
		"associative": func(tb assert.TB, s *Subject, m string) {
			assert.Associative(tb, s.Combine, s.A, s.B, s.C, m)
		},
		"round-trip": func(tb assert.TB, s *Subject, m string) {
			assert.RoundTrip(tb, s.Forward, s.Inverse, s.Input, m)
		},
		"stable-order":  func(tb assert.TB, s *Subject, m string) { assert.StableOrder(tb, s.Iterate, m) },
		"no-duplicates": func(tb assert.TB, s *Subject, m string) { assert.NoDuplicates(tb, s.Iterate, m) },
		"monotonic": func(tb assert.TB, s *Subject, m string) {
			assert.Monotonic(tb, s.Observe, s.Advance, s.Steps, m)
		},
		"total":    func(tb assert.TB, s *Subject, m string) { assert.Total(tb, s.Call, s.Domain, m) },
		"not-pure": func(tb assert.TB, s *Subject, m string) { assert.NotPure(tb, s.Observe, s.Bare, m) },
		"after-close": func(tb assert.TB, s *Subject, m string) {
			assert.FailsAfterClose(tb, s.Closer, s.Use, s.Sentinel, m)
		},
		"poisoned": func(tb assert.TB, s *Subject, m string) { assert.Poisoned(tb, s.Induce, s.Read, m) },
	},
	"expect": {
		"throws":               func(tb assert.TB, s *Subject, m string) { expect.Panics(tb, s.Bare, m) },
		"not-throws":           func(tb assert.TB, s *Subject, m string) { expect.NotPanics(tb, s.Bare, m) },
		"honours-cancellation": func(tb assert.TB, s *Subject, m string) { expect.HonoursCancellation(tb, s.Ctx, m) },
		"honours-deadline":     func(tb assert.TB, s *Subject, m string) { expect.HonoursDeadline(tb, s.Ctx, m) },
		"nil-context-safe":     func(tb assert.TB, s *Subject, m string) { expect.NilContextSafe(tb, s.Ctx, m) },
		"pure":                 func(tb assert.TB, s *Subject, m string) { expect.Pure(tb, s.Observe, s.Bare, m) },
		"eventually": func(tb assert.TB, s *Subject, m string) {
			expect.Eventually(tb, retryTimeout, retryInterval, s.Seated, m)
		},
		"eventually-true": func(tb assert.TB, s *Subject, m string) {
			expect.EventuallyTrue(tb, retryTimeout, flips(s), m)
		},
		"idempotent": func(tb assert.TB, s *Subject, m string) {
			expect.Idempotent(tb, s.Call, s.Input, s.Observe, m)
		},
		"accumulates": func(tb assert.TB, s *Subject, m string) {
			expect.Accumulates(tb, s.Call, s.Input, s.Observe, m)
		},
		"deterministic": func(tb assert.TB, s *Subject, m string) { expect.Deterministic(tb, s.Compute, s.Input, m) },
		"commutative":   func(tb assert.TB, s *Subject, m string) { expect.Commutative(tb, s.Combine, s.A, s.B, m) },
		"associative": func(tb assert.TB, s *Subject, m string) {
			expect.Associative(tb, s.Combine, s.A, s.B, s.C, m)
		},
		"round-trip": func(tb assert.TB, s *Subject, m string) {
			expect.RoundTrip(tb, s.Forward, s.Inverse, s.Input, m)
		},
		"stable-order":  func(tb assert.TB, s *Subject, m string) { expect.StableOrder(tb, s.Iterate, m) },
		"no-duplicates": func(tb assert.TB, s *Subject, m string) { expect.NoDuplicates(tb, s.Iterate, m) },
		"monotonic": func(tb assert.TB, s *Subject, m string) {
			expect.Monotonic(tb, s.Observe, s.Advance, s.Steps, m)
		},
		"total":    func(tb assert.TB, s *Subject, m string) { expect.Total(tb, s.Call, s.Domain, m) },
		"not-pure": func(tb assert.TB, s *Subject, m string) { expect.NotPure(tb, s.Observe, s.Bare, m) },
		"after-close": func(tb assert.TB, s *Subject, m string) {
			expect.FailsAfterClose(tb, s.Closer, s.Use, s.Sentinel, m)
		},
		"poisoned": func(tb assert.TB, s *Subject, m string) { expect.Poisoned(tb, s.Induce, s.Read, m) },
	},
}

// flips returns a predicate that reads the subject's seated shape, so
// one behaviour serves both retrying assertions.
func flips(held *Subject) func() bool {
	return func() bool {
		trial := assert.NewRecorder()
		held.Seated(trial)
		return !trial.Failed()
	}
}

// RunSubject drives one subject case, and reports whether it ran.
//
// It reports false when this language builds no such behaviour or the
// assertion takes none. The corpus runner fails such a case, because only
// a skip that the definition states excuses one.
func RunSubject(surface, assertion, kind string, tb assert.TB, msg string) bool {
	build, buildable := Subjects[kind]
	drive, driven := SubjectDrivers[surface][assertion]
	if !buildable || !driven {
		return false
	}
	drive(tb, build(), msg)
	return true
}
