// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"slices"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestFunctions checks that each form of the registry and each surface of
// the subject drivers calls the function of its own surface: an aborting
// form reports a failure as fatal, and a recording form as a failure that
// the test continues after. A record does not state the surface that
// reported it, so the corpus cannot tell the two apart. It drives the
// first failing case of each assertion. Written with testing rather than
// with this library, because a verdict is not written with the subject.
func TestFunctions(t *testing.T) {
	t.Parallel()

	byAssertion := conformance.Cases()
	aborting := map[conformance.Form]bool{
		conformance.AbortingCall:   true,
		conformance.AbortingChain:  true,
		conformance.RecordingCall:  false,
		conformance.RecordingChain: false,
	}

	t.Run("Registry", func(t *testing.T) {
		t.Parallel()

		for id, invokers := range conformance.Registry {
			tc, ok := firstFailing(byAssertion[id], false)
			if !ok {
				t.Fatalf("the corpus states no failing case of %s with values and without options", id)
			}
			t.Run("reports a failure of "+string(id)+" through the surface of each form", func(t *testing.T) {
				t.Parallel()
				args, err := tc.Decoded()
				if err != nil {
					t.Fatalf("the case's arguments decode: %v", err)
				}
				for form, invoke := range invokers {
					r := assert.NewRecorder()
					invoke(r, args, tc.ID, nil)
					if got := fatal(t, r); got != aborting[form] {
						t.Errorf("%s reports its failure as fatal: %t, want %t", form, got, aborting[form])
					}
				}
			})
		}
	})

	t.Run("SubjectDrivers", func(t *testing.T) {
		t.Parallel()

		surfaces := map[string]bool{"check": true, "expect": false}
		for assertion := range conformance.SubjectDrivers["check"] {
			tc, ok := firstFailing(byAssertion[conformance.ID(assertion)], true)
			if !ok {
				t.Fatalf("the corpus states no failing case of %s that names a subject", assertion)
			}
			t.Run("reports a failure of "+assertion+" through each surface", func(t *testing.T) {
				t.Parallel()
				for surface, abort := range surfaces {
					r := assert.NewRecorder().WithClock(assert.NewControlled(time.Time{}))
					conformance.RunSubject(surface, assertion, tc.Subject.Kind, r, tc.ID)
					if got := fatal(t, r); got != abort {
						t.Errorf("%s reports its failure as fatal: %t, want %t", surface, got, abort)
					}
				}
			})
		}
	})
}

// firstFailing returns the first case of cases that expects a failure,
// states no options and no skip of Go, and names a subject when subject is
// set, or states values when it is not.
func firstFailing(cases []conformance.Case, subject bool) (conformance.Case, bool) {
	i := slices.IndexFunc(cases, func(c conformance.Case) bool {
		_, skipped := c.SkipReason()
		return c.Expect == "fail" && len(c.Options) == 0 && !skipped && (c.Subject.Kind != "") == subject
	})
	if i < 0 {
		return conformance.Case{}, false
	}
	return cases[i], true
}

// fatal reports whether r recorded its failure as fatal: a failure that no
// message of Errorf states. It fails t when r recorded no failure.
func fatal(t *testing.T, r *assert.Recorder) bool {
	t.Helper()
	if !r.Failed() {
		t.Fatal("the failing case reported no failure")
	}
	return len(r.Messages()) == 0
}
