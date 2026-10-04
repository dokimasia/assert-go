// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// TestOther checks the further failures of a run, pinned to the
// definition's behaviour vectors.
func TestOther(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("reports a second failure with its own counterexample and token", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				v := c.Draw(prop.Integer(0, 1000), drawn)
				if v%2 == 1 {
					fail(c, "odd")
				}
				if v >= 51 {
					fail(c, big)
				}
			}
			got := detailOf(body, prop.Seed(7))
			want := []prop.Other{{
				Counterexample: []prop.Drawn{{Label: drawn, Value: 52}},
				Failure:        assert.Failure{Assertion: big},
				Choices:        "prop1:ADQ",
			}}
			assert.Equal(t, got[failureField], any(assert.Failure{Assertion: "odd"}), "the first failure found")
			assert.Equal(t, got[othersField], any(want), "the smallest even value of 51 or more")
		})

		t.Run("reports no other failure as an empty list", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7))
			assert.Equal(t, got[othersField], any([]prop.Other{}), "an empty list, not nil")
		})
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the failure record, each draw's label and typed literal, and the token", func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(other)
			assert.NoError(t, err, "the failure is JSON")
			assert.Equal(t, string(got), `{"failure":{"assertion":"big","contract":"","detail":{}},`+
				`"counterexample":[{"label":"value","value":{"type":"int","value":52}}],"choices":"prop1:ADQ"}`,
				"the failure as the record of a run states it")
		})
	})
}

// other is the further failure of the definition's vector.
var other = prop.Other{
	Counterexample: []prop.Drawn{{Label: drawn, Value: 52}},
	Failure:        assert.Failure{Assertion: big},
	Choices:        "prop1:ADQ",
}

// otherJSONAllocs are the allocations of MarshalJSON on the further failure
// of the definition's vector, measured.
const otherJSONAllocs = 17

// TestOtherAllocs checks the ceiling of MarshalJSON.
func TestOtherAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _, _ = other.MarshalJSON() }, otherJSONAllocs, "MarshalJSON allocates its JSON")
}

// BenchmarkOther measures MarshalJSON.
func BenchmarkOther(b *testing.B) {
	b.Run("MarshalJSON", func(b *testing.B) {
		got, _ := other.MarshalJSON()
		c := bench.Start(b).MaxAllocs(otherJSONAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = other.MarshalJSON()
		}
		assert.Contains(b, string(got), `"choices":"prop1:ADQ"`, "the token")
	})
}
