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
// The assertions taking a callable differ in shape, so each says how it
// is called rather than the corpus runner knowing all of them.
type SubjectDriver func(tb assert.TB, held *Subject, msg string)

// aborting and recording name the two surfaces a subject case is driven
// through, the same pair every other case is driven through.
const (
	retryTimeout  = time.Hour
	retryInterval = time.Minute
)

// SubjectDrivers says how each subject-taking assertion is called, by
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
	},
}

// flips answers a predicate that reads the subject's seated shape, so
// one behaviour serves both retrying assertions.
func flips(held *Subject) func() bool {
	return func() bool {
		trial := assert.NewRecorder()
		held.Seated(trial)
		return !trial.Failed()
	}
}

// RunSubject drives one subject case, answering whether it ran.
//
// It answers false when this language builds no such behaviour or the
// assertion takes none, which the corpus runner reports as a skip.
func RunSubject(surface, assertion, kind string, tb assert.TB, msg string) bool {
	build, buildable := Subjects[kind]
	drive, driven := SubjectDrivers[surface][assertion]
	if !buildable || !driven {
		return false
	}
	drive(tb, build(), msg)
	return true
}
