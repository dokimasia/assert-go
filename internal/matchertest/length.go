// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

// LengthCases are the cases every surface's length assertion must
// produce. Drive them with [RunPair], reading the second argument as
// the wanted length.
func LengthCases() []Case {
	return []Case{
		{Name: "a slice has a length", Args: []any{[]int{1, 2, 3}, 3}},
		{Name: "an array has a length", Args: []any{[2]string{"a", "b"}, 2}},
		{Name: "a map has a length", Args: []any{map[string]int{"a": 1}, 1}},
		{Name: "a string has a length", Args: []any{"abcd", 4}},
		{Name: "a string has the length of its Unicode scalar values", Args: []any{"naïve", 5}},
		{Name: "a byte slice has the length of its bytes", Args: []any{[]byte("ï"), 2}},
		{Name: "a nil slice has length zero", Args: []any{[]int(nil), 0}},
		{Name: "a defined slice type has a length", Args: []any{ids{1, 2}, 2}},
		{
			Name:      "a wrong length reports both",
			Args:      []any{[]int{1}, 3},
			Fails:     true,
			Assertion: "length",
			Detail:    map[string]any{"want": 3, "got": 1},
		},
		{
			Name:      "a type with no length reports no length",
			Args:      []any{42, 1},
			Fails:     true,
			Assertion: "length",
			Detail:    map[string]any{"want": 1, "got": nil},
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
			Name:      "a populated slice reports its items and its length",
			Args:      []any{[]int{1, 2}},
			Fails:     true,
			Assertion: "empty",
			Detail:    map[string]any{"got": []int{1, 2}, "length": 2},
		},
		{
			Name:      "a type with no length reports the value and no length",
			Args:      []any{42},
			Fails:     true,
			Assertion: "empty",
			Detail:    map[string]any{"got": 42, "length": nil},
		},
		{
			Name:      "nil is no container and reports",
			Args:      []any{nil},
			Fails:     true,
			Assertion: "empty",
			Detail:    map[string]any{"got": nil, "length": nil},
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
			Name:      "an empty slice reports the slice",
			Args:      []any{[]int{}},
			Fails:     true,
			Assertion: "not-empty",
			Detail:    map[string]any{"got": []int{}},
		},
		{
			Name:      "a nil slice reports the nil slice",
			Args:      []any{[]int(nil)},
			Fails:     true,
			Assertion: "not-empty",
			Detail:    map[string]any{"got": []int(nil)},
		},
		{
			Name:      "nil is no container and reports",
			Args:      []any{nil},
			Fails:     true,
			Assertion: "not-empty",
			Detail:    map[string]any{"got": nil},
		},
	}
}
