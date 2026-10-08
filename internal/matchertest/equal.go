// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"math"
	"time"
)

// counter returns a closure of one function literal that returns n.
func counter(n int) func() int { return func() int { return n } }

// instants returns one instant in two locations: two times that the Equal
// method of time.Time equates, and whose fields differ.
func instants() (time.Time, time.Time) {
	noon := time.Date(2026, time.October, 4, 12, 0, 0, 0, time.UTC)
	return noon, noon.In(time.FixedZone("CEST", 2*60*60))
}

// EqualCases are the cases every surface's equal assertion must
// produce. Drive them with [RunPair].
//
// The null-against-empty case is the one that matters most: it is
// where this library departs from the comparison most Go test helpers
// configure, and where the other languages' defaults would disagree if
// nobody wrote it down.
func EqualCases() []Case {
	var nilSlice []int
	var nilMap map[string]int
	nan := math.NaN()
	here, there := instants()

	return []Case{
		{
			Name:      "times that their Equal method equates differ by their fields",
			Args:      []any{here, there},
			Fails:     true,
			Assertion: "equal",
		},
		{Name: "two closures of one function literal compare by their code", Args: []any{counter(1), counter(2)}},
		{
			Name:      "maps with a NaN key differ",
			Args:      []any{map[float64]int{nan: 1}, map[float64]int{nan: 1}},
			Fails:     true,
			Assertion: "equal",
		},
		{Name: "maps whose pointer keys have equal targets pass", Args: []any{
			map[*node]int{{N: 1}: 1}, map[*node]int{{N: 1}: 1},
		}},
		{Name: "identical ints pass", Args: []any{1, 1}},
		{Name: "identical strings pass", Args: []any{"a", "a"}},
		{
			Name:      "differing ints report want and got",
			Args:      []any{1, 2},
			Fails:     true,
			Assertion: "equal",
		},
		{
			Name:      "a nil slice does not equal an empty one",
			Args:      []any{nilSlice, []int{}},
			Fails:     true,
			Assertion: "equal",
		},
		{
			Name:      "a nil map does not equal an empty one",
			Args:      []any{nilMap, map[string]int{}},
			Fails:     true,
			Assertion: "equal",
		},
		{Name: "two empty slices pass", Args: []any{[]int{}, []int{}}},
		{Name: "equal maps pass", Args: []any{
			map[string]int{"a": 1}, map[string]int{"a": 1},
		}},
		{
			Name:      "different types do not compare",
			Args:      []any{1, "1"},
			Fails:     true,
			Assertion: "equal",
		},
		{
			Name:      "zero does not equal false",
			Args:      []any{0, false},
			Fails:     true,
			Assertion: "equal",
		},
	}
}

// NotEqualCases are the cases every surface's not-equal assertion must
// produce. Drive them with [RunPair].
func NotEqualCases() []Case {
	var nilSlice []int
	nan := math.NaN()
	here, there := instants()

	return []Case{
		{Name: "differing ints pass", Args: []any{1, 2}},
		{Name: "times that their Equal method equates differ", Args: []any{here, there}},
		{Name: "maps with a NaN key differ", Args: []any{map[float64]int{nan: 1}, map[float64]int{nan: 1}}},
		{
			Name:      "identical ints report got",
			Args:      []any{1, 1},
			Fails:     true,
			Assertion: "not-equal",
		},
		{Name: "a nil slice differs from an empty one", Args: []any{nilSlice, []int{}}},
		{Name: "different types differ", Args: []any{1, "1"}},
	}
}
