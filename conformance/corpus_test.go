// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestCorpus runs every case of the definition's corpus against this
// library, and compares each assertion's outcome and record with the ones
// the case states. The completeness gate checks that an assertion exists,
// and this test checks what it reports.
//
// Written with testing rather than with this library. Every assertion
// reports through one function, so a verdict written with the subject goes
// quiet when the subject does: a silenced reporting function leaves every
// case passing, having checked nothing.
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

// runSubjectCase runs a case that names a behaviour through both surfaces,
// and compares each outcome with the one the case states. A kind that this
// language cannot build skips the case, as the standard states for a
// behaviour that an implementation cannot make.
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

// statesOnlySubjects reports whether every case of an assertion names a
// behaviour instead of stating values.
func statesOnlySubjects(cases []conformance.Case) bool {
	for _, one := range cases {
		if one.Subject.Kind == "" {
			return false
		}
	}
	return true
}

// checkWhere fails t unless every record of r names a call site outside the
// library's own reporting code: a file, and a line above zero. A case
// cannot state the line, which is wherever the registry calls the
// assertion, so the check is that the record points at a file a reader can
// open and never at the code that built the record.
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

// TestCorpusRules drives the rules of the corpus reader that the cases
// cannot: a reader that passes every case still refuses a case it does not
// understand.
func TestCorpusRules(t *testing.T) {
	t.Parallel()

	t.Run("Cases", func(t *testing.T) {
		t.Parallel()

		t.Run("sets each case's assertion to the assertion of its file", func(t *testing.T) {
			t.Parallel()

			byAssertion, err := conformance.Cases()
			if err != nil {
				t.Fatalf("the corpus can be read: %v", err)
			}
			for id, cases := range byAssertion {
				for _, c := range cases {
					if c.Assertion != string(id) {
						t.Fatalf("case %s states the assertion %q, want %q", c.ID, c.Assertion, id)
					}
				}
			}
		})
	})

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		t.Run("returns an error for an unknown expectation", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "made-up", Expect: "maybe"}
			if c.Check(assert.NewRecorder()) == nil {
				t.Fatal("a case stating neither pass nor fail is refused")
			}
		})

		t.Run("returns an error for a failure of a case that expects a pass", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Errorf("a message reported without a record")

			c := conformance.Case{ID: "x", Expect: "pass"}
			if c.Check(r) == nil {
				t.Fatal("a failure of a case that expects a pass is refused")
			}
		})

		t.Run("returns an error for a pass of a case that expects a failure", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "x", Expect: "fail"}
			if c.Check(assert.NewRecorder()) == nil {
				t.Fatal("a pass of a case that expects a failure is refused")
			}
		})

		t.Run("returns an error for a detail field of a literal of an unknown type", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{Assertion: "equal", Contract: "x", Detail: map[string]any{"want": 1}}, true)

			c := conformance.Case{ID: "x", Expect: "fail", Detail: detail(`{"want": {"type":"widget"}}`)}
			if err := c.Check(r); !errors.Is(err, conformance.ErrUnknownType) {
				t.Fatalf("Check returns %v, want ErrUnknownType", err)
			}
		})

		t.Run("returns an error for a failure without a record", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("a message reported without a record")

			c := conformance.Case{ID: "x", Expect: "fail"}
			if c.Check(r) == nil {
				t.Fatal("a failure reporting no record is refused")
			}
		})

		t.Run("returns an error for a record without a stated detail field", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{
				Assertion: "equal", Contract: "x",
				Detail: map[string]any{"got": 2},
			}, true)

			c := conformance.Case{ID: "x", Expect: "fail", Detail: detail(`{"want": {"type":"int","value":1}}`)}
			if c.Check(r) == nil {
				t.Fatal("a record without the stated want is refused")
			}
		})

		t.Run("returns an error for a detail field of another value", func(t *testing.T) {
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

		t.Run("returns nil for a record that matches the case", func(t *testing.T) {
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

		t.Run("returns an error for an argument of an unknown type", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "x", Args: []json.RawMessage{[]byte(`{"type":"widget"}`)}}
			if _, err := c.Decoded(); err == nil {
				t.Fatal("an argument the encoding does not cover is refused")
			}
		})
	})

	t.Run("SkipReason", func(t *testing.T) {
		t.Parallel()

		t.Run("reports false for a case without a skip table", func(t *testing.T) {
			t.Parallel()

			if _, skipped := (conformance.Case{ID: "x"}).SkipReason(); skipped {
				t.Fatal("a case naming no skip applies to this language")
			}
		})

		t.Run("reports false for a skip of another language", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "x", Skip: map[string]string{"php": "no generics"}}
			if _, skipped := c.SkipReason(); skipped {
				t.Fatal("a skip naming another language does not apply here")
			}
		})
	})
}

// detail returns the detail block of a case from the JSON text that a
// corpus file states it in.
func detail(raw string) map[string]json.RawMessage {
	var out map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		panic(err)
	}
	return out
}
