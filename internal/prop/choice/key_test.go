// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"math"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// floatsSimplestFirst are floats in the order of their keys: integral
// below 2^53, then by fractional bits, then the infinities, then NaN.
var floatsSimplestFirst = []float64{
	0, math.Copysign(0, -1), 1, -1, 2, 1 << 53, 0.5, -0.5, 1.5, 0.25,
	math.Inf(1), math.Inf(-1), choice.NaN(),
}

// TestKey checks the order that keys give choices: by kind, then by each
// kind's own key.
func TestKey(t *testing.T) {
	t.Parallel()

	t.Run("Compare", func(t *testing.T) {
		t.Parallel()

		integer := choice.Key{}
		float := choice.FloatKey(0)
		sequence := choice.SequenceKey(nil)
		tests := []struct {
			name string
			k, o choice.Key
			want int
		}{
			{name: "returns -1 for an integer against a float", k: integer, o: float, want: -1},
			{name: "returns -1 for a float against a sequence", k: float, o: sequence, want: -1},
			{name: "returns 1 for a sequence against an integer", k: sequence, o: integer, want: 1},
			{name: "returns 0 for two keys of one value", k: choice.FloatKey(0.5), o: choice.FloatKey(0.5), want: 0},
			{
				name: "returns -1 for a shorter sequence",
				k:    choice.SequenceKey([]uint32{9}),
				o:    choice.SequenceKey([]uint32{0, 0}),
				want: -1,
			},
			{
				name: "returns 1 for a sequence larger at its first difference",
				k:    choice.SequenceKey([]uint32{1, 0}),
				o:    choice.SequenceKey([]uint32{0, 1}),
				want: 1,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.k.Compare(tt.o), tt.want, "the order of the two keys")
			})
		}
	})

	t.Run("FloatKey", func(t *testing.T) {
		t.Parallel()

		t.Run("orders integral then fractional then infinite then NaN", func(t *testing.T) {
			t.Parallel()
			got := slices.Clone(floatsSimplestFirst)
			slices.Reverse(got)
			slices.SortStableFunc(got, func(a, b float64) int {
				return choice.FloatKey(a).Compare(choice.FloatKey(b))
			})
			for i, want := range floatsSimplestFirst {
				assert.True(t, choice.SameFloat(got[i], want), "the float at this position of the key order")
			}
		})

		tests := []struct {
			name    string
			simpler float64
			other   float64
		}{
			{name: "orders 2^52 + 1 before 2^53 by magnitude", simpler: 1<<52 + 1, other: 1 << 53},
			{name: "orders 2^53 before 2^53 + 2 by magnitude", simpler: 1 << 53, other: 1<<53 + 2},
			{name: "orders 1e300 before 0.5 as a value without fractional bits", simpler: 1e300, other: 0.5},
			{name: "orders 0.5 before -0.5 at equal magnitude", simpler: 0.5, other: -0.5},
			{name: "orders 0.75 before 0.625 by fractional bits", simpler: 0.75, other: 0.625},
			{
				name:    "orders the largest float before the smallest subnormal",
				simpler: math.MaxFloat64,
				other:   math.SmallestNonzeroFloat64,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := choice.FloatKey(tt.simpler).Compare(choice.FloatKey(tt.other))
				assert.Equal(t, got, -1, "the simpler float's key is smaller")
			})
		}

		t.Run("returns one key for every NaN payload", func(t *testing.T) {
			t.Parallel()
			got := choice.FloatKey(math.NaN()).Compare(choice.FloatKey(choice.NaN()))
			assert.Equal(t, got, 0, "two NaNs are one value")
		})
	})

	t.Run("SequenceKey", func(t *testing.T) {
		t.Parallel()

		t.Run("orders shorter first then element by element", func(t *testing.T) {
			t.Parallel()
			got := [][]uint32{{1, 0}, {0, 0}, {1}, {0, 1}}
			slices.SortFunc(got, func(a, b []uint32) int {
				return choice.SequenceKey(a).Compare(choice.SequenceKey(b))
			})
			assert.Equal(t, got, [][]uint32{{1}, {0, 0}, {0, 1}, {1, 0}}, "the sequences in key order")
		})
	})
}

// TestKeyAllocs checks that no function or method of Key allocates.
func TestKeyAllocs(t *testing.T) {
	elements := []uint32{1, 2, 3}
	k, o := choice.FloatKey(0.25), choice.FloatKey(0.5)
	assert.MaxAllocs(t, func() { _ = choice.FloatKey(0.625) }, 0, "FloatKey allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.SequenceKey(elements) }, 0, "SequenceKey allocates nothing")
	assert.MaxAllocs(t, func() { _ = k.Compare(o) }, 0, "Compare allocates nothing")
}

// BenchmarkKey measures each function and method of Key under a ceiling
// of no allocation.
func BenchmarkKey(b *testing.B) {
	b.Run("FloatKey", func(b *testing.B) {
		var got choice.Key
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.FloatKey(0.625)
		}
		assert.Equal(b, got.Compare(choice.FloatKey(0.625)), 0, "the key of 0.625")
	})

	b.Run("SequenceKey", func(b *testing.B) {
		var got choice.Key
		elements := []uint32{1, 2, 3}
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.SequenceKey(elements)
		}
		assert.Equal(b, got.Compare(choice.SequenceKey(elements)), 0, "the key of the sequence")
	})

	b.Run("Compare", func(b *testing.B) {
		var got int
		k, o := choice.SequenceKey([]uint32{1, 2, 3}), choice.SequenceKey([]uint32{1, 2, 4})
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = k.Compare(o)
		}
		assert.Equal(b, got, -1, "the first sequence is simpler")
	})
}
