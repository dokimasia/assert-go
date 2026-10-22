// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// valuesAllocs is the ceiling of the allocations of Values of one value of
// an integer, which measures 14: the slice, and the case, the goroutine and
// the record of the case that decodes the value.
const valuesAllocs = 18

// sumAbove returns whether the elements of xs sum to more than n, the
// predicate that the filter of the definition's vector states.
func sumAbove(xs []int, n int) bool {
	sum := 0
	for _, x := range xs {
		sum += x
	}
	return sum > n
}

// TestValues checks the values of a generator for a seed against the
// generation vectors of the definition.
func TestValues(t *testing.T) {
	t.Parallel()

	t.Run("Values", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the values of the random cases of the seed, in order", func(t *testing.T) {
			t.Parallel()

			// The vector integer/draws-from-a-signed-range of the definition.
			got := prop.Values(prop.Integer(-100, 100), 42, 6)
			assert.Equal(t, got, []int{11, 0, 76, 0, 96, 11}, "the values of the first six cases of seed 42")
		})

		t.Run("leaves out the cases that the generator rejects", func(t *testing.T) {
			t.Parallel()

			// The vector filter/reuses-no-value-of-a-rejected-attempt of the
			// definition, whose cases 0, 2 and 3 the filter rejects.
			lists := prop.List(prop.Integer(0, 9), prop.MaxSize(4)).Filter(func(xs []int) bool {
				return sumAbove(xs, 10)
			})
			got := prop.Values(lists, 7, 3)
			assert.Equal(t, got, [][]int{{0, 6, 8, 8}, {2, 2, 9}, {7, 4, 0, 5}}, "the values of cases 1, 4 and 5")
		})

		t.Run("returns the same first values for a larger n", func(t *testing.T) {
			t.Parallel()

			first := prop.Values(prop.Integer(-100, 100), 42, 3)
			assert.Equal(t, prop.Values(prop.Integer(-100, 100), 42, 6)[:3], first, "the first three values")
		})

		t.Run("returns no value for n of 0", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, prop.Values(prop.Integer(-100, 100), 42, 0), "the values of no case")
		})

		t.Run("panics for a negative n", func(t *testing.T) {
			t.Parallel()

			got := assert.Panics(t, func() { prop.Values(prop.Integer(-100, 100), 42, -1) }, "a negative n")
			assert.Equal(t, got, any("prop: Values(g, 42, -1) states a count below 0"),
				"the panic names the call")
		})

		t.Run("passes on a panic of the generator", func(t *testing.T) {
			t.Parallel()

			refusing := prop.Integer(0, 9).Map(func(int) int { panic("the conversion refuses the value") })
			got := assert.Panics(t, func() { prop.Values(refusing, 42, 1) }, "a generator that panics")
			assert.Equal(t, got, any("the conversion refuses the value"), "the panic of the generator")
		})

		t.Run("panics for a generator that rejects more than ten cases for each value", func(t *testing.T) {
			t.Parallel()

			never := prop.Integer(0, 9).Filter(func(int) bool { return false })
			got := assert.Panics(t, func() { prop.Values(never, 42, 2) }, "a generator that rejects every case")
			assert.Equal(t, got, any("prop: Values(g, 42, 2) stops after g rejects 21 cases for 2 values"),
				"the panic counts the rejected cases")
		})
	})
}

// TestValuesAllocs checks the allocation ceiling of Values of one value. It
// does not run in parallel, because its count covers the whole process.
func TestValuesAllocs(t *testing.T) {
	digit := prop.Integer(0, 9)
	assert.MaxAllocs(t, func() { _ = prop.Values(digit, 42, 1) }, valuesAllocs,
		"Values allocates the slice and the case of one value")
}

// BenchmarkValues measures Values of one value.
func BenchmarkValues(b *testing.B) {
	digit := prop.Integer(0, 9)
	var got []int
	c := bench.Start(b).MaxAllocs(valuesAllocs)
	defer c.End()
	for c.Loop() {
		got = prop.Values(digit, 42, 1)
	}
	assert.Length(b, got, 1, "one value")
}
