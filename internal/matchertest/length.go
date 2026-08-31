// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

// LengthCases are the cases every surface's length assertion must
// produce. Drive them with [RunPair], reading the second argument as
// the wanted length.
func LengthCases() []Case {
	return []Case{
		{Name: "a slice answers its length", Args: []any{[]int{1, 2, 3}, 3}},
		{Name: "an array answers its length", Args: []any{[2]string{"a", "b"}, 2}},
		{Name: "a map answers its length", Args: []any{map[string]int{"a": 1}, 1}},
		{Name: "a string answers its length", Args: []any{"abcd", 4}},
		{Name: "a nil slice has length zero", Args: []any{[]int(nil), 0}},
		{Name: "a defined slice type answers its length", Args: []any{ids{1, 2}, 2}},
		{
			Name:      "a wrong length reports both",
			Args:      []any{[]int{1}, 3},
			Fails:     true,
			Assertion: "length",
		},
		{
			Name:      "a type with no length reports rather than panicking",
			Args:      []any{42, 1},
			Fails:     true,
			Assertion: "length",
		},
	}
}

// EmptyCases are the cases every surface's empty assertion must
// produce. Drive them with [RunOne].
func EmptyCases() []Case {
	return []Case{
		{Name: "an empty slice passes", Args: []any{[]int{}}},
		{Name: "a nil slice passes", Args: []any{[]int(nil)}},
		{Name: "an empty string passes", Args: []any{""}},
		{Name: "an empty map passes", Args: []any{map[string]int{}}},
		{
			Name:      "a populated slice reports its length",
			Args:      []any{[]int{1, 2}},
			Fails:     true,
			Assertion: "empty",
		},
		{
			Name:      "a type with no length reports rather than panicking",
			Args:      []any{42},
			Fails:     true,
			Assertion: "empty",
		},
	}
}

// NotEmptyCases are the cases every surface's not-empty assertion must
// produce. Drive them with [RunOne].
func NotEmptyCases() []Case {
	return []Case{
		{Name: "a populated slice passes", Args: []any{[]int{1}}},
		{Name: "a non-empty string passes", Args: []any{"a"}},
		{
			Name:      "an empty slice reports",
			Args:      []any{[]int{}},
			Fails:     true,
			Assertion: "not-empty",
		},
		{
			Name:      "a nil slice reports",
			Args:      []any{[]int(nil)},
			Fails:     true,
			Assertion: "not-empty",
		},
	}
}
