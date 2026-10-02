// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// invalidVerdict is the first value past the two verdicts.
const invalidVerdict prop.Verdict = 2

// The labels of the coverage requirements of the definition's behaviour
// vectors.
const (
	// even counts the even values.
	even = "even"
	// nine counts the value 9.
	nine = "nine"
	// small counts the values below 5.
	small = "small"
)

// TestShortfall checks the coverage requirements that a run misses,
// pinned to the definition's behaviour vectors, and pins each verdict's
// spelling.
func TestShortfall(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Verdict
			want bool
		}{
			{name: "reports true for Refuted", give: prop.Refuted, want: true},
			{name: "reports true for Unmet", give: prop.Unmet, want: true},
			{name: "reports false past Unmet", give: invalidVerdict, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a verdict")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Verdict
			want string
		}{
			{name: "returns refuted for Refuted", give: prop.Refuted, want: "refuted"},
			{name: "returns unmet for Unmet", give: prop.Unmet, want: "unmet"},
			{name: "returns Verdict(2) for a value that is no verdict", give: invalidVerdict, want: "Verdict(2)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the verdict's spelling")
			})
		}
	})

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("reports a refuted requirement with every detail field", func(t *testing.T) {
			t.Parallel()
			want := map[string]any{
				outcomeField:        prop.CoverageUnmet,
				casesField:          100,
				rejectedField:       0,
				seedField:           "7",
				counterexampleField: nil,
				failureField:        nil,
				choicesField:        nil,
				othersField:         nil,
				divergenceField:     nil,
				coverageField: &prop.Shortfall{
					Label:   even,
					Share:   0.9,
					Counted: 45,
					Valid:   100,
					Verdict: prop.Refuted,
				},
			}
			got := detailOf(classifies(1000, even, isEven), prop.Seed(7), prop.Require(even, 0.9))
			assert.Equal(t, got, want, "the record of the definition's vector")
		})

		t.Run("reports an unmet requirement of a domain tested in full", func(t *testing.T) {
			t.Parallel()
			got := detailOf(classifies(9, nine, func(v int) bool { return v == 9 }), prop.Seed(7),
				prop.Require(nine, 0.2))
			want := &prop.Shortfall{Label: nine, Share: 0.2, Counted: 1, Valid: 10, Verdict: prop.Unmet}
			assert.Equal(t, got[coverageField], any(want), "the exact share of the ten digits")
			assert.Equal(t, got[outcomeField], any(prop.CoverageUnmet), "the outcome")
		})

		t.Run("reports no record for a met requirement", func(t *testing.T) {
			t.Parallel()
			got := detailOf(classifies(1000, even, isEven), prop.Seed(7), prop.Require(even, 0.3))
			assert.Nil(t, got, "no record")
		})

		t.Run("reports no record for a requirement met by an exact share", func(t *testing.T) {
			t.Parallel()
			got := detailOf(classifies(9, small, func(v int) bool { return v < 5 }), prop.Seed(7),
				prop.Require(small, 0.5))
			assert.Nil(t, got, "no record")
		})
	})
}

// TestShortfallZeroAlloc checks that no method of Verdict allocates.
func TestShortfallZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.Unmet.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.Unmet.String() }, 0, "String allocates nothing")
}

// BenchmarkShortfall measures each method of Verdict under a ceiling of no
// allocation.
func BenchmarkShortfall(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Unmet.Valid()
		}
		assert.True(b, got, "Unmet is a verdict")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Unmet.String()
		}
		assert.Equal(b, got, "unmet", "the verdict's spelling")
	})
}

// classifies returns the body that draws an integer in [0, most], and
// classifies the case under label when holds reports true for the value.
func classifies(most int, label string, holds func(int) bool) func(*prop.Case) {
	g := prop.Integer(0, most)
	return func(c *prop.Case) {
		if holds(c.Draw(g, drawn)) {
			c.Classify(label)
		}
	}
}

// isEven reports whether v is even.
func isEven(v int) bool {
	return v%2 == 0
}
