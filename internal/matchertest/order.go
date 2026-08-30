// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import "testing"

// PairwiseInvoke calls a surface's pairwise assertion.
//
// The cases fix the element type at int rather than driving a generic
// one. What varies between surfaces is the call, not the type, and an
// int slice exercises every path a pairwise assertion has.
type PairwiseInvoke func(seat *Seat, items []int, pred func(earlier, later int) bool, msg string)

// RunPairwise drives invoke against every case a pairwise assertion
// must produce.
func RunPairwise(t *testing.T, invoke PairwiseInvoke) {
	t.Helper()

	ascending := func(earlier, later int) bool { return earlier < later }

	cases := []struct {
		name     string
		items    []int
		fails    bool
		contains []string
	}{
		{name: "an ordered slice passes", items: []int{1, 2, 3}},
		{name: "an empty slice passes", items: []int{}},
		{name: "a nil slice passes", items: nil},
		{name: "one item passes", items: []int{7}},
		{
			name:     "a break reports its index and both values",
			items:    []int{1, 5, 3},
			fails:    true,
			contains: []string{"5", "3"},
		},
		{
			name:     "equal neighbours break a strict order",
			items:    []int{1, 1},
			fails:    true,
			contains: []string{"1"},
		},
		{
			name:     "a break at the first pair reports",
			items:    []int{9, 1, 2},
			fails:    true,
			contains: []string{"9", "1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seat := &Seat{}
			invoke(seat, tc.items, ascending, contractMsg)
			checkOutcome(t, seat, tc.fails, tc.contains)
		})
	}
}
