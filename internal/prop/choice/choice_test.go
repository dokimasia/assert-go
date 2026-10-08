// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestChoice checks when two choices are one.
func TestChoice(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			c, d choice.Choice
			want bool
		}{
			{
				name: "reports false for choices of different kinds",
				c:    choice.Choice{Kind: choice.Integer},
				d:    choice.Choice{Kind: choice.Float},
				want: false,
			},
			{
				name: "reports true for equal integers",
				c:    choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(-5)},
				d:    choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(-5)},
				want: true,
			},
			{
				name: "reports false for -5 against 2^64 - 5",
				c:    choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(-5)},
				d:    choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(math.MaxUint64 - 4)},
				want: false,
			},
			{
				name: "reports false for +0 against -0",
				c:    choice.Choice{Kind: choice.Float},
				d:    choice.Choice{Kind: choice.Float, Float: negativeZero},
				want: false,
			},
			{
				name: "reports true for two NaN payloads",
				c:    choice.Choice{Kind: choice.Float, Float: math.NaN()},
				d:    choice.Choice{Kind: choice.Float, Float: choice.NaN()},
				want: true,
			},
			{name: "reports true for equal sequences", c: sequenceChoice(1, 2), d: sequenceChoice(1, 2), want: true},
			{
				name: "reports false for sequences of different lengths",
				c:    sequenceChoice(1, 2),
				d:    sequenceChoice(1),
				want: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.c.Equal(tt.d), tt.want, "whether the two choices are one")
			})
		}
	})
}

// TestChoiceAllocs checks that Equal allocates nothing.
func TestChoiceAllocs(t *testing.T) {
	c, d := sequenceChoice(1, 2, 3), sequenceChoice(1, 2, 3)
	assert.MaxAllocs(t, func() { _ = c.Equal(d) }, 0, "Equal allocates nothing")
}

// BenchmarkChoice measures Equal under a ceiling of no allocation.
func BenchmarkChoice(b *testing.B) {
	b.Run("Equal", func(b *testing.B) {
		var got bool
		c, d := sequenceChoice(1, 2, 3), sequenceChoice(1, 2, 3)
		bc := bench.Start(b).MaxAllocs(0)
		defer bc.End()
		for bc.Loop() {
			got = c.Equal(d)
		}
		assert.True(b, got, "the sequences are one")
	})
}
