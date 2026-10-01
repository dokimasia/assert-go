// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocations of each collection generator's constructor, measured.
const (
	// listAllocs are the allocations of List and UniqueList: the decode
	// and the decode with its type erased.
	listAllocs = 2
	// dictAllocs are the allocations of Dict: the decode of an entry, the
	// key of an entry, the decode and the decode with its type erased.
	dictAllocs = 4
)

// rejectedPairs are the choices of a collection with a minimum length of
// 2 whose every element repeats the first: a forced continue flag and the
// element's target, eleven times, before the tenth discard rejects it.
var rejectedPairs = integers(1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0)

// TestCollection checks the list, unique list and dict generators: the
// values they decode, the choices and spans they record, and the
// rejections of a collection that cannot reach its minimum length.
func TestCollection(t *testing.T) {
	t.Parallel()

	digit := engine.Integer(0, 9)

	t.Run("List", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			sizes    choice.Sizes
			give     []choice.Choice
			want     []int
			recorded []choice.Choice
		}{
			{
				name:     "returns an element for each continue flag",
				sizes:    sizes(t, 0, 3),
				give:     integers(1, 7, 1, 3, 0),
				want:     []int{7, 3},
				recorded: integers(1, 7, 1, 3, 0),
			},
			{
				name:     "returns the minimum number of targets past the last choice",
				sizes:    unbounded(t, 2),
				want:     []int{0, 0},
				recorded: integers(1, 0, 1, 0, 0),
			},
			{
				name:     "returns the maximum number of elements for a continue flag past it",
				sizes:    sizes(t, 0, 2),
				give:     integers(1, 1, 1, 2, 1, 3, 0),
				want:     []int{1, 2},
				recorded: integers(1, 1, 1, 2, 0),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, engine.List(digit, tt.sizes), tt.give...)
				assert.Equal(t, got, tt.want, "the list")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "a flag before each element and one after")
			})
		}

		t.Run("returns a span for each element that starts at its continue flag", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, engine.List(digit, sizes(t, 0, 3)), integers(1, 7, 1, 3, 0)...)
			assert.Equal(t, e.Case.Spans(), []engine.Span{
				{Label: "list", Start: 0, End: 5, Depth: 0, Parent: -1},
				{Label: "element", Start: 0, End: 2, Depth: 1, Parent: 0},
				{Label: "integer", Start: 1, End: 2, Depth: 2, Parent: 1},
				{Label: "element", Start: 2, End: 4, Depth: 1, Parent: 0},
				{Label: "integer", Start: 3, End: 4, Depth: 2, Parent: 3},
			}, "the list's span, then each element's span around its integer")
		})

		t.Run("returns the pinned lists of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.List(digit, sizes(t, 0, 10)), 42, 6)
			assert.Equal(t, values, [][]int{nil, nil, {1, 1, 1, 1, 9, 1, 1, 8, 0, 1}, {9, 5}, {1, 1, 5}, {3, 5, 5}},
				"the lists of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(0),
				integers(0),
				integers(1, 1, 1, 1, 1, 1, 1, 1, 1, 9, 1, 1, 1, 1, 1, 8, 1, 0, 1, 1, 0),
				integers(1, 9, 1, 5, 0),
				integers(1, 1, 1, 1, 1, 5, 0),
				integers(1, 3, 1, 5, 1, 5, 0),
			}), "the recorded flags and elements of each case")
		})

		t.Run("returns the pinned lengths of seed 7 for elements without a choice", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.List(engine.Just(5), sizes(t, 0, 4)), 7, 4)
			assert.Equal(t, values, [][]int{{5}, {5, 5, 5, 5}, nil, {5, 5}}, "the lists of the first four cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(1, 0), integers(1, 1, 1, 1, 0), integers(0), integers(1, 1, 0),
			}), "the recorded flags of each case")
		})
	})

	t.Run("UniqueList", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a list without an element equal to an earlier one", func(t *testing.T) {
			t.Parallel()
			choices := integers(1, 4, 1, 4, 1, 5, 0)
			got, e := decode(t, engine.UniqueList(digit, sizes(t, 0, 3)), choices...)
			assert.Equal(t, got, []int{4, 5}, "the second 4 is discarded")
			assert.True(t, sameChoices(e.Case.Choices(), choices), "the discarded element stays recorded")
		})

		t.Run("discards a dict equal to an earlier one in another order", func(t *testing.T) {
			t.Parallel()
			pair := engine.Dict(engine.SampledFrom("a", "b"), digit, sizes(t, 0, 2))
			choices := integers(1, 1, 0, 1, 1, 1, 2, 0, 1, 1, 1, 2, 1, 0, 1, 0, 0)
			got, e := decode(t, engine.UniqueList(pair, sizes(t, 0, 3)), choices...)
			assert.Equal(t, got, []map[string]int{{"a": 1, "b": 2}}, "the second dict, b then a, is discarded")
			assert.True(t, sameChoices(e.Case.Choices(), choices), "the discarded dict stays recorded")
		})

		t.Run("rejects the case below its minimum length after ten discards", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, engine.UniqueList(digit, unbounded(t, 2)))
			assert.Equal(t, e.Status, engine.CaseRejected, "the list cannot reach two elements")
			assert.True(t, sameChoices(e.Case.Choices(), rejectedPairs), "ten discards after the first element")
		})

		t.Run("returns the list after ten discards at its minimum length", func(t *testing.T) {
			t.Parallel()
			flags := integers(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1)
			got, e := decode(t, engine.UniqueList(engine.Just(5), unbounded(t, 1)), flags...)
			assert.Equal(t, got, []int{5}, "one element")
			assert.Equal(t, e.Status, engine.CasePassed, "the list reached its minimum")
			assert.True(t, sameChoices(e.Case.Choices(), flags), "eleven continue flags and no stop flag")
		})

		t.Run("returns the pinned lists of seed 7", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.UniqueList(engine.Integer(0, 3), sizes(t, 1, 4)), 7, 6)
			assert.Equal(t, values, [][]int{{3}, {0, 1, 3}, {2}, {3}, {0, 3, 1}, {2, 0}},
				"the lists of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(1, 3, 0),
				integers(1, 0, 1, 1, 1, 3, 0),
				integers(1, 2, 0),
				integers(1, 3, 0),
				integers(1, 0, 1, 3, 1, 3, 1, 0, 1, 1, 0),
				integers(1, 2, 1, 0, 0),
			}), "the recorded flags and elements of each case, the discarded ones included")
		})
	})

	t.Run("Dict", func(t *testing.T) {
		t.Parallel()

		letters := engine.Dict(engine.SampledFrom("a", "b"), digit, unbounded(t, 0))
		tests := []struct {
			name string
			give []choice.Choice
			want map[string]int
		}{
			{
				name: "returns an entry for each continue flag",
				give: integers(1, 1, 5, 1, 0, 7, 0),
				want: map[string]int{"b": 5, "a": 7},
			},
			{
				name: "returns a map without an entry whose key an earlier entry has",
				give: integers(1, 0, 5, 1, 0, 6, 1, 1, 2, 0),
				want: map[string]int{"a": 5, "b": 2},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, letters, tt.give...)
				assert.Equal(t, got, tt.want, "the map")
				assert.True(t, sameChoices(e.Case.Choices(), tt.give), "the flags, keys and values")
			})
		}

		t.Run("returns a span for each entry that starts at its continue flag", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, letters, integers(1, 1, 5, 0)...)
			assert.Equal(t, labels(e.Case.Spans()), []string{"dict", "entry", "sampled-from", "integer"},
				"the dict's span, then the entry's span around its key and its value")
			assert.Equal(t, e.Case.Spans()[1], engine.Span{Label: "entry", Start: 0, End: 3, Depth: 1, Parent: 0},
				"the entry's span from its flag")
		})

		t.Run("rejects the case below its minimum length after ten discards", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, engine.Dict(engine.Just("k"), digit, unbounded(t, 2)))
			assert.Equal(t, e.Status, engine.CaseRejected, "the dict cannot reach two keys")
			assert.True(t, sameChoices(e.Case.Choices(), rejectedPairs), "ten discards after the first entry")
		})

		t.Run("returns one entry for the two zeros", func(t *testing.T) {
			t.Parallel()
			zeros := engine.Dict(engine.SampledFrom(0.0, math.Copysign(0, -1)), digit, unbounded(t, 0))
			got, e := decode(t, zeros, integers(1, 0, 1, 1, 1, 2, 0)...)
			assert.Equal(t, got, map[float64]int{0: 2}, "Go's equality joins the zeros, and the later value is kept")
			entry := []string{"entry", "sampled-from", "integer"}
			assert.Equal(t, labels(e.Case.Spans())[1:], slices.Concat(entry, entry),
				"the definition keeps both entries")
		})

		t.Run("returns one entry for NaN", func(t *testing.T) {
			t.Parallel()
			keys := engine.SampledFrom(math.NaN(), math.Float64frombits(0x7FF8000000000001))
			nans := engine.Dict(keys, digit, unbounded(t, 0))
			got, _ := decode(t, nans, integers(1, 0, 1, 1, 1, 2, 0)...)
			assert.Length(t, got, 1, "every NaN is one key")
		})

		t.Run("returns the pinned maps of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.Dict(engine.Integer(0, 5), engine.Boolean(1, 2), sizes(t, 0, 4)), 42, 6)
			assert.Equal(t, values, []map[int]bool{
				{},
				{},
				{1: false, 5: false, 4: true},
				{5: true},
				{1: false, 2: false, 5: false},
				{1: true, 2: true, 0: true, 5: false},
			}, "the maps of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(0),
				integers(0),
				integers(1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 5, 0, 1, 1, 1, 1, 4, 1, 0),
				integers(1, 5, 1, 1, 5, 1, 0),
				integers(1, 1, 0, 1, 1, 1, 1, 2, 0, 1, 5, 0, 1, 5, 1, 0),
				integers(1, 1, 1, 1, 2, 1, 1, 0, 1, 1, 5, 0, 0),
			}), "the recorded flags, keys and values of each case")
		})
	})
}

