// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"iter"
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// sequenceDelete deletes 2^k consecutive elements of a sequence, the
// largest k first. Sequences are visited in order, and within one the last
// run first. A deletion keeps the sequence at its minimum length or longer.
func (sh *shrinker) sequenceDelete() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes := sh.nodes()
			for index, n := range nodes {
				if n.r.bounds.Kind() != choice.Sequence {
					continue
				}
				elements, count := n.c.Sequence, len(n.c.Sequence)
				for size := highestPower(count); size >= 1; size /= 2 {
					if count-size < n.r.bounds.Sequence().Sizes().Min() {
						continue
					}
					for first := count - size; first >= 0; first-- {
						kept := slices.Concat(elements[:first], elements[first+size:])
						if !yield(replaced(nodes, index, sequenceChoice(kept))) {
							return
						}
					}
				}
			}
		}
	})
}

// sequenceLower moves each element of each sequence towards 0: 0 first,
// then a binary search. Sequences are visited in order, and their elements
// from the first. The pass visits the indices of the best case it starts
// from, and an index that a lowering cut off has no element.
func (sh *shrinker) sequenceLower() bool {
	improved := false
	for index := range len(sh.nodes()) {
		for position := 0; sh.hasElement(index, position) && !sh.spent(); position++ {
			improved = sh.lowerElement(index, position) || improved
		}
	}
	return improved
}

// hasElement reports whether the best case has a choice at index whose
// sequence has an element at position.
func (sh *shrinker) hasElement(index, position int) bool {
	nodes := sh.nodes()
	return index < len(nodes) && position < len(nodes[index].c.Sequence)
}

// lowerElement moves the element at position of the sequence at index
// towards 0.
func (sh *shrinker) lowerElement(index, position int) bool {
	nodes := sh.nodes()
	elements := nodes[index].c.Sequence
	at := func(element uint64) bool {
		changed := slices.Clone(elements)
		changed[position] = uint32(element)
		return sh.consider(replaced(nodes, index, sequenceChoice(changed)))
	}
	return search(uint64(elements[position]), at)
}

// deleteStructurePair deletes two adjacent free structure choices, from
// the last pair to the first. Deleting the stop flag of one collection and
// the continue flag of the next joins the two collections, which no span
// deletion does.
func (sh *shrinker) deleteStructurePair() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes := sh.nodes()
			for first := len(nodes) - 2; first >= 0; first-- {
				pair := freeStructure(nodes[first]) && freeStructure(nodes[first+1])
				if pair && !yield(without(nodes, first, first+2)) {
					return
				}
			}
		}
	})
}

// freeStructure reports whether a choice decides structure and its bounds
// admit two values or more.
func freeStructure(n node) bool {
	bounds := n.r.bounds.Integer()
	return n.r.structure && bounds.Lo() != bounds.Hi()
}

// sequenceChoice returns the sequence choice of elements.
func sequenceChoice(elements []uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}
