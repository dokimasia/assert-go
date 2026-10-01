// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocations of each selection generator's constructor, measured.
const (
	// sampledFromAllocs are the allocations of SampledFrom and OneOf: the
	// copy of the values, the decode and the decode with its type erased.
	sampledFromAllocs = 3
	// optionalAllocs are the allocations of Optional: the decode and the
	// decode with its type erased.
	optionalAllocs = 2
	// permutationAllocs are the allocations of Permutation: the copy of
	// the values, the decode and the decode with its type erased.
	permutationAllocs = 3
)

// TestSelection checks the generators that select among stated values or
// generators: the values they decode and the choices they record.
func TestSelection(t *testing.T) {
	t.Parallel()

	t.Run("SampledFrom", func(t *testing.T) {
		t.Parallel()

		letters := engine.SampledFrom("a", "b", "c")
		tests := []struct {
			name     string
			give     []choice.Choice
			want     string
			recorded []choice.Choice
		}{
			{name: "returns the value at the chosen index", give: integers(2), want: "c", recorded: integers(2)},
			{
				name:     "returns the first value for an index past the values",
				give:     integers(3),
				want:     "a",
				recorded: integers(0),
			},
			{name: "returns the first value past the last choice", want: "a", recorded: integers(0)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, letters, tt.give...)
				assert.Equal(t, got, tt.want, "the selected value")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded index")
				assert.Equal(t, labels(e.Case.Spans()), []string{"sampled-from"}, "one span")
			})
		}

		t.Run("returns the pinned values of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(letters, 42, 6)
			assert.Equal(t, values, []string{"a", "b", "c", "a", "c", "a"}, "the values of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(0), integers(1), integers(2), integers(0), integers(2), integers(0),
			}), "the recorded indices")
		})

		t.Run("returns the stated values after the caller changes its slice", func(t *testing.T) {
			t.Parallel()
			values := []string{"a", "b"}
			g := engine.SampledFrom(values...)
			values[1] = "changed"
			got, _ := decode(t, g, integers(1)...)
			assert.Equal(t, got, "b", "the value stated at construction")
		})

		t.Run("panics for no value", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { engine.SampledFrom[int]() }, "no domain")
		})
	})

	t.Run("OneOf", func(t *testing.T) {
		t.Parallel()

		mixed := engine.OneOf(anyOf(engine.Integer(0, 9)), anyOf(engine.StringOver("ab", sizes(t, 0, 2))))
		tests := []struct {
			name     string
			give     []choice.Choice
			want     any
			recorded []choice.Choice
		}{
			{
				name:     "returns a value of the chosen generator",
				give:     []choice.Choice{unsigned(1), sequence(1, 0)},
				want:     "ba",
				recorded: []choice.Choice{unsigned(1), sequence(1, 0)},
			},
			{
				name:     "returns the target of the chosen generator for a choice of another kind",
				give:     integers(1, 5),
				want:     "",
				recorded: []choice.Choice{unsigned(1), sequence()},
			},
			{
				name:     "returns the first generator's target past the last choice",
				want:     0,
				recorded: integers(0, 0),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, mixed, tt.give...)
				assert.Equal(t, got, tt.want, "the chosen generator's value")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the index, then the value's choices")
			})
		}

		t.Run("returns the pinned values of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.OneOf(anyOf(engine.Integer(0, 9)), anyOf(engine.Boolean(1, 2))), 42, 6)
			assert.Equal(t, values, []any{5, 1, false, 7, false, 6}, "the values of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(0, 5), integers(0, 1), integers(1, 0), integers(0, 7), integers(1, 0), integers(0, 6),
			}), "the recorded index and value of each case")
		})

		t.Run("panics for no generator", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { engine.OneOf[int]() }, "no domain")
		})
	})

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		maybe := engine.Optional(engine.Integer(1, 100))
		tests := []struct {
			name     string
			give     []choice.Choice
			want     *int
			recorded []choice.Choice
		}{
			{
				name:     "returns a present value for a presence of 1",
				give:     integers(1, 42),
				want:     new(42),
				recorded: integers(1, 42),
			},
			{name: "returns nil for a presence of 0", give: integers(0, 42), recorded: integers(0)},
			{name: "returns nil past the last choice", recorded: integers(0)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, maybe, tt.give...)
				assert.Equal(t, got, tt.want, "the optional value")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the presence, then the value's choices")
			})
		}

		t.Run("returns the pinned values of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.Optional(engine.Integer(0, 100)), 42, 6)
			assert.Equal(t, values, []*int{nil, nil, new(76), nil, new(97), nil}, "the values of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(0), integers(0), integers(1, 76), integers(0), integers(1, 97), integers(0),
			}), "the recorded presence and value of each case")
		})
	})

	t.Run("Permutation", func(t *testing.T) {
		t.Parallel()

		letters := engine.Permutation("a", "b", "c", "d")
		tests := []struct {
			name     string
			give     []choice.Choice
			want     []string
			recorded []choice.Choice
		}{
			{
				name:     "returns the values swapped once per position",
				give:     integers(2, 3, 3),
				want:     []string{"c", "d", "b", "a"},
				recorded: integers(2, 3, 3),
			},
			{
				name:     "returns the position for a swap outside its bounds",
				give:     integers(1, 0, 0),
				want:     []string{"b", "a", "c", "d"},
				recorded: integers(1, 1, 2),
			},
			{
				name:     "returns the stated order past the last choice",
				want:     []string{"a", "b", "c", "d"},
				recorded: integers(0, 1, 2),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, letters, tt.give...)
				assert.Equal(t, got, tt.want, "the ordering")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "one swap per position")
			})
		}

		t.Run("returns the pinned values of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.Permutation(1, 2, 3, 4, 5), 42, 6)
			assert.Equal(t, values, [][]int{
				{2, 3, 4, 5, 1}, {3, 1, 4, 2, 5}, {5, 2, 1, 3, 4}, {1, 5, 4, 3, 2}, {5, 1, 3, 2, 4}, {2, 3, 4, 1, 5},
			}, "the orderings of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(1, 2, 3, 4), integers(2, 2, 3, 3), integers(4, 1, 4, 4),
				integers(0, 4, 3, 3), integers(4, 4, 2, 4), integers(1, 2, 3, 3),
			}), "the recorded swaps of each case")
		})

		t.Run("returns one value without a choice", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, engine.Permutation("x"), integers(5)...)
			assert.Equal(t, got, []string{"x"}, "the one ordering")
			assert.Empty(t, e.Case.Choices(), "no choice")
		})

		t.Run("returns a new slice for each case", func(t *testing.T) {
			t.Parallel()
			first, _ := decode(t, letters)
			first[0] = "changed"
			second, _ := decode(t, letters)
			assert.Equal(t, second, []string{"a", "b", "c", "d"}, "the stated order")
		})
	})
}

