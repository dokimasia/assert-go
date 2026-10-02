// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
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
}