// TestCollectionZeroAlloc checks the allocation ceilings of the collection
// generators' constructors.
func TestCollectionZeroAlloc(t *testing.T) {
	digit, upTo := engine.Integer(0, 9), sizes(t, 0, 3)
	assert.MaxAllocs(t, func() { _ = engine.List(digit, upTo) }, listAllocs, "List allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.UniqueList(digit, upTo) }, listAllocs, "UniqueList allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Dict(digit, digit, upTo) }, dictAllocs,
		"Dict allocates its entry's decode and key and its decodes")
}

// BenchmarkCollection measures the constructors of the collection
// generators.
func BenchmarkCollection(b *testing.B) {
	digit, upTo := engine.Integer(0, 9), sizes(b, 0, 3)

	b.Run("List", func(b *testing.B) {
		var got engine.Generator[[]int]
		c := bench.Start(b).MaxAllocs(listAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.List(digit, upTo)
		}
		assert.Equal(b, got.ID(), "list", "the id")
	})

	b.Run("UniqueList", func(b *testing.B) {
		var got engine.Generator[[]int]
		c := bench.Start(b).MaxAllocs(listAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.UniqueList(digit, upTo)
		}
		assert.Equal(b, got.ID(), "list", "the id")
	})

	b.Run("Dict", func(b *testing.B) {
		var got engine.Generator[map[int]int]
		c := bench.Start(b).MaxAllocs(dictAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Dict(digit, digit, upTo)
		}
		assert.Equal(b, got.ID(), "dict", "the id")
	})
}
