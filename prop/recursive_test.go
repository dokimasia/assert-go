// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// recursiveAllocs are the allocations of Recursive, measured: the engine's
// recursion, the closures of its decodes and of the inverses of a position
// and of the whole, the adapter of extend, and the extension that extend
// builds.
const recursiveAllocs = 9

// TestRecursive checks the generator of values whose positions are values
// of the generator itself, and the bound on their base values.
func TestRecursive(t *testing.T) {
	t.Parallel()

	t.Run("MaxLeaves", func(t *testing.T) {
		t.Parallel()

		t.Run("makes every position past the bound take the base", func(t *testing.T) {
			t.Parallel()
			got := first(pairs(prop.MaxLeaves(1)), 1, 1, 0, 1, 1)
			assert.Equal(t, got, "(x x)", "the second position takes the base after one leaf")
		})

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.MaxLeaves(0) }, "a bound of no leaf")
			assert.Equal(t, got, any("prop: MaxLeaves(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("Recursive", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []prop.RecursiveOption
			want string
		}{
			{name: "extends a position whose choice is 1", want: "(x (x x))"},
			{name: "keeps the default bound after a zero option", give: []prop.RecursiveOption{{}}, want: "(x (x x))"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(pairs(tt.give...), 1, 1, 0, 1, 1), tt.want, "the nested pair")
			})
		}

		t.Run("returns the base's simplest value as the simplest", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(pairs()), "x", "the base")
		})
	})
}

// TestRecursiveAllocs checks the allocation ceilings of the recursive
// generator and its option.
func TestRecursiveAllocs(t *testing.T) {
	base := prop.Just("x")
	assert.MaxAllocs(t, func() { _ = prop.MaxLeaves(1) }, 0, "MaxLeaves allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.Recursive(base, nest) }, recursiveAllocs,
		"Recursive allocates its recursion and its decodes")
}

// BenchmarkRecursive measures the recursive generator and its option.
func BenchmarkRecursive(b *testing.B) {
	b.Run("MaxLeaves", func(b *testing.B) {
		var got prop.RecursiveOption
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.MaxLeaves(1)
		}
		assert.Equal(b, got, prop.MaxLeaves(1), "the option")
	})

	b.Run("Recursive", func(b *testing.B) {
		var got prop.Generator[string]
		base := prop.Just("x")
		c := bench.Start(b).MaxAllocs(recursiveAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Recursive(base, nest)
		}
		assert.Equal(b, first(got, 1, 0), "(x)", "the nested value")
	})
}

// pairs returns the generator of "x", or of a pair of its own values in
// parentheses, under opts.
func pairs(opts ...prop.RecursiveOption) prop.Generator[string] {
	pair := func(self prop.Generator[string]) prop.Generator[string] {
		return prop.List(self, prop.MinSize(2), prop.MaxSize(2)).Map(func(p []string) string {
			return "(" + p[0] + " " + p[1] + ")"
		})
	}
	return prop.Recursive(prop.Just("x"), pair, opts...)
}

// nest returns the generator of a value of self in parentheses.
func nest(self prop.Generator[string]) prop.Generator[string] {
	return self.Map(func(s string) string { return "(" + s + ")" })
}
