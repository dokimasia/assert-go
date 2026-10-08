// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package align

// maxCells is the most pairs of elements that Edits aligns by their
// longest common subsequence, after the elements that the two sequences
// share at their start and at their end.
const maxCells = 1 << 20

// Op states which of two sequences have an element of an alignment.
type Op uint8

// The ops of an alignment.
const (
	// Keep is an element that both sequences have.
	Keep Op = iota
	// Delete is an element that the first sequence has alone.
	Delete
	// Insert is an element that the second sequence has alone.
	Insert
)

// Edit is one element of an alignment.
type Edit struct {
	// Op states which sequences have the element.
	Op Op
	// X is the element's index in the first sequence, and -1 for Insert.
	X int
	// Y is the element's index in the second sequence, and -1 for Delete.
	Y int
}

// Edits returns the alignment of a sequence of n elements with one of m
// elements: the longest run of pairs of equal elements that the two share
// in order as Keep, and between them the elements that each has alone, in
// the order of the two sequences. equal reports whether element i of the
// first sequence equals element j of the second.
//
// Edits takes the elements that the two share at their start and at their
// end first, and aligns the rest by its longest common subsequence. A rest
// of more than 1,048,576 pairs of elements shares no element: each of its
// elements is one sequence's alone. Of two alignments of one length, Edits
// lists an element of the first sequence before one of the second.
//
// # Allocation contract
//
// Edits allocates its edits, and for a rest of x and y elements a table of
// (x+1)·(y+1) lengths and one of x·y verdicts of equal.
func Edits(n, m int, equal func(i, j int) bool) []Edit {
	start := 0
	for start < n && start < m && equal(start, start) {
		start++
	}
	end := 0
	for end < n-start && end < m-start && equal(n-1-end, m-1-end) {
		end++
	}
	edits := make([]Edit, 0, n+m)
	for i := range start {
		edits = append(edits, Edit{Op: Keep, X: i, Y: i})
	}
	edits = appendRest(edits, start, n-end, start, m-end, equal)
	for k := range end {
		edits = append(edits, Edit{Op: Keep, X: n - end + k, Y: m - end + k})
	}
	return edits
}

// appendRest appends the alignment of the elements from x0 up to x1 of the
// first sequence with the elements from y0 up to y1 of the second.
func appendRest(edits []Edit, x0, x1, y0, y1 int, equal func(i, j int) bool) []Edit {
	rows, cols := x1-x0, y1-y0
	if rows > 0 && cols > maxCells/rows {
		return appendApart(edits, x0, x1, y0, y1)
	}
	// same[i*cols+j] reports whether element x0+i equals element y0+j, and
	// length[i*width+j] is the length of the longest common subsequence of
	// the elements from x0+i and from y0+j on.
	width := cols + 1
	same := make([]bool, rows*cols)
	length := make([]int32, (rows+1)*width)
	for i := rows - 1; i >= 0; i-- {
		for j := cols - 1; j >= 0; j-- {
			same[i*cols+j] = equal(x0+i, y0+j)
			if same[i*cols+j] {
				length[i*width+j] = length[(i+1)*width+j+1] + 1
			} else {
				length[i*width+j] = max(length[(i+1)*width+j], length[i*width+j+1])
			}
		}
	}
	i, j := 0, 0
	for i < rows && j < cols {
		if same[i*cols+j] {
			edits = append(edits, Edit{Op: Keep, X: x0 + i, Y: y0 + j})
			i, j = i+1, j+1
			continue
		}
		if length[(i+1)*width+j] >= length[i*width+j+1] {
			edits = append(edits, Edit{Op: Delete, X: x0 + i, Y: -1})
			i++
			continue
		}
		edits = append(edits, Edit{Op: Insert, X: -1, Y: y0 + j})
		j++
	}
	return appendApart(edits, x0+i, x1, y0+j, y1)
}

// appendApart appends the elements from x0 up to x1 of the first sequence
// as Delete, and then the elements from y0 up to y1 of the second as
// Insert.
func appendApart(edits []Edit, x0, x1, y0, y1 int) []Edit {
	for i := x0; i < x1; i++ {
		edits = append(edits, Edit{Op: Delete, X: i, Y: -1})
	}
	for j := y0; j < y1; j++ {
		edits = append(edits, Edit{Op: Insert, X: -1, Y: j})
	}
	return edits
}
