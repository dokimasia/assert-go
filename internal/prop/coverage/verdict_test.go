// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/coverage"
)

// invalidVerdict is the first value past the four verdicts.
const invalidVerdict coverage.Verdict = 4

// TestVerdict checks which values are verdicts, and pins each verdict's
// spelling in the definition.
func TestVerdict(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give coverage.Verdict
			want bool
		}{
			{name: "reports true for Met", give: coverage.Met, want: true},
			{name: "reports true for Unmet", give: coverage.Unmet, want: true},
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
			give coverage.Verdict
			want string
		}{
			{name: "returns met for Met", give: coverage.Met, want: "met"},
			{name: "returns refuted for Refuted", give: coverage.Refuted, want: "refuted"},
			{name: "returns undecided for Undecided", give: coverage.Undecided, want: "undecided"},
			{name: "returns unmet for Unmet", give: coverage.Unmet, want: "unmet"},
			{name: "returns Verdict(4) for a value that is no verdict", give: invalidVerdict, want: "Verdict(4)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the verdict's spelling")
			})
		}
	})
}

// TestVerdictZeroAlloc checks that no method of Verdict allocates.
func TestVerdictZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = coverage.Unmet.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = coverage.Unmet.String() }, 0, "String allocates nothing")
}

// BenchmarkVerdict measures each method of Verdict under a ceiling of no
// allocation.
func BenchmarkVerdict(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Unmet.Valid()
		}
		assert.True(b, got, "Unmet is a verdict")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Unmet.String()
		}
		assert.Equal(b, got, "unmet", "the verdict's spelling")
	})
}