// TestSelectionZeroAlloc checks the allocation ceilings of the selection
// generators' constructors.
func TestSelectionZeroAlloc(t *testing.T) {
	digit := engine.Integer(0, 9)
	assert.MaxAllocs(t, func() { _ = engine.SampledFrom(1, 2, 3) }, sampledFromAllocs,
		"SampledFrom allocates its values and its decodes")
	assert.MaxAllocs(t, func() { _ = engine.OneOf(digit, digit) }, sampledFromAllocs,
		"OneOf allocates its generators and its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Optional(digit) }, optionalAllocs, "Optional allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Permutation(1, 2, 3) }, permutationAllocs,
		"Permutation allocates its values and its decodes")
}

// BenchmarkSelection measures the constructors of the selection
// generators.
func BenchmarkSelection(b *testing.B) {
	digit := engine.Integer(0, 9)

	b.Run("SampledFrom", func(b *testing.B) {
		var got engine.Generator[int]
		c := bench.Start(b).MaxAllocs(sampledFromAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.SampledFrom(1, 2, 3)
		}
		assert.Equal(b, got.ID(), "sampled-from", "the id")
	})

	b.Run("OneOf", func(b *testing.B) {
		var got engine.Generator[int]
		c := bench.Start(b).MaxAllocs(sampledFromAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.OneOf(digit, digit)
		}
		assert.Equal(b, got.ID(), "one-of", "the id")
	})

	b.Run("Optional", func(b *testing.B) {
		var got engine.Generator[*int]
		c := bench.Start(b).MaxAllocs(optionalAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Optional(digit)
		}
		assert.Equal(b, got.ID(), "optional", "the id")
	})

	b.Run("Permutation", func(b *testing.B) {
		var got engine.Generator[[]int]
		c := bench.Start(b).MaxAllocs(permutationAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Permutation(1, 2, 3)
		}
		assert.Equal(b, got.ID(), "permutation", "the id")
	})
}

// anyOf returns g with its values as any, so generators of different types
// share one one-of.
func anyOf[T any](g engine.Generator[T]) engine.Generator[any] {
	return g.Map(func(v T) any { return v })
}
