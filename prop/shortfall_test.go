// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// nine is the label of the coverage requirement that counts the value 9.
const nine = "nine"

// TestShortfall checks which values are verdicts, and the coverage
// requirements that a run misses, pinned to the definition's behaviour
// vectors.
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

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the verdict", func(t *testing.T) {
			t.Parallel()
			got, err := prop.Refuted.MarshalText()
			assert.NoError(t, err, "every verdict has a spelling")
			assert.Equal(t, string(got), "refuted", "the spelling of the definition")
		})
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

// TestShortfallAllocs checks that Valid allocates nothing, and that
// MarshalText allocates its text.
func TestShortfallAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.Unmet.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = prop.Unmet.MarshalText() }, 1, "MarshalText allocates its text")
}

// BenchmarkShortfall measures Valid under a ceiling of no allocation, and
// MarshalText.
func BenchmarkShortfall(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got, _ = prop.Unmet.MarshalText()
		}
		assert.Equal(b, string(got), "unmet", "the spelling")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Unmet.Valid()
		}
		assert.True(b, got, "Unmet is a verdict")
	})
}
