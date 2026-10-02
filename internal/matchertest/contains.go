// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"math"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

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

// PermutationInvoke calls a surface's permutation assertion over slices of
// any element, so a case can set an int against a float.
type PermutationInvoke func(seat *Seat, got, want []any, msg string, opts ...matcher.Option)

// RunPermutation drives invoke against every case a permutation assertion
// must produce.
func RunPermutation(t *testing.T, invoke PermutationInvoke) {
	t.Helper()

	nan := math.NaN()
	cases := []struct {
		name      string
		got, want []any
		opts      []matcher.Option
		fails     bool
	}{
		{name: "the same elements in another order pass", got: []any{3, 1, 2}, want: []any{1, 2, 3}},
		{name: "repeated elements in another order pass", got: []any{1, 2, 1}, want: []any{1, 1, 2}},
		{name: "two empty slices pass", got: []any{}, want: []any{}},
		{name: "two nil slices pass"},
		{
			name: "an element repeated a different number of times reports both slices",
			got:  []any{1, 1, 2}, want: []any{1, 2, 2}, fails: true,
		},
		{name: "an extra element reports both slices", got: []any{1, 2, 3}, want: []any{1, 2}, fails: true},
		{name: "a missing element reports both slices", got: []any{1, 2}, want: []any{1, 2, 3}, fails: true},
		{
			name: "an element in place of another reports both slices",
			got:  []any{1, 2, 3}, want: []any{1, 2, 4}, fails: true,
		},
		{name: "an int does not match a float", got: []any{1}, want: []any{1.0}, fails: true},
		{name: "a NaN does not match a NaN", got: []any{nan}, want: []any{nan}, fails: true},
		{
			name: "a NaN matches a NaN under EquateNaNs",
			got:  []any{nan}, want: []any{nan}, opts: []matcher.Option{matcher.EquateNaNs()},
		},
		{name: "a nil slice does not match an empty one", want: []any{}, fails: true},
		{
			name: "a nil slice matches an empty one under EquateEmpty",
			want: []any{}, opts: []matcher.Option{matcher.EquateEmpty()},
		},
	}

	relations := make([]relationCase, 0, len(cases))
	for _, tc := range cases {
		want := Case{}
		if tc.fails {
			want = failure("permutation", map[string]any{"want": tc.want, "got": tc.got})
		}
		relations = append(relations, relationCase{
			name:  tc.name,
			drive: func(seat *Seat) { invoke(seat, tc.got, tc.want, contractMsg, tc.opts...) },
			want:  want,
		})
	}
	runRelation(t, relations)
}
