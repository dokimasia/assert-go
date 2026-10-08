// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// recursiveAllocs is the ceiling of the allocations of Recursive with an
// extension that allocates nothing: the recursion, the generator of one
// position with its decodes and its inverse, and the decodes and the
// inverse of the generator it returns.
const recursiveAllocs = 9

// TestRecursive checks the recursive generator: the values it decodes, the
// choices it records, and the bound on the base values of one value.
func TestRecursive(t *testing.T) {
	t.Parallel()

	t.Run("Recursive", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			width     int
			maxLeaves int
			give      []choice.Choice
			want      any
			recorded  []choice.Choice
		}{
			{
				name:      "returns a nested value",
				width:     2,
				maxLeaves: 3,
				give:      integers(1, 1, 1, 1, 0, 2, 0, 1, 0, 3),
				want:      []any{[]any{2}, 3},
				recorded:  integers(1, 1, 1, 1, 0, 2, 0, 1, 0, 3, 0),
			},
			{
				name:      "returns the base at every position after the last leaf",
				width:     3,
				maxLeaves: 1,
				give:      integers(1, 1, 0, 4, 1, 1, 5, 0),
				want:      []any{4, 5},
				recorded:  integers(1, 1, 0, 4, 1, 0, 5, 0),
			},
			{
				name:      "returns the base's target past the last choice",
				width:     2,
				maxLeaves: engine.DefaultMaxLeaves,
				want:      0,
				recorded:  integers(0, 0),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, tree(t, tt.width, tt.maxLeaves), tt.give...)
				assert.Equal(t, got, tt.want, "the recursive value")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded),
					"a choice of base or extension at each position")
			})
		}

		t.Run("returns a span for each position", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, tree(t, 2, 3), integers(1, 1, 0, 2, 0)...)
			want := []string{"recursive", "list", "element", "recursive", "integer"}
			assert.Equal(t, labels(e.Case.Spans()), want,
				"the outer position, the list, its element and the inner position with its leaf")
		})

		t.Run("returns values whose leaves are counted apart", func(t *testing.T) {
			t.Parallel()
			g, choices := tree(t, 3, 1), integers(0, 4, 1, 1, 0, 5, 0)
			var first, second any
			e := engine.Replay(func(c *engine.Case) {
				first, second = engine.Draw(c, g, "first"), engine.Draw(c, g, "second")
			}, choices, nil)
			assert.Equal(t, [2]any{first, second}, [2]any{4, []any{5}},
				"the second value extends after the first took a leaf")
			assert.True(t, sameChoices(e.Case.Choices(), choices), "the choices as stated")
		})

		t.Run("returns a value nested in itself whose leaves leave the outer count alone", func(t *testing.T) {
			t.Parallel()
			var outer engine.Generator[any]
			inner := engine.Composite(func(c *engine.Case) any { return engine.Draw(c, outer, "inner") })
			upTo := sizes(t, 0, 2)
			extend := func(self engine.Generator[any]) engine.Generator[any] {
				return anyOf(engine.List(engine.OneOf(self, inner), upTo))
			}
			outer = engine.Recursive(anyOf(engine.Integer(0, 9)), extend, 1)
			nested := integers(1, 1, 1, 0, 3, 1, 0, 1, 0, 0)
			got, e := decode(t, outer, nested...)
			assert.Equal[any](t, got, []any{3, []any{}}, "the outer position after the inner leaf still extends")
			assert.True(t, sameChoices(e.Case.Choices(), nested), "the choices as stated")
		})

		t.Run("returns the pinned trees of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(tree(t, 3, 10), 42, 6)
			assert.Equal(t, values, []any{5, 1, []any{}, 7, []any{5}, 6}, "the trees of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(0, 5), integers(0, 1), integers(1, 0), integers(0, 7), integers(1, 1, 0, 5, 0), integers(0, 6),
			}), "the recorded choices of each case")
		})

		t.Run("returns the pinned optional and listed positions of seed 7", func(t *testing.T) {
			t.Parallel()
			upTo := sizes(t, 0, 2)
			g := engine.Recursive(anyOf(engine.Boolean(1, 2)), func(self engine.Generator[any]) engine.Generator[any] {
				return engine.OneOf(anyOf(engine.List(self, upTo)), engine.Optional(self).Map(present))
			}, 5)
			values, choices := generated(g, 7, 6)
			assert.Equal(t, values, []any{nil, true, []any{}, nil, true, true}, "the values of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(1, 1, 0), integers(0, 1), integers(1, 0, 0),
				integers(1, 1, 0), integers(0, 1), integers(1, 1, 1, 0, 1),
			}), "the recorded choices of each case")
		})

		t.Run("panics for fewer than one leaf", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { tree(t, 2, 0) }, "no base value")
		})
	})
}

// TestRecursiveAllocs checks the allocation ceiling of Recursive's
// constructor.
func TestRecursiveAllocs(t *testing.T) {
	base := engine.Integer(0, 9)
	same := func(self engine.Generator[int]) engine.Generator[int] { return self }
	assert.MaxAllocs(t, func() { _ = engine.Recursive(base, same, 3) }, recursiveAllocs,
		"Recursive allocates its recursion and its decodes")
}

// BenchmarkRecursive measures the constructor of a recursive generator.
func BenchmarkRecursive(b *testing.B) {
	base := engine.Integer(0, 9)
	same := func(self engine.Generator[int]) engine.Generator[int] { return self }

	b.Run("Recursive", func(b *testing.B) {
		var got engine.Generator[int]
		c := bench.Start(b).MaxAllocs(recursiveAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Recursive(base, same, 3)
		}
		assert.Equal(b, got.ID(), "recursive", "the id")
	})
}

// present returns the value p points at, and nil for a nil p, as the
// definition decodes an optional.
func present(p *any) any {
	if p == nil {
		return nil
	}
	return *p
}
