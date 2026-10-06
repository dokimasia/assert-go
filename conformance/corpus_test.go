// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/prop"
)

// TestCorpus runs every case of the definition's corpus against this
// library, in every form that the registry drives, and compares each
// assertion's outcome and record with the ones the case states. The
// completeness gate checks that an assertion exists, and this test checks
// what it reports. It also drives the rules of a case that the corpus
// cannot: a reader that passes every case still refuses a case it does not
// understand.
//
// Written with testing rather than with this library. Every assertion
// reports through one function, so a verdict written with the subject goes
// quiet when the subject does: a silenced reporting function leaves every
// case passing, having checked nothing.
func TestCorpus(t *testing.T) {
	t.Parallel()

	byAssertion := conformance.Cases()
	if len(byAssertion) == 0 {
		t.Fatal("the corpus states something, but it read no assertions")
	}

	t.Run("Cases", func(t *testing.T) {
		t.Parallel()

		t.Run("sets each case's assertion to the assertion of its file", func(t *testing.T) {
			t.Parallel()
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
			assertions := conformance.Assertions()
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

		overlay := conformance.Overlay()
		for id, cases := range byAssertion {
			// Every assertion that a case states values for has invokers. One
			// that the cases cover only by naming a behaviour is driven through
			// the subject registry instead, and needs none.
			forms, registered := conformance.Registry[id]
			if !registered && !statesOnlySubjects(cases) {
				t.Fatalf("an invoker is registered for %s", id)
			}

			for _, tc := range cases {
				t.Run("returns nil for case "+tc.ID, func(t *testing.T) {
					t.Parallel()

					if why, skipped := tc.SkipReason(); skipped {
						t.Skipf("declared skip: %s", why)
					}
					opts := options(t, tc, overlay)
					if tc.Subject.Kind != "" {
						runSubjectCase(t, tc, opts)
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
						aborting := form == conformance.AbortingCall || form == conformance.AbortingChain
						if err := tc.Check(r, aborting); err != nil {
							t.Errorf("%s: the outcome is what the case states: %v", form, err)
						}
						checkWhere(t, r)
					}
				})
			}
		}

		failing := func(f assert.Failure) *assert.Recorder {
			r := assert.NewRecorder()
			r.Report(f, true)
			return r
		}
		atDetail := func(field string) fault.Path {
			return inVector(fault.Field(detailAt), fault.Key(field))
		}
		tests := []struct {
			name         string
			give         conformance.Case
			giveSeat     *assert.Recorder
			giveAborting bool
			wantPath     fault.Path
			wantReason   string
		}{
			{
				name: "returns nil for a record and a call record that match the case",
				give: conformance.Case{
					ID: builtID, Assertion: "equal", Expect: "fail", Fields: []string{"want", "got"},
					Detail: detail(`{"want": {"type":"int","value":1}, "got": {"type":"int","value":2}}`),
				},
				giveSeat:     called(func(r *assert.Recorder) { assert.Equal(r, 2, 1, builtID) }),
				giveAborting: true,
			},
			{
				name:       "returns a fault at expect for an unknown expectation",
				give:       conformance.Case{ID: builtID, Expect: "maybe"},
				giveSeat:   assert.NewRecorder(),
				wantPath:   inVector(fault.Field("expect")),
				wantReason: `"maybe" is neither pass nor fail`,
			},
			{
				name:       "returns a fault at expect for a pass of a case that expects a failure",
				give:       conformance.Case{ID: builtID, Expect: "fail"},
				giveSeat:   assert.NewRecorder(),
				wantPath:   inVector(fault.Field("expect")),
				wantReason: "the assertion passes",
			},
			{
				name:       "returns a fault for a record of another assertion",
				give:       conformance.Case{ID: builtID, Assertion: "equal", Expect: "fail"},
				giveSeat:   failing(assert.Failure{Assertion: "not-equal", Contract: builtID}),
				wantPath:   inVector(),
				wantReason: `the record is of the assertion "not-equal", want "equal"`,
			},
			{
				name:     "returns a fault for a record whose contract is not the message",
				give:     conformance.Case{ID: builtID, Assertion: "equal", Expect: "fail"},
				giveSeat: failing(assert.Failure{Assertion: "equal", Contract: builtID + ", reworded"}),
				wantPath: inVector(),
				wantReason: `the record states the contract "` + builtID + `, reworded", want the message "` +
					builtID + `"`,
			},
			{
				name: "returns a fault at the detail for a record of a field that the assertion does not declare",
				give: conformance.Case{
					ID:        builtID,
					Assertion: "equal",
					Expect:    "fail",
					Fields:    []string{"want", "got"},
				},
				giveSeat: failing(assert.Failure{
					Assertion: "equal", Contract: builtID,
					Detail: map[string]any{"want": 1, "got": 2, "diff": "-1 +2"},
				}),
				wantPath:   inVector(fault.Field(detailAt)),
				wantReason: `the record states the fields ["diff" "got" "want"], want ["got" "want"]`,
			},
			{
				name: "returns a fault at the detail for a record without a declared field",
				give: conformance.Case{
					ID:        builtID,
					Assertion: "equal",
					Expect:    "fail",
					Fields:    []string{"want", "got"},
				},
				giveSeat: failing(assert.Failure{
					Assertion: "equal", Contract: builtID,
					Detail: map[string]any{"want": 1},
				}),
				wantPath:   inVector(fault.Field(detailAt)),
				wantReason: `the record states the fields ["want"], want ["got" "want"]`,
			},
			{
				name: "returns a fault at the field for a record without a stated detail field",
				give: conformance.Case{
					ID: builtID, Assertion: "equal", Expect: "fail", Fields: []string{"got"},
					Detail: detail(`{"want": {"type":"int","value":1}}`),
				},
				giveSeat: failing(
					assert.Failure{Assertion: "equal", Contract: builtID, Detail: map[string]any{"got": 2}},
				),
				wantPath:   atDetail("want"),
				wantReason: `the record states no such field, want {"type":"int","value":1}`,
			},
			{
				name: "returns a fault at the field for a detail field of another value",
				give: conformance.Case{
					ID: builtID, Assertion: "equal", Expect: "fail", Fields: []string{"want"},
					Detail: detail(`{"want": {"type":"int","value":1}}`),
				},
				giveSeat: failing(
					assert.Failure{Assertion: "equal", Contract: builtID, Detail: map[string]any{"want": 9}},
				),
				wantPath:   atDetail("want"),
				wantReason: `the field is int:9, want {"type":"int","value":1}`,
			},
			{
				name:       "returns a fault for a recorder without a call record",
				give:       conformance.Case{ID: builtID, Assertion: "true", Expect: "pass"},
				giveSeat:   assert.NewRecorder(),
				wantPath:   inVector(),
				wantReason: "the recorder keeps 0 call records, want 1",
			},
			{
				name:       "returns a fault for a call record of the other surface",
				give:       conformance.Case{ID: builtID, Assertion: "true", Expect: "pass"},
				giveSeat:   called(func(r *assert.Recorder) { assert.True(r, true, builtID) }),
				wantPath:   inVector(),
				wantReason: `the call record is ` + passedTrue(true) + `, want ` + passedTrue(false),
			},
			{
				name:         "returns a fault at the detail for a call record of a pass that states a detail",
				give:         conformance.Case{ID: builtID, Assertion: "prop-for-all", Expect: "pass"},
				giveSeat:     called(func(r *assert.Recorder) { prop.ForAll(r, builtID, drawsDigit, prop.Seed(7)) }),
				giveAborting: true,
				wantPath:     inVector(fault.Field(detailAt)),
				wantReason: `the call record of a pass states the detail {"outcome":"passed","cases":10,` +
					`"rejected":0,"seed":"7","counterexample":null,"failure":null,"choices":null,"others":null,` +
					`"divergence":null,"coverage":null}`,
			},
			{
				name: "returns a fault at the detail for a call record of other fields than the case's",
				give: conformance.Case{
					ID: builtID, Assertion: "equal", Expect: "fail", Fields: []string{"want", "got", "extra"},
				},
				giveSeat: called(func(r *assert.Recorder) {
					r.Report(assert.Failure{
						Assertion: "equal", Contract: builtID, Detail: map[string]any{"want": 1, "got": 2, "extra": 3},
					}, false)
					expect.Equal(r, 2, 1, builtID)
				}),
				wantPath:   inVector(fault.Field(detailAt)),
				wantReason: `the call record states the fields ["got" "want"], want ["extra" "got" "want"]`,
			},
			{
				name: "returns a fault at the field for a call record of another value",
				give: conformance.Case{
					ID: builtID, Assertion: "equal", Expect: "fail", Fields: []string{"want", "got"},
					Detail: detail(`{"want": {"type":"int","value":1}}`),
				},
				giveSeat: called(func(r *assert.Recorder) {
					r.Report(assert.Failure{
						Assertion: "equal", Contract: builtID, Detail: map[string]any{"want": 1, "got": 2},
					}, false)
					expect.Equal(r, 2, 3, builtID)
				}),
				wantPath:   atDetail("want"),
				wantReason: `the call record states {"type":"int","value":3}, want {"type":"int","value":1}`,
			},
			{
				name: "returns a fault at the field for a call record that states no typed literal of it",
				give: conformance.Case{
					ID: builtID, Assertion: "equal", Expect: "fail", Fields: []string{"want", "got"},
					Detail: detail(`{"got": {"type":"int","value":2}}`),
				},
				giveSeat: called(func(r *assert.Recorder) {
					r.Report(assert.Failure{
						Assertion: "equal", Contract: builtID, Detail: map[string]any{"want": 1, "got": 2},
					}, false)
					expect.Equal[any](r, errors.New("the ledger is closed"), 1, builtID)
				}),
				wantPath:   atDetail("got"),
				wantReason: "the call record states no typed literal of the field",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, tt.give.Check(tt.giveSeat, tt.giveAborting), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns a fault at expect for a failure of a case that expects a pass", func(t *testing.T) {
			t.Parallel()
			r := assert.NewRecorder()
			r.Errorf("a message reported without a record")
			c := conformance.Case{ID: builtID, Expect: "pass"}
			expectFault(t, c.Check(r, false), inVector(fault.Field("expect")),
				"the assertion fails: a message reported without a record")
		})

		t.Run("returns a fault for a failure without a record", func(t *testing.T) {
			t.Parallel()
			r := assert.NewRecorder()
			r.Fatalf("a message reported without a record")
			c := conformance.Case{ID: builtID, Expect: "fail"}
			expectFault(t, c.Check(r, false), inVector(), "the assertion fails without a record")
		})

		t.Run("returns a fault at the field for a detail field of a literal of an unknown type", func(t *testing.T) {
			t.Parallel()
			c := conformance.Case{
				ID: builtID, Assertion: "equal", Expect: "fail", Fields: []string{"want"},
				Detail: detail(`{"want": ` + widget + `}`),
			}
			err := c.Check(
				failing(assert.Failure{Assertion: "equal", Contract: builtID, Detail: map[string]any{"want": 1}}),
				false,
			)
			expectFault(t, err, append(atDetail("want"), fault.Field(typeAt)), unknownWidget)
			if !errors.Is(err, literal.ErrUnknownType) {
				t.Fatalf("Check returns %v, want ErrUnknownType", err)
			}
		})
	})

	t.Run("Decoded", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a fault at the argument for an argument of an unknown type", func(t *testing.T) {
			t.Parallel()
			c := conformance.Case{ID: builtID, Args: []json.RawMessage{[]byte(four), []byte(widget)}}
			_, err := c.Decoded()
			expectFault(t, err, inVector(fault.Field(argsAt), fault.Index(1), fault.Field(typeAt)), unknownWidget)
		})
	})

	t.Run("SkipReason", func(t *testing.T) {
		t.Parallel()

		t.Run("reports false for a case without a skip table", func(t *testing.T) {
			t.Parallel()
			if _, skipped := (conformance.Case{ID: builtID}).SkipReason(); skipped {
				t.Fatal("a case naming no skip applies to this language")
			}
		})

		t.Run("reports false for a skip of another language", func(t *testing.T) {
			t.Parallel()
			c := conformance.Case{ID: builtID, Skip: map[string]string{"php": "no generics"}}
			if _, skipped := c.SkipReason(); skipped {
				t.Fatal("a skip naming another language does not apply here")
			}
		})

		t.Run("returns the reason of a skip of Go", func(t *testing.T) {
			t.Parallel()
			c := conformance.Case{ID: builtID, Skip: map[string]string{"go": "no such type"}}
			if why, skipped := c.SkipReason(); !skipped || why != "no such type" {
				t.Fatalf("SkipReason returns %q and %t, want the reason and true", why, skipped)
			}
		})
	})
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
// with the options of its relaxations, and compares each outcome with the
// one the case states. A kind that this language cannot build, or an
// assertion that no driver calls, fails the case, because only a skip in the
// definition excuses a case.
func runSubjectCase(t *testing.T, tc conformance.Case, opts []assert.Option) {
	t.Helper()

	for _, surface := range []string{"check", "expect"} {
		r := assert.NewRecorder().WithClock(assert.NewControlled(time.Time{}))
		if !conformance.RunSubject(surface, tc.Assertion, tc.Subject.Kind, r, tc.ID, opts...) {
			t.Fatalf("%s: no subject named %q, or no driver of %s, and the case declares no skip",
				surface, tc.Subject.Kind, tc.Assertion)
		}
		if err := tc.Check(r, surface == "check"); err != nil {
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

// checkWhere fails t unless every record and every call record of r names
// a line of this file. A record names the innermost frame of a test file,
// and this file is the innermost test file under every call. The frames
// between the assertion and this file are of the registry and the subject
// drivers, and neither is a test file.
func checkWhere(t *testing.T, r *assert.Recorder) {
	t.Helper()

	for _, kept := range r.Failures() {
		if filepath.Base(kept.Where.File) != "corpus_test.go" || kept.Where.Line == 0 {
			t.Errorf("%s points at %s:%d, want a line of corpus_test.go",
				kept.Assertion, kept.Where.File, kept.Where.Line)
		}
	}
	for _, line := range r.Records() {
		var call struct {
			Where assert.Where `json:"where"`
		}
		if err := json.Unmarshal([]byte(line), &call); err != nil {
			t.Fatalf("the call record %s is no JSON object: %v", line, err)
		}
		if call.Where.File != "corpus_test.go" || call.Where.Line == 0 {
			t.Errorf("the call record %s points at another line than one of corpus_test.go", line)
		}
	}
}

// called returns a recorder on which call has made its calls.
func called(call func(r *assert.Recorder)) *assert.Recorder {
	r := assert.NewRecorder()
	call(r)
	return r
}

// passedTrue returns the JSON of the call record of a passing call of true
// whose contract is builtID, on the aborting surface or the recording one,
// as a fault of Check states it.
func passedTrue(aborting bool) string {
	return fmt.Sprintf(`{"definition":%q,"seq":1,"assertion":"true","contract":%q,"verdict":"pass","aborting":%t}`,
		conformance.Version(), builtID, aborting)
}

// drawsDigit is a property's body that draws a digit and passes.
func drawsDigit(c *prop.Case) {
	c.Draw(prop.Integer(0, 9), "value")
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
