// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// The allocations of the combinators, measured. Each is the engine's
// construction of the generator: the closure of its decode and of its
// erased decode, a closure that adapts a function, and for Filter and Just
// the closure of the inverse.
const (
	// mapAllocs are the allocations of Map of an integer, with the mapping
	// of its values for the explain phase.
	mapAllocs = 3
	// filterAllocs are the allocations of Filter.
	filterAllocs = 4
	// bindAllocs are the allocations of Bind.
	bindAllocs = 3
	// compositeAllocs are the allocations of Composite.
	compositeAllocs = 3
	// justAllocs are the allocations of Just.
	justAllocs = 3
)

// TestGenerator checks the combinators that build a generator from
// others: the values they decode and how they shrink.
func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Map", func(t *testing.T) {
		t.Parallel()

		t.Run("returns f applied to the value", func(t *testing.T) {
			t.Parallel()
			got, outcome := decoded(prop.Integer(0, 9).Map(double), 7)
			assert.Equal(t, got, 14, "twice the replayed value")
			assert.Equal(t, outcome, prop.Passed, "the case passes")
		})

		t.Run("shrinks a value as the value it maps from", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				if c.Draw(prop.Integer(0, 100).Map(double), drawn) >= 20 {
					fail(c, big)
				}
			}
			got := detailOf(body, prop.Seed(7), prop.Explain(false))
			want := []prop.Drawn{{Label: drawn, Value: 20}}
			assert.Equal(t, got[counterexampleField], any(want), "twice the smallest failing source")
		})

		t.Run("reports f of the nearest passing integer", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				if c.Draw(prop.Integer(0, 10000).Map(double), drawn) >= 2002 {
					fail(c, big)
				}
			}
			got := detailOf(body, prop.Seed(7))
			want := []prop.Drawn{{Label: drawn, Value: 2002, Relevance: prop.ValueMatters, NearestPassing: 2000}}
			assert.Equal(t, got[counterexampleField], any(want), "twice the minimal value and twice the one below")
		})
	})

	t.Run("Filter", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a value that keep reports true for", func(t *testing.T) {
			t.Parallel()
			got, outcome := decoded(prop.Integer(0, 9).Filter(isEven), 4)
			assert.Equal(t, got, 4, "the replayed value")
			assert.Equal(t, outcome, prop.Passed, "the case passes")
		})

		t.Run("rejects a case whose replayed value keep reports false for", func(t *testing.T) {
			t.Parallel()
			_, outcome := decoded(prop.Integer(0, 9).Filter(isEven), 3)
			assert.Equal(t, outcome, prop.Rejected, "every attempt reads the odd value")
		})
	})

	t.Run("Bind", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []uint64
			want int
		}{
			{name: "returns a value of the generator that f returns", give: []uint64{3, 2}, want: 2},
			{name: "returns the target of the second generator for a value outside it", give: []uint64{3, 9}, want: 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				upTo := prop.Integer(1, 3).Bind(func(n int) prop.Generator[int] { return prop.Integer(0, n) })
				got, _ := decoded(upTo, tt.give...)
				assert.Equal(t, got, tt.want, "the value of the bound generator")
			})
		}
	})

	t.Run("Composite", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the value that f returns from its draws", func(t *testing.T) {
			t.Parallel()
			got, _ := decoded(sum(), 3, 4)
			assert.Equal(t, got, 7, "the sum of the two replayed values")
		})

		t.Run("records its draws before its own value", func(t *testing.T) {
			t.Parallel()
			detail := replayed(func(c *prop.Case) {
				c.Draw(sum(), drawn)
				fail(c, every)
			}, 3, 4)
			want := []prop.Drawn{{Label: "first", Value: 3}, {Label: "second", Value: 4}, {Label: drawn, Value: 7}}
			assert.Equal(t, detail[counterexampleField], any(want), "the inner draws, then the sum")
		})
	})

	t.Run("Just", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the value without a choice", func(t *testing.T) {
			t.Parallel()
			got, outcome := decoded(prop.Just(42))
			assert.Equal(t, got, 42, "the value")
			assert.Equal(t, outcome, prop.Passed, "a draw that requests the value")
		})
	})
}

// TestGeneratorAllocs checks the allocation ceilings of the
// combinators.
func TestGeneratorAllocs(t *testing.T) {
	digit := prop.Integer(0, 9)
	assert.MaxAllocs(t, func() { _ = digit.Map(double) }, mapAllocs, "Map allocates its decode")
	assert.MaxAllocs(t, func() { _ = digit.Filter(isEven) }, filterAllocs, "Filter allocates its decode")
	assert.MaxAllocs(t, func() { _ = digit.Bind(upTo) }, bindAllocs, "Bind allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.Composite(twice) }, compositeAllocs, "Composite allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.Just(42) }, justAllocs, "Just allocates its decode")
}

// BenchmarkGenerator measures each combinator.
func BenchmarkGenerator(b *testing.B) {
	digit := prop.Integer(0, 9)

	b.Run("Map", func(b *testing.B) {
		var got prop.Generator[int]
		c := bench.Start(b).MaxAllocs(mapAllocs)
		defer c.End()
		for c.Loop() {
			got = digit.Map(double)
		}
		assert.Equal(b, first(got, 7), 14, "the mapped value")
	})

	b.Run("Filter", func(b *testing.B) {
		var got prop.Generator[int]
		c := bench.Start(b).MaxAllocs(filterAllocs)
		defer c.End()
		for c.Loop() {
			got = digit.Filter(isEven)
		}
		assert.Equal(b, first(got, 4), 4, "the kept value")
	})

	b.Run("Bind", func(b *testing.B) {
		var got prop.Generator[int]
		c := bench.Start(b).MaxAllocs(bindAllocs)
		defer c.End()
		for c.Loop() {
			got = digit.Bind(upTo)
		}
		assert.Equal(b, first(got, 3, 2), 2, "the bound value")
	})

	b.Run("Composite", func(b *testing.B) {
		var got prop.Generator[int]
		c := bench.Start(b).MaxAllocs(compositeAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Composite(twice)
		}
		assert.Equal(b, first(got, 4), 8, "the composed value")
	})

	b.Run("Just", func(b *testing.B) {
		var got prop.Generator[int]
		c := bench.Start(b).MaxAllocs(justAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Just(42)
		}
		assert.Equal(b, first(got), 42, "the value")
	})
}

// double returns twice v.
func double(v int) int {
	return 2 * v
}

// upTo returns the generator of the integers in [0, n].
func upTo(n int) prop.Generator[int] {
	return prop.Integer(0, n)
}

// twice draws a digit and returns twice its value.
func twice(c *prop.Case) int {
	return double(c.Draw(prop.Integer(0, 9), "digit"))
}

// sum returns the generator of the sum of two digits, drawn under the
// labels first and second.
func sum() prop.Generator[int] {
	return prop.Composite(func(c *prop.Case) int {
		return c.Draw(prop.Integer(0, 9), "first") + c.Draw(prop.Integer(0, 9), "second")
	})
}
