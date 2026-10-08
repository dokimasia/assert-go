// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// sizeAllocs are the allocations of MinSize and MaxSize, measured: the
// closure of the bound, which escapes with the option a caller keeps.
const sizeAllocs = 1

// TestSize checks the bounds on the length of a list, a dict, a string and
// a byte string.
func TestSize(t *testing.T) {
	t.Parallel()

	t.Run("MinSize", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []prop.ListOption
			want []int
		}{
			{name: "makes n the shortest length", give: []prop.ListOption{prop.MinSize(2)}, want: []int{0, 0}},
			{
				name: "makes a later bound override an earlier one",
				give: []prop.ListOption{prop.MinSize(1), prop.MinSize(3)},
				want: []int{0, 0, 0},
			},
			{
				name: "keeps the bound after a zero option",
				give: []prop.ListOption{prop.MinSize(2), prop.SizeOption{}},
				want: []int{0, 0},
			},
			{name: "makes 0 the shortest length", give: []prop.ListOption{prop.MinSize(0)}, want: []int{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(prop.List(prop.Integer(0, 9), tt.give...)), tt.want, "the simplest list")
			})
		}

		t.Run("bounds the length of a string", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(prop.String(prop.MinSize(2))), "00", "the simplest string")
		})

		t.Run("panics for n below 0", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.MinSize(-1) }, "a negative length")
			assert.Equal(t, got, any("prop: MinSize(-1) is below 0"), "the panic names the option")
		})
	})

	t.Run("MaxSize", func(t *testing.T) {
		t.Parallel()

		t.Run("makes n the longest length", func(t *testing.T) {
			t.Parallel()
			got := first(prop.List(prop.Integer(0, 9), prop.MaxSize(1)), 1, 5, 1, 6)
			assert.Equal(t, got, []int{5}, "the list stops at one element")
		})

		t.Run("makes 0 the longest length", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(prop.Bytes(prop.MaxSize(0))), []byte{}, "the empty byte string")
		})

		t.Run("panics for n below 0", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.MaxSize(-1) }, "a negative length")
			assert.Equal(t, got, any("prop: MaxSize(-1) is below 0"), "the panic names the option")
		})

		t.Run("makes a generator panic below the shortest length", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.Bytes(prop.MinSize(3), prop.MaxSize(2)) }, "no length")
			assert.Equal(t, got, any("prop: MinSize(3) and MaxSize(2) state no length"), "the panic names both")
		})
	})
}

// TestSizeAllocs checks the allocation ceilings of the size options.
func TestSizeAllocs(t *testing.T) {
	var kept prop.SizeOption
	assert.MaxAllocs(t, func() { kept = prop.MinSize(2) }, sizeAllocs, "MinSize allocates its bound")
	assert.MaxAllocs(t, func() { kept = prop.MaxSize(2) }, sizeAllocs, "MaxSize allocates its bound")
	assert.NotEqual(t, kept, prop.SizeOption{}, "the kept option states a bound")
}

// BenchmarkSize measures each size option.
func BenchmarkSize(b *testing.B) {
	b.Run("MinSize", func(b *testing.B) {
		var got prop.SizeOption
		c := bench.Start(b).MaxAllocs(sizeAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.MinSize(2)
		}
		assert.Equal(b, first(prop.String(got)), "00", "the shortest string")
	})

	b.Run("MaxSize", func(b *testing.B) {
		var got prop.SizeOption
		c := bench.Start(b).MaxAllocs(sizeAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.MaxSize(1)
		}
		assert.Equal(b, first(prop.List(prop.Integer(0, 9), got), 1, 5, 1, 6), []int{5}, "the longest list")
	})
}
