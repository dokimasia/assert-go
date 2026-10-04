// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

// allocRuns is the number of calls whose allocations an allocation check
// averages.
const allocRuns = 100

// link is a node of a list, which can contain itself.
type link struct {
	ID   int
	Next *link
}

// hidden is a struct with an unexported field.
type hidden struct {
	Shown int
	kept  int
}

// holder is a struct with a field of an interface type.
type holder struct {
	V any
}

// spot is a map key of two ints, which == compares as equal does.
type spot struct {
	X, Y int
}

// list returns a list of n links with the IDs 0 to n - 1, and the last link
// with the ID last.
func list(n, last int) *link {
	head := &link{ID: last}
	for i := n - 2; i >= 0; i-- {
		head = &link{ID: i, Next: head}
	}
	return head
}
