// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestCorpus runs every case of the definition's corpus against this
// library, in every form that the registry drives, and compares each
// assertion's outcome and record with the ones the case states. The
// completeness gate checks that an assertion exists, and this test checks
// what it reports.
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
	overlay, err := conformance.Overlay()
	if err != nil {
		t.Fatalf("the overlay can be read: %v", err)
	}

	for id, cases := range byAssertion {
		// Every assertion that a case states values for has invokers. One
		// that the cases cover only by naming a behaviour is driven through
		// the subject registry instead, and needs none.
		forms, registered := conformance.Registry[id]
		if !registered && !statesOnlySubjects(cases) {
			t.Fatalf("an invoker is registered for %s", id)
		}

		for _, tc := range cases {
			t.Run(tc.ID, func(t *testing.T) {
				t.Parallel()

				if why, skipped := tc.SkipReason(); skipped {
					t.Skipf("declared skip: %s", why)
				}
				opts := options(t, tc, overlay)
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

				for _, form := range slices.Sorted(maps.Keys(forms)) {
					r := assert.NewRecorder()
					forms[form](r, args, tc.ID, opts)

					if err := tc.Check(r); err != nil {
						t.Errorf("%s: the outcome is what the case states: %v", form, err)
					}
					checkWhere(t, r)
				}
			})
		}
	}
}

// options returns the options of the relaxations that a case names. It
// skips the case when the overlay declines one of them, and fails it when
// no option is registered for one.
func options(t *testing.T, tc conformance.Case, overlay conformance.OverlayDoc) []assert.Option {
	t.Helper()

	var out []assert.Option
	for _, id := range tc.Options {
		if overlay.DeclinesRelaxation(id) {
			t.Skipf("declared skip: the overlay declines the relaxation %s", id)
		}
		option, offered := conformance.Relaxations[id]
		if !offered {
			t.Fatalf("no option is registered for the relaxation %s", id)
		}
		out = append(out, option)
	}
	return out
}

// runSubjectCase runs a case that names a behaviour through both surfaces,
// and compares each outcome with the one the case states. A kind that this
// language cannot build, or an assertion that no driver calls, fails the
// case, because only a skip in the definition excuses a case.
func runSubjectCase(t *testing.T, tc conformance.Case) {
	t.Helper()

	if len(tc.Options) > 0 {
		t.Fatalf("the case names the options %v, and no subject driver passes options", tc.Options)
	}
	for _, surface := range []string{"check", "expect"} {
		r := assert.NewRecorder().WithClock(assert.NewControlled(time.Time{}))
		if !conformance.RunSubject(surface, tc.Assertion, tc.Subject.Kind, r, tc.ID) {
			t.Fatalf("%s: no subject named %q, or no driver of %s, and the case declares no skip",
				surface, tc.Subject.Kind, tc.Assertion)
		}
		if err := tc.Check(r); err != nil {
			t.Errorf("%s: the outcome is what the case states: %v", surface, err)
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

// checkWhere fails t unless every record of r names a line of this file. A
// record names the innermost frame of a test file, and this file is the
// innermost test file under every call. The frames between the assertion
// and this file are of the registry and the subject drivers, and neither
// is a test file.
func checkWhere(t *testing.T, r *assert.Recorder) {
	t.Helper()

	for _, held := range r.Failures() {
		if filepath.Base(held.Where.File) != "corpus_test.go" || held.Where.Line == 0 {
			t.Errorf("%s points at %s:%d, want a line of corpus_test.go",
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

		t.Run("sets each case's fields to the detail fields of its assertion", func(t *testing.T) {
			t.Parallel()

			byAssertion, err := conformance.Cases()
			if err != nil {
				t.Fatalf("the corpus can be read: %v", err)
			}
			assertions, err := conformance.Assertions()
			if err != nil {
				t.Fatalf("the assertion table can be read: %v", err)
			}
			for id, cases := range byAssertion {
				declared := assertions[id].DetailFields
				for _, c := range cases {
					if !slices.Equal(c.Fields, declared) {
						t.Fatalf("case %s states the fields %q, want %q", c.ID, c.Fields, declared)
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

			c := conformance.Case{
				ID: "x", Assertion: "equal", Expect: "fail", Fields: []string{"want"},
				Detail: detail(`{"want": {"type":"widget"}}`),
			}
			if err := c.Check(r); !errors.Is(err, conformance.ErrUnknownType) {
				t.Fatalf("Check returns %v, want ErrUnknownType", err)
			}
		})

		t.Run("returns an error for a record of another assertion", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{Assertion: "not-equal", Contract: "x"}, true)

			c := conformance.Case{ID: "x", Assertion: "equal", Expect: "fail"}
			if c.Check(r) == nil {
				t.Fatal("a record naming another assertion is refused")
			}
		})

		t.Run("returns an error for a record whose contract is not the message", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{Assertion: "equal", Contract: "x, reworded"}, true)

			c := conformance.Case{ID: "x", Assertion: "equal", Expect: "fail"}
			if c.Check(r) == nil {
				t.Fatal("a record whose contract differs from the message is refused")
			}
		})

		t.Run("returns an error for a record of a field that the assertion does not declare", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{
				Assertion: "equal", Contract: "x",
				Detail: map[string]any{"want": 1, "got": 2, "diff": "-1 +2"},
			}, true)

			c := conformance.Case{ID: "x", Assertion: "equal", Expect: "fail", Fields: []string{"want", "got"}}
			if c.Check(r) == nil {
				t.Fatal("a record of an undeclared field is refused")
			}
		})

		t.Run("returns an error for a record without a declared field", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Report(assert.Failure{
				Assertion: "equal", Contract: "x",
				Detail: map[string]any{"want": 1},
			}, true)

			c := conformance.Case{ID: "x", Assertion: "equal", Expect: "fail", Fields: []string{"want", "got"}}
			if c.Check(r) == nil {
				t.Fatal("a record without a declared field is refused")
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

			c := conformance.Case{
				ID: "x", Assertion: "equal", Expect: "fail", Fields: []string{"got"},
				Detail: detail(`{"want": {"type":"int","value":1}}`),
			}
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

			c := conformance.Case{
				ID: "x", Assertion: "equal", Expect: "fail", Fields: []string{"want"},
				Detail: detail(`{"want": {"type":"int","value":1}}`),
			}
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
				ID: "x", Assertion: "equal", Expect: "fail", Fields: []string{"want", "got"},
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
