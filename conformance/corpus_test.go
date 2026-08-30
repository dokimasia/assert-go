// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"testing"

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
		invoke, registered := conformance.Registry[id]
		if !registered {
			t.Fatalf("an invoker is registered for %s", id)
		}

		for _, tc := range cases {
			t.Run(tc.ID, func(t *testing.T) {
				t.Parallel()

				if why, skipped := tc.SkipReason(); skipped {
					t.Skipf("declared skip: %s", why)
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
			})
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

		t.Run("a failure missing a required substring is refused", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("reported without the detail")

			c := conformance.Case{ID: "x", Expect: "fail", MessageContains: []string{"absent"}}
			if c.Check(r) == nil {
				t.Fatal("a failure that drops a required substring is refused")
			}
		})

		t.Run("a failure carrying every substring is accepted", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("want 1, got 2")

			c := conformance.Case{ID: "x", Expect: "fail", MessageContains: []string{"want", "got"}}
			if err := c.Check(r); err != nil {
				t.Fatalf("a failure carrying every substring is accepted: %v", err)
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
