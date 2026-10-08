// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// The allocation ceilings of the selection generators: the engine's
// construction of the generator, the closures of its decodes and of its
// inverse and its copy of the stated values, and for OneOf the slice of the
// engine's generators.
const (
	// sampledFromAllocs is the ceiling of the allocations of SampledFrom.
	sampledFromAllocs = 5
	// oneOfAllocs is the ceiling of the allocations of OneOf.
	oneOfAllocs = 7
	// optionalAllocs is the ceiling of the allocations of Optional.
	optionalAllocs = 4
	// permutationAllocs is the ceiling of the allocations of Permutation.
	permutationAllocs = 5
)

// TestSelection checks the generators that select among stated values or
// generators.
func TestSelection(t *testing.T) {
	t.Parallel()

	t.Run("SampledFrom", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give uint64
			want string
		}{
			{name: "returns the value at the replayed index", give: 2, want: "c"},
			{name: "returns the first value for an index past the last", give: 5, want: "a"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(prop.SampledFrom("a", "b", "c"), tt.give), tt.want, "the value")
			})
		}

		t.Run("keeps a copy of the values", func(t *testing.T) {
			t.Parallel()
			values := []string{"a", "b"}
			g := prop.SampledFrom(values...)
			values[1] = "changed"
			assert.Equal(t, first(g, 1), "b", "the value stated at construction")
		})

		t.Run("panics for no value", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.SampledFrom[int]() }, "no value to sample")
			assert.Equal(t, got, any("prop: sampled-from of no value"), "the panic names the generator")
		})
	})

	t.Run("OneOf", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a value of the generator at the replayed index", func(t *testing.T) {
			t.Parallel()
			got := first(prop.OneOf(prop.Just(-1), prop.Integer(0, 9)), 1, 7)
			assert.Equal(t, got, 7, "the second generator's value")
		})

		t.Run("returns the first generator's simplest value", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(prop.OneOf(prop.Just(-1), prop.Integer(0, 9))), -1, "the first generator")
		})

		t.Run("panics for no generator", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.OneOf[int]() }, "no generator to choose")
			assert.Equal(t, got, any("prop: one-of of no generator"), "the panic names the generator")
		})
	})

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for an absent value", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, first(prop.Optional(prop.Integer(0, 9)), 0), "no value")
		})

		t.Run("returns a pointer to a present value", func(t *testing.T) {
			t.Parallel()
			got := first(prop.Optional(prop.Integer(0, 9)), 1, 7)
			assert.NotNil(t, got, "a value")
			assert.Equal(t, *got, 7, "the replayed value")
		})
	})

	t.Run("Permutation", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []uint64
			want []string
		}{
			{
				name: "returns the values in their stated order as the simplest",
				give: nil,
				want: []string{"a", "b", "c"},
			},
			{name: "swaps each position with the replayed index", give: []uint64{2, 1}, want: []string{"c", "b", "a"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(prop.Permutation("a", "b", "c"), tt.give...), tt.want, "the ordering")
			})
		}

		t.Run("keeps a copy of the values", func(t *testing.T) {
			t.Parallel()
			values := []string{"a", "b"}
			g := prop.Permutation(values...)
			values[0] = "changed"
			assert.Equal(t, first(g), []string{"a", "b"}, "the values stated at construction")
		})
	})
}

// TestSelectionAllocs checks the allocation ceilings of the selection
// generators.
func TestSelectionAllocs(t *testing.T) {
	digit := prop.Integer(0, 9)
	assert.MaxAllocs(t, func() { _ = prop.SampledFrom("a", "b") }, sampledFromAllocs,
		"SampledFrom allocates its copy and its decode")
	assert.MaxAllocs(t, func() { _ = prop.OneOf(digit, digit) }, oneOfAllocs, "OneOf allocates its copy and its decode")
	assert.MaxAllocs(t, func() { _ = prop.Optional(digit) }, optionalAllocs, "Optional allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.Permutation("a", "b") }, permutationAllocs,
		"Permutation allocates its copy and its decode")
}

// BenchmarkSelection measures each selection generator.
func BenchmarkSelection(b *testing.B) {
	digit := prop.Integer(0, 9)

	b.Run("SampledFrom", func(b *testing.B) {
		var got prop.Generator[string]
		c := bench.Start(b).MaxAllocs(sampledFromAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.SampledFrom("a", "b")
		}
		assert.Equal(b, first(got, 1), "b", "the sampled value")
	})

	b.Run("OneOf", func(b *testing.B) {
		var got prop.Generator[int]
		c := bench.Start(b).MaxAllocs(oneOfAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.OneOf(digit, digit)
		}
		assert.Equal(b, first(got, 1, 7), 7, "the chosen value")
	})

	b.Run("Optional", func(b *testing.B) {
		var got prop.Generator[*int]
		c := bench.Start(b).MaxAllocs(optionalAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Optional(digit)
		}
		assert.Nil(b, first(got), "the absent value")
	})

	b.Run("Permutation", func(b *testing.B) {
		var got prop.Generator[[]string]
		c := bench.Start(b).MaxAllocs(permutationAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Permutation("a", "b")
		}
		assert.Equal(b, first(got, 1), []string{"b", "a"}, "the swapped ordering")
	})
}
