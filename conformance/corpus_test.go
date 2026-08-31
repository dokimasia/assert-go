// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestCorpus drives every case the definition states against this
// library. It is what checks meaning rather than membership: the
// completeness gate says an assertion exists, and this says it answers
// what the standard says it should.
//
// Written with testing rather than with this library. Every assertion
// reports through one function, so a verdict written with the subject
// goes quiet exactly when the subject does: silencing that function
// leaves every case passing, having checked nothing.
// detail reads a case's detail block from the JSON a corpus file
// would carry, so a test states it the way the corpus does.
func detail(raw string) map[string]json.RawMessage {
	var out map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		panic(err)
	}
	return out
}

func TestCorpus(t *testing.T) {
	t.Parallel()

	byAssertion, err := conformance.Cases()
	if err != nil {
		t.Fatalf("the corpus can be read: %v", err)
	}
	if len(byAssertion) == 0 {
		t.Fatal("the corpus states something, but it read no assertions")
	}

	for id, cases := range byAssertion {
		// Every assertion the corpus reaches with values has an invoker.
		// One it reaches only by naming a behaviour is driven through
		// the subject registry instead, and needs none.
		invoke, registered := conformance.Registry[id]
		if !registered && !statesOnlySubjects(cases) {
			t.Fatalf("an invoker is registered for %s", id)
		}

		for _, tc := range cases {
			t.Run(tc.ID, func(t *testing.T) {
				t.Parallel()

				if why, skipped := tc.SkipReason(); skipped {
					t.Skipf("declared skip: %s", why)
				}
				if tc.Subject.Kind != "" {
					runSubjectCase(t, tc)
					return
				}

				if !registered {
					t.Fatalf("no invoker for %s and not every case names a subject", id)
				}

				args, err := tc.Decoded()
				if err != nil {
					t.Fatalf("the case's arguments decode: %v", err)
				}

				r := assert.NewRecorder()
				invoke(r, args, tc.ID)

				if err := tc.Check(r); err != nil {
					t.Fatalf("the outcome is what the case states: %v", err)
				}
				checkWhere(t, r)
			})
		}
	}
}

// runSubjectCase drives a case that names a behaviour, through both
// surfaces, and holds the outcome to what the case states.
//
// A kind this language cannot build is a skip, which is what the
// standard states for one an implementation cannot make.
func runSubjectCase(t *testing.T, tc conformance.Case) {
	t.Helper()

	for _, surface := range []string{"check", "expect"} {
		r := assert.NewRecorder().WithClock(assert.NewControlled(time.Time{}))
		if !conformance.RunSubject(surface, tc.Assertion, tc.Subject.Kind, r, tc.ID) {
			t.Skipf("no subject named %q on %s", tc.Subject.Kind, surface)
		}
		if err := tc.Check(r); err != nil {
			t.Fatalf("%s: the outcome is what the case states: %v", surface, err)
		}
		checkWhere(t, r)
	}
}

// statesOnlySubjects reports whether every case for an assertion names
// a behaviour rather than stating values.
func statesOnlySubjects(cases []conformance.Case) bool {
	for _, one := range cases {
		if one.Subject.Kind == "" {
			return false
		}
	}
	return true
}

// checkWhere holds every reported record to naming a real call site
// outside the library's own reporting code.
//
// A case cannot state a line: the line is wherever the caller put the
// call, which here is the registry. What every case can state is that
// the record points somewhere a reader can open, and never at the
// machinery that built it. Both call-site bugs this standard has found
// were of that shape.
func checkWhere(t *testing.T, r *assert.Recorder) {
	t.Helper()

	for _, held := range r.Failures() {
		switch {
		case held.Where.File == "":
			t.Fatalf("%s reported no call site", held.Assertion)
		case held.Where.Line == 0:
			t.Fatalf("%s reported line zero, want the caller's line", held.Assertion)
		case strings.Contains(held.Where.File, "/internal/matcher/"):
			t.Fatalf("%s points at %s:%d, which is the library reporting its own frame",
				held.Assertion, held.Where.File, held.Where.Line)
		}
	}
}

// TestCorpusRules drives the corpus reader's own rules, which the
// cases cannot: a corpus that passes every case still has to refuse a
// case it does not understand.
func TestCorpusRules(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		t.Run("an unknown expectation is refused", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "made-up", Expect: "maybe"}
			if c.Check(assert.NewRecorder()) == nil {
				t.Fatal("a case stating neither pass nor fail is refused")
			}
		})

		t.Run("a failure with no record is refused", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("a message reported without a record")

			c := conformance.Case{ID: "x", Expect: "fail"}
			if c.Check(r) == nil {
				t.Fatal("a failure reporting no record is refused")
			}
		})

		t.Run("a record missing a stated detail field is refused", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{
				Assertion: "equal", Contract: "x",
				Detail: map[string]any{"got": 2},
			}, true)

			c := conformance.Case{ID: "x", Expect: "fail", Detail: detail(`{"want": {"type":"int","value":1}}`)}
			if c.Check(r) == nil {
				t.Fatal("a record holding none of the stated want is refused")
			}
		})

		t.Run("a record holding a different value is refused", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{
				Assertion: "equal", Contract: "x",
				Detail: map[string]any{"want": 9},
			}, true)

			c := conformance.Case{ID: "x", Expect: "fail", Detail: detail(`{"want": {"type":"int","value":1}}`)}
			if c.Check(r) == nil {
				t.Fatal("a record whose want differs from the case is refused")
			}
		})

		t.Run("a record matching the case is accepted", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{
				Assertion: "equal", Contract: "x",
				Detail: map[string]any{"want": 1, "got": 2},
			}, true)

			c := conformance.Case{
				ID: "x", Expect: "fail",
				Detail: detail(`{"want": {"type":"int","value":1}, "got": {"type":"int","value":2}}`),
			}
			if err := c.Check(r); err != nil {
				t.Fatalf("a record matching the case is accepted: %v", err)
			}
		})
	})

	t.Run("Decoded", func(t *testing.T) {
		t.Parallel()

		t.Run("a literal it cannot read is refused", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "x", Args: []json.RawMessage{[]byte(`{"type":"widget"}`)}}
			if _, err := c.Decoded(); err == nil {
				t.Fatal("an argument the encoding does not cover is refused")
			}
		})
	})

	t.Run("SkipReason", func(t *testing.T) {
		t.Parallel()

		t.Run("a case with no skip table applies", func(t *testing.T) {
			t.Parallel()

			if _, skipped := (conformance.Case{ID: "x"}).SkipReason(); skipped {
				t.Fatal("a case naming no skip applies to this language")
			}
		})

		t.Run("a skip for another language does not apply here", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "x", Skip: map[string]string{"php": "no generics"}}
			if _, skipped := c.SkipReason(); skipped {
				t.Fatal("a skip naming another language does not apply here")
			}
		})
	})
}
