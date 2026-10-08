// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package align_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/align"
)

// editsAllocs are the allocations of Edits of two sequences of three
// elements whose middle elements differ: the edits, and the two tables of
// the rest of one element each.
const editsAllocs = 3

// sink receives the edits that a measured call returns.
var sink []align.Edit

// keep, del and ins return the edits of one element.
func keep(x, y int) align.Edit { return align.Edit{Op: align.Keep, X: x, Y: y} }
func del(x int) align.Edit     { return align.Edit{Op: align.Delete, X: x, Y: -1} }
func ins(y int) align.Edit     { return align.Edit{Op: align.Insert, X: -1, Y: y} }

// editsOf returns the alignment of a and b, whose elements compare with ==.
func editsOf(a, b []int) []align.Edit {
	return align.Edits(len(a), len(b), func(i, j int) bool { return a[i] == b[j] })
}

// TestAlign checks the alignment of two sequences.
func TestAlign(t *testing.T) {
	t.Parallel()

	t.Run("Edits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			giveA []int
			giveB []int
			want  []align.Edit
		}{
			{name: "returns no edits for two empty sequences", giveA: nil, giveB: nil, want: []align.Edit{}},
			{
				name:  "keeps every element of two equal sequences",
				giveA: []int{1, 2},
				giveB: []int{1, 2},
				want:  []align.Edit{keep(0, 0), keep(1, 1)},
			},
			{
				name:  "inserts an element at the start",
				giveA: []int{1, 2, 3},
				giveB: []int{0, 1, 2, 3},
				want:  []align.Edit{ins(0), keep(0, 1), keep(1, 2), keep(2, 3)},
			},
			{
				name:  "deletes an element in the middle",
				giveA: []int{1, 2, 3},
				giveB: []int{1, 3},
				want:  []align.Edit{keep(0, 0), del(1), keep(2, 1)},
			},
			{
				name:  "deletes before it inserts an element that differs",
				giveA: []int{1, 2, 3},
				giveB: []int{1, 4, 3},
				want:  []align.Edit{keep(0, 0), del(1), ins(1), keep(2, 2)},
			},
			{
				name:  "keeps the longest common subsequence of a rest",
				giveA: []int{0, 1, 2, 3, 9},
				giveB: []int{8, 1, 3, 2, 3, 7},
				want:  []align.Edit{del(0), ins(0), keep(1, 1), ins(2), keep(2, 3), keep(3, 4), del(4), ins(5)},
			},
			{
				name:  "lists an element of the first sequence first between two alignments of one length",
				giveA: []int{1, 2},
				giveB: []int{2, 1},
				want:  []align.Edit{del(0), keep(1, 0), ins(1)},
			},
			{
				name:  "inserts every element of the second after the first ends",
				giveA: []int{1},
				giveB: []int{1, 2, 3},
				want:  []align.Edit{keep(0, 0), ins(1), ins(2)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, editsOf(tt.giveA, tt.giveB), tt.want, "the alignment")
			})
		}

		t.Run("aligns a rest of 1,048,576 pairs by its longest common subsequence", func(t *testing.T) {
			t.Parallel()
			got := align.Edits(1024, 1024, func(i, j int) bool { return i == 1000 && j == 3 })
			assert.Contains(t, got, keep(1000, 3), "the one pair of equal elements")
		})

		t.Run("shares no element of a rest of more than 1,048,576 pairs", func(t *testing.T) {
			t.Parallel()
			got := align.Edits(1025, 1024, func(i, j int) bool { return i == 1000 && j == 3 })
			assert.Equal(t, [2]align.Edit{got[0], got[1025]}, [2]align.Edit{del(0), ins(0)},
				"every element of the first, then every element of the second")
			assert.Length(t, got, 2049, "one edit for each element")
		})
	})
}

// TestAlignAllocs checks the allocation ceiling of Edits.
func TestAlignAllocs(t *testing.T) {
	a, b := []int{1, 2, 3}, []int{1, 4, 3}
	assert.MaxAllocs(t, func() { sink = editsOf(a, b) }, editsAllocs, "Edits allocates its edits and its tables")
}

// BenchmarkAlign measures Edits of two sequences of three elements whose
// middle elements differ.
func BenchmarkAlign(b *testing.B) {
	x, y := []int{1, 2, 3}, []int{1, 4, 3}

	b.Run("Edits", func(b *testing.B) {
		c := bench.Start(b).MaxAllocs(editsAllocs)
		defer c.End()
		for c.Loop() {
			sink = editsOf(x, y)
		}
		assert.Length(b, sink, 4, "two kept elements, and one of each sequence alone")
	})
}
