// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

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

	return []Case{
		{Name: "identical ints pass", Args: []any{1, 1}},
		{Name: "identical strings pass", Args: []any{"a", "a"}},
		{
			Name:     "differing ints report want and got",
			Args:     []any{1, 2},
			Fails:    true,
			Contains: []string{"want", "got"},
		},
		{
			Name:     "a nil slice does not equal an empty one",
			Args:     []any{nilSlice, []int{}},
			Fails:    true,
			Contains: []string{"want", "got"},
		},
		{
			Name:     "a nil map does not equal an empty one",
			Args:     []any{nilMap, map[string]int{}},
			Fails:    true,
			Contains: []string{"want", "got"},
		},
		{Name: "two empty slices pass", Args: []any{[]int{}, []int{}}},
		{Name: "equal maps pass", Args: []any{
			map[string]int{"a": 1}, map[string]int{"a": 1},
		}},
		{
			Name:     "different types do not compare",
			Args:     []any{1, "1"},
			Fails:    true,
			Contains: []string{"want", "got"},
		},
		{
			Name:     "zero does not equal false",
			Args:     []any{0, false},
			Fails:    true,
			Contains: []string{"want", "got"},
		},
	}
}

// NotEqualCases are the cases every surface's not-equal assertion must
// produce. Drive them with [RunPair].
func NotEqualCases() []Case {
	var nilSlice []int

	return []Case{
		{Name: "differing ints pass", Args: []any{1, 2}},
		{
			Name:     "identical ints report got",
			Args:     []any{1, 1},
			Fails:    true,
			Contains: []string{"got"},
		},
		{Name: "a nil slice differs from an empty one", Args: []any{nilSlice, []int{}}},
		{Name: "different types differ", Args: []any{1, "1"}},
	}
}
