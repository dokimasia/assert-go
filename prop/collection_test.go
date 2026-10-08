// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// The allocations of the collection generators, measured. Each is the
// engine's construction of the generator: the closures of its decodes and
// of its inverse.
const (
	// listAllocs are the allocations of List.
	listAllocs = 4
	// dictAllocs are the allocations of Dict.
	dictAllocs = 6
)

// TestCollection checks the list and dict generators and the option that
// makes a list unique.
func TestCollection(t *testing.T) {
	t.Parallel()

	t.Run("Unique", func(t *testing.T) {
		t.Parallel()

		t.Run("makes List discard an element equal to an earlier one", func(t *testing.T) {
			t.Parallel()
			got := first(prop.List(prop.Integer(0, 9), prop.Unique()), 1, 5, 1, 5, 1, 6, 0)
			assert.Equal(t, got, []int{5, 6}, "the second 5 is discarded")
		})

		t.Run("makes List reject a case still below its shortest length after ten discards", func(t *testing.T) {
			t.Parallel()
			_, outcome := decoded(prop.List(prop.Integer(0, 0), prop.Unique(), prop.MinSize(2)))
			assert.Equal(t, outcome, prop.Rejected, "one zero, then ten discarded zeros")
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the elements between the choices to continue", func(t *testing.T) {
			t.Parallel()
			got := first(prop.List(prop.Integer(0, 9), prop.MaxSize(3)), 1, 7, 1, 3, 0)
			assert.Equal(t, got, []int{7, 3}, "the value of the definition's vector")
		})

		t.Run("keeps a repeated element without Unique", func(t *testing.T) {
			t.Parallel()
			got := first(prop.List(prop.Integer(0, 9)), 1, 5, 1, 5, 0)
			assert.Equal(t, got, []int{5, 5}, "both elements")
		})
	})

	t.Run("Dict", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the entries between the choices to continue", func(t *testing.T) {
			t.Parallel()
			got := first(prop.Dict(prop.Integer(0, 9), prop.Integer(0, 9)), 1, 3, 4, 1, 5, 6, 0)
			assert.Equal(t, got, map[int]int{3: 4, 5: 6}, "two entries")
		})

		t.Run("discards an entry whose key repeats an earlier key", func(t *testing.T) {
			t.Parallel()
			got := first(prop.Dict(prop.Integer(0, 9), prop.Integer(0, 9)), 1, 3, 4, 1, 3, 9, 1, 5, 6, 0)
			assert.Equal(t, got, map[int]int{3: 4, 5: 6}, "the first entry of key 3")
		})

		t.Run("returns the fewest entries that its lengths admit", func(t *testing.T) {
			t.Parallel()
			got := first(prop.Dict(prop.Integer(0, 9), prop.Integer(0, 9), prop.MinSize(1)))
			assert.Equal(t, got, map[int]int{0: 0}, "one entry of simplest values")
		})

		t.Run("stores the two zeros of float keys under one key", func(t *testing.T) {
			t.Parallel()
			g := prop.Dict(prop.Float(-1.0, 1.0), prop.Integer(0, 9))
			got := replayedOf(g, integer(1), floating(0), integer(7), integer(1), floating(math.Copysign(0, -1)),
				integer(8), integer(0))
			assert.Equal(t, got, map[float64]int{0: 8}, "the draw keeps both zeros, the map one")
		})
	})
}

// TestCollectionAllocs checks the allocation ceilings of the collection
// generators and options.
func TestCollectionAllocs(t *testing.T) {
	digit := prop.Integer(0, 9)
	assert.MaxAllocs(t, func() { _ = prop.Unique() }, 0, "Unique allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.List(digit) }, listAllocs, "List allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.Dict(digit, digit) }, dictAllocs, "Dict allocates its decode")
}

// BenchmarkCollection measures each collection generator and option.
func BenchmarkCollection(b *testing.B) {
	digit := prop.Integer(0, 9)

	b.Run("Unique", func(b *testing.B) {
		var got prop.ListOption
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Unique()
		}
		assert.Equal(b, first(prop.List(digit, got), 1, 5, 1, 5, 0), []int{5}, "the unique list")
	})

	b.Run("List", func(b *testing.B) {
		var got prop.Generator[[]int]
		c := bench.Start(b).MaxAllocs(listAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.List(digit)
		}
		assert.Equal(b, first(got, 1, 7, 0), []int{7}, "the list")
	})

	b.Run("Dict", func(b *testing.B) {
		var got prop.Generator[map[int]int]
		c := bench.Start(b).MaxAllocs(dictAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Dict(digit, digit)
		}
		assert.Equal(b, first(got, 1, 3, 4, 0), map[int]int{3: 4}, "the dict")
	})
}
