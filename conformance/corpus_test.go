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
func TestCorpus(t *testing.T) {
	t.Parallel()

	byAssertion, err := conformance.Cases()
	assert.NoError(t, err, "the corpus can be read")
	assert.NotEmpty(t, byAssertion, "the corpus states something")

	for id, cases := range byAssertion {
		invoke, registered := conformance.Registry[id]
		assert.True(t, registered, "an invoker is registered for "+string(id))

		for _, tc := range cases {
			t.Run(tc.ID, func(t *testing.T) {
				t.Parallel()

				if why, skipped := tc.SkipReason(); skipped {
					t.Skipf("declared skip: %s", why)
				}

				args, err := tc.Decoded()
				assert.NoError(t, err, "the case's arguments decode")

				r := assert.NewRecorder()
				invoke(r, args, tc.ID)

				assert.NoError(t, tc.Check(r), "the outcome is what the case states")
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
			assert.HasError(t, c.Check(assert.NewRecorder()),
				"a case stating neither pass nor fail is refused")
		})

		t.Run("a failure missing a required substring is refused", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("reported without the detail")

			c := conformance.Case{ID: "x", Expect: "fail", MessageContains: []string{"absent"}}
			assert.HasError(t, c.Check(r),
				"a failure that drops a required substring is refused")
		})

		t.Run("a failure carrying every substring is accepted", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("want 1, got 2")

			c := conformance.Case{ID: "x", Expect: "fail", MessageContains: []string{"want", "got"}}
			assert.NoError(t, c.Check(r), "a failure carrying every substring is accepted")
		})
	})

	t.Run("Decoded", func(t *testing.T) {
		t.Parallel()

		t.Run("a literal it cannot read is refused", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "x", Args: []json.RawMessage{[]byte(`{"type":"widget"}`)}}
			_, err := c.Decoded()
			assert.HasError(t, err, "an argument the encoding does not cover is refused")
		})
	})

	t.Run("SkipReason", func(t *testing.T) {
		t.Parallel()

		t.Run("a case with no skip table applies", func(t *testing.T) {
			t.Parallel()

			_, skipped := conformance.Case{ID: "x"}.SkipReason()
			assert.False(t, skipped, "a case naming no skip applies to this language")
		})

		t.Run("a skip for another language does not apply here", func(t *testing.T) {
			t.Parallel()

			c := conformance.Case{ID: "x", Skip: map[string]string{"php": "no generics"}}
			_, skipped := c.SkipReason()
			assert.False(t, skipped, "a skip naming another language does not apply here")
		})
	})
}
