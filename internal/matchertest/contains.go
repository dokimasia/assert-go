// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

// ContainsCases are the cases every surface's contains assertion must
// produce. Drive them with [RunPair].
//
// Containment depends on the type, and the cases state how: text
// contains a substring, a sequence contains an element, and a map
// contains a key.
func ContainsCases() []Case {
	return []Case{
		{Name: "text contains a substring", Args: []any{"hello world", "lo wo"}},
		{Name: "bytes contain a substring", Args: []any{[]byte("hello"), "ell"}},
		{Name: "a slice contains an element", Args: []any{[]int{1, 2, 3}, 2}},
		{Name: "an array contains an element", Args: []any{[3]int{1, 2, 3}, 3}},
		{Name: "a map contains a key", Args: []any{map[string]int{"a": 1}, "a"}},
		{Name: "a defined string type reads as text", Args: []any{name("hello"), "ell"}},
		{
			Name:      "an absent element reports",
			Args:      []any{[]int{1, 2}, 9},
			Fails:     true,
			Assertion: "contains",
		},
		{
			Name:      "an absent substring reports",
			Args:      []any{"hello", "xyz"},
			Fails:     true,
			Assertion: "contains",
			Detail:    map[string]any{"needle": "xyz"},
		},
		{
			Name:      "an absent map key reports",
			Args:      []any{map[string]int{"a": 1}, "b"},
			Fails:     true,
			Assertion: "contains",
			Detail:    map[string]any{"needle": "b"},
		},
		{
			Name:      "a map key of another type is absent",
			Args:      []any{map[string]int{"a": 1}, 42},
			Fails:     true,
			Assertion: "contains",
		},
		{
			Name:      "a type with no containment reports",
			Args:      []any{42, 4},
			Fails:     true,
			Assertion: "contains",
		},
		{
			Name:      "text reports a needle that is not text",
			Args:      []any{"hello 42", 42},
			Fails:     true,
			Assertion: "contains",
			Detail:    map[string]any{"haystack": "hello 42", "needle": 42},
		},
	}
}

// NotContainsCases are the cases every surface's not-contains
// assertion must produce. Drive them with [RunPair].
func NotContainsCases() []Case {
	return []Case{
		{Name: "an absent element passes", Args: []any{[]int{1, 2}, 9}},
		{Name: "an absent substring passes", Args: []any{"hello", "xyz"}},
		{
			Name:      "a present element reports",
			Args:      []any{[]int{1, 2}, 1},
			Fails:     true,
			Assertion: "not-contains",
		},
		{
			Name:      "a present substring reports",
			Args:      []any{"hello", "ell"},
			Fails:     true,
			Assertion: "not-contains",
		},
		{
			Name:      "a type with no containment reports",
			Args:      []any{42, 4},
			Fails:     true,
			Assertion: "not-contains",
			Detail:    map[string]any{"haystack": 42, "needle": 4},
		},
		{
			Name:      "text reports a needle that is not text",
			Args:      []any{"hello 42", 42},
			Fails:     true,
			Assertion: "not-contains",
		},
	}
}

// ContainsInOrderCases are the cases every surface's ordered
// containment assertion must produce. Drive them with [RunPair],
// reading the second argument as a []string of needles.
func ContainsInOrderCases() []Case {
	return []Case{
		{Name: "needles in order pass", Args: []any{"a-b-c", []string{"a", "b", "c"}}},
		{Name: "one needle passes", Args: []any{"a-b-c", []string{"b"}}},
		{Name: "no needles pass", Args: []any{"anything", []string(nil)}},
		{
			Name:      "needles out of order report the one that broke it",
			Args:      []any{"a-b-c", []string{"c", "b"}},
			Fails:     true,
			Assertion: "contains-in-order",
		},
		{
			Name:      "an absent needle reports",
			Args:      []any{"a-b-c", []string{"a", "z"}},
			Fails:     true,
			Assertion: "contains-in-order",
		},
		{
			Name:      "a repeated needle must match twice",
			Args:      []any{"a-b", []string{"a", "a"}},
			Fails:     true,
			Assertion: "contains-in-order",
		},
		{
			Name:      "a value that is not text reports",
			Args:      []any{42, []string{"4"}},
			Fails:     true,
			Assertion: "contains-in-order",
		},
	}
}
