// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"cmp"
	"iter"
	"slices"
)

// smallestChunk is the fewest siblings that deleteSpanChunk deletes at
// once. deleteSpan deletes one.
const smallestChunk = 2

// deleteSpanChunk deletes 2^k consecutive siblings, the largest k first,
// the last chunk first. The groups are visited in the order parents
// returns, and k runs down to 1. Deleting a chunk of empty spans leaves the
// best case unchanged, so the shrinker does not run it.
func (sh *shrinker) deleteSpanChunk() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes := sh.nodes()
			for _, group := range groups(sh.spans()) {
				count := len(group)
				for size := highestPower(count); size >= smallestChunk; size /= 2 {
					for first := count - size; first >= 0; first-- {
						start, end := group[first].Start, group[first+size-1].End
						if !yield(without(nodes, start, end)) {
							return
						}
					}
				}
			}
		}
	})
}

// deleteSpan deletes one span, from the span that starts last to the
// first. Of two spans that start together, the longer is tried first.
// Deleting an empty span leaves the best case unchanged, so the shrinker
// does not run it.
func (sh *shrinker) deleteSpan() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes := sh.nodes()
			for _, span := range latestFirst(sh.spans()) {
				if !yield(without(nodes, span.Start, span.End)) {
					return
				}
			}
		}
	})
}

// liftDescendant replaces a span by a descendant with the same label.
// Spans are visited in order, and each one's descendants in order. Lifting
// a descendant as long as its span leaves the best case unchanged, so the
// shrinker does not run it.
func (sh *shrinker) liftDescendant() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes, spans := sh.nodes(), sh.spans()
			for index, span := range spans {
				for _, inner := range descendants(spans, index) {
					if inner.Label != span.Label {
						continue
					}
					lifted := slices.Concat(nodes[:span.Start], nodes[inner.Start:inner.End], nodes[span.End:])
					if !yield(lifted) {
						return
					}
				}
			}
		}
	})
}

// deleteSpanRun deletes as long a run of siblings as findInteger finds, at
// each start. The groups are visited in the order parents returns, which
// keeps the spans of a parent not yet visited where they were. Within a
// group the runs start at the first sibling and move to the last. After a
// deletion, the next start is the sibling after the one that ended the
// run, because deleting it with the run failed.
func (sh *shrinker) deleteSpanRun() bool {
	improved := false
	for _, parent := range parents(sh.spans()) {
		for first := 0; first < len(children(sh.spans(), parent)) && !sh.spent(); first++ {
			original, group := sh.nodes(), children(sh.spans(), parent)
			deleted := func(count uint64) bool {
				return sh.consider(without(original, group[first].Start, group[first+int(count)-1].End))
			}
			if findInteger(uint64(len(group)-first), deleted) > 0 {
				improved = true
			}
		}
	}
	return improved
}

// targetSpan sets every choice of one span to its target, for each span in
// order.
func (sh *shrinker) targetSpan() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes := sh.nodes()
			for _, span := range sh.spans() {
				targeted := slices.Clone(nodes)
				for i := span.Start; i < span.End; i++ {
					targeted[i] = atTarget(targeted[i])
				}
				if !yield(targeted) {
					return
				}
			}
		}
	})
}

// sortSiblings swaps adjacent siblings with one label when the later one
// is smaller. The groups are visited in the order parents returns, and the
// pairs from the first.
func (sh *shrinker) sortSiblings() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes := sh.nodes()
			for _, group := range groups(sh.spans()) {
				for i := 1; i < len(group); i++ {
					left, right := group[i-1], group[i]
					first, second := nodes[left.Start:left.End], nodes[right.Start:right.End]
					if left.Label != right.Label || compareNodes(second, first) != -1 {
						continue
					}
					between := nodes[left.End:right.Start]
					swapped := slices.Concat(nodes[:left.Start], second, between, first, nodes[right.End:])
					if !yield(swapped) {
						return
					}
				}
			}
		}
	})
}

// children returns the spans whose parent is parent, in order.
func children(spans []Span, parent int) []Span {
	var out []Span
	for _, span := range spans {
		if span.Parent == parent {
			out = append(out, span)
		}
	}
	return out
}

// parents returns every parent of a sibling group: the later parents
// first, then the top, -1. A case without spans has the top alone, whose
// group is empty.
func parents(spans []Span) []int {
	var out []int
	for _, span := range spans {
		if span.Parent >= 0 && !slices.Contains(out, span.Parent) {
			out = append(out, span.Parent)
		}
	}
	slices.SortFunc(out, func(a, b int) int { return cmp.Compare(b, a) })
	return append(out, -1)
}

// groups returns the sibling groups, in the order parents returns their
// parents.
func groups(spans []Span) [][]Span {
	var out [][]Span
	for _, parent := range parents(spans) {
		out = append(out, children(spans, parent))
	}
	return out
}

// descendants returns the descendants of the span at index, in order.
func descendants(spans []Span, index int) []Span {
	inside := []int{index}
	var out []Span
	for later := index + 1; later < len(spans); later++ {
		if !slices.Contains(inside, spans[later].Parent) {
			return out
		}
		inside = append(inside, later)
		out = append(out, spans[later])
	}
	return out
}

// latestFirst returns spans sorted by start and then end, the latest
// first.
func latestFirst(spans []Span) []Span {
	sorted := slices.Clone(spans)
	slices.SortStableFunc(sorted, func(a, b Span) int {
		return cmp.Or(cmp.Compare(b.Start, a.Start), cmp.Compare(b.End, a.End))
	})
	return sorted
}

// highestPower returns the largest power of two that is at most count,
// and 0 for a count of 0. It clears the lowest set bit of count until at
// most one bit is set.
func highestPower(count int) int {
	for count&(count-1) != 0 {
		count &= count - 1
	}
	return count
}
