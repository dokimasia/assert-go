// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"iter"
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/token"
)

// linearProbes is the number of probes findInteger makes one by one before
// it starts doubling.
const linearProbes = 4

// minimizeChoice moves each choice towards its target, in order. An
// integer tries its target and then a binary search over its rank. A float
// or a sequence tries its target.
func (sh *shrinker) minimizeChoice() bool {
	improved := false
	for index := 0; index < len(sh.nodes()) && !sh.spent(); index++ {
		n := sh.nodes()[index]
		if n.r.bounds.Kind() == choice.Integer {
			improved = sh.lower([]int{index}) || improved
			continue
		}
		improved = sh.consider(replaced(sh.nodes(), index, n.r.bounds.Target())) || improved
	}
	return improved
}

// lower moves the integer choices at indices, which share one value and
// one bounds, together towards their target: the target first, then a
// binary search over the value's rank in the key order of its bounds, so a
// value below the target can move to one above it.
func (sh *shrinker) lower(indices []int) bool {
	nodes := sh.nodes()
	bounds := nodes[indices[0]].r.bounds.Integer()
	at := func(rank uint64) bool {
		moved := slices.Clone(nodes)
		for _, index := range indices {
			moved[index].c = integerChoice(bounds.AtRank(rank))
		}
		return sh.consider(moved)
	}
	return search(bounds.Rank(nodes[indices[0]].c.Integer), at)
}

// lowerAndDelete steps each integer towards its target, deleting a later
// span it sizes. Integers are visited in order. A step accepted with a
// deletion is tried again on the same integer, so a count falls one
// element at a time. A step accepted alone moves on to the next integer.
func (sh *shrinker) lowerAndDelete() bool {
	improved := false
	for index := 0; index < len(sh.nodes()) && !sh.spent(); {
		before := sh.best()
		deleted := sh.stepAndDelete(index)
		improved = improved || sh.best() != before
		if !deleted {
			index++
		}
	}
	return improved
}

// stepAndDelete steps the integer at index by one, alone and then with a
// later span deleted, and reports whether a step with a deletion was
// accepted.
//
// When the step alone makes the run record another number of choices, the
// integer sizes what follows it. The step is then tried with each span
// that starts after the integer deleted, at any depth, the span that
// starts last first, and of two that start together the longer first.
func (sh *shrinker) stepAndDelete(index int) bool {
	nodes := sh.nodes()
	n := nodes[index]
	if n.r.bounds.Kind() != choice.Integer || n.c.Integer == n.r.bounds.Integer().Target() {
		return false
	}
	stepped := replaced(nodes, index, integerChoice(towards(n.c.Integer, n.r.bounds.Integer().Target(), 1)))
	if sh.consider(stepped) {
		return false
	}
	size, ok := sh.sizes[token.Encode(choicesOf(stepped))]
	if !ok || size == len(nodes) {
		return false
	}
	for _, span := range latestFirst(sh.spans()) {
		if index < span.Start && sh.consider(without(stepped, span.Start, span.End)) {
			return true
		}
	}
	return false
}

// redistribute moves value from an integer to the next integer with the
// same bounds, for each integer in order: the whole distance to its
// target, then a binary search for the largest amount the property
// accepts. The sum of the two values stays the same.
func (sh *shrinker) redistribute() bool {
	improved := false
	for index := 0; index < len(sh.nodes()) && !sh.spent(); index++ {
		improved = sh.move(index) || improved
	}
	return improved
}

// move moves value from the integer at index to the next integer with its
// bounds.
func (sh *shrinker) move(index int) bool {
	nodes := sh.nodes()
	later, ok := nextAlike(nodes, index)
	if !ok {
		return false
	}
	bounds := nodes[index].r.bounds.Integer()
	value, other, target := nodes[index].c.Integer, nodes[later].c.Integer, bounds.Target()
	if value == target {
		return false
	}
	down := value.Compare(target) == 1
	moved := func(amount uint64) bool {
		var raised, lowered choice.Int
		if down {
			if amount > bounds.Hi().Distance(other) {
				return false
			}
			raised, lowered = other.Add(amount), value.Sub(amount)
		} else {
			if amount > other.Distance(bounds.Lo()) {
				return false
			}
			raised, lowered = other.Sub(amount), value.Add(amount)
		}
		candidate := replaced(replaced(nodes, index, integerChoice(lowered)), later, integerChoice(raised))
		return sh.consider(candidate)
	}
	distance := value.Distance(target)
	if moved(distance) {
		return true
	}
	// low stays below high, so the search narrows until they are adjacent.
	low, high, improved := uint64(0), distance, false
	for high-low != 1 {
		middle := low + (high-low)/2
		if moved(middle) {
			low, improved = middle, true
		} else {
			high = middle
		}
	}
	return improved
}

// lowerTogether moves an integer and the next integer with its bounds
// towards their target by one amount, so their difference stays the same.
// Both must lie on one side of the target. For each integer in order,
// findInteger finds the largest amount the property accepts.
func (sh *shrinker) lowerTogether() bool {
	improved := false
	for index := 0; index < len(sh.nodes()) && !sh.spent(); index++ {
		improved = sh.lowerPair(index) || improved
	}
	return improved
}

// lowerPair moves the integer at index and the next integer with its bounds
// towards their target by one amount.
func (sh *shrinker) lowerPair(index int) bool {
	nodes := sh.nodes()
	later, ok := nextAlike(nodes, index)
	if !ok {
		return false
	}
	target := nodes[index].r.bounds.Integer().Target()
	value, other := nodes[index].c.Integer, nodes[later].c.Integer
	side := value.Compare(target)
	if side == 0 || other.Compare(target) != side {
		return false
	}
	room := min(value.Distance(target), other.Distance(target))
	moved := func(amount uint64) bool {
		a, b := towards(value, target, amount), towards(other, target, amount)
		return sh.consider(replaced(replaced(nodes, index, integerChoice(a)), later, integerChoice(b)))
	}
	return findInteger(room, moved) > 0
}

// minimizeDuplicates moves every group of choices that share one value
// and one bounds, away from their target, towards the target together.
// Groups are visited by their first choice. Integers search their rank;
// floats and sequences try the target.
func (sh *shrinker) minimizeDuplicates() bool {
	improved := false
	for group := 0; ; group++ {
		groups := duplicates(sh.nodes())
		if group >= len(groups) || sh.spent() {
			return improved
		}
		indices := groups[group]
		if sh.nodes()[indices[0]].r.bounds.Kind() == choice.Integer {
			improved = sh.lower(indices) || improved
			continue
		}
		targeted := slices.Clone(sh.nodes())
		for _, index := range indices {
			targeted[index] = atTarget(targeted[index])
		}
		improved = sh.consider(targeted) || improved
	}
}

// deleteAndLower steps each integer towards its target, and deletes data
// before the integer in the same candidate. Integers are visited in order.
// Each step is tried with one span deleted that is not empty and ends at
// or before the integer, the span that starts last first, and of two that
// start together the longer first. Then it is tried with one element
// deleted from a sequence before the integer, the last sequence and its
// last element first, when the sequence is longer than its minimum length.
// The step alone is no candidate of the pass. The pass skips the integer
// at index 0, which has no data before it, and sorts the spans only when a
// later integer is away from its target.
func (sh *shrinker) deleteAndLower() bool {
	return sh.sweep(func() iter.Seq[[]node] {
		return func(yield func([]node) bool) {
			nodes := sh.nodes()
			var spans []Span
			for index, n := range nodes {
				if index == 0 || n.r.bounds.Kind() != choice.Integer || n.c.Integer == n.r.bounds.Integer().Target() {
					continue
				}
				if spans == nil {
					spans = latestFirst(sh.spans())
				}
				stepped := replaced(nodes, index, integerChoice(towards(n.c.Integer, n.r.bounds.Integer().Target(), 1)))
				for _, span := range spans {
					if span.Start < span.End && span.End <= index && !yield(without(stepped, span.Start, span.End)) {
						return
					}
				}
				for at := index - 1; at >= 0; at-- {
					before := stepped[at]
					if before.r.bounds.Kind() != choice.Sequence {
						continue
					}
					elements := before.c.Sequence
					if len(elements) <= before.r.bounds.Sequence().Sizes().Min() {
						continue
					}
					for position := range slices.Backward(elements) {
						kept := slices.Concat(elements[:position], elements[position+1:])
						if !yield(replaced(stepped, at, sequenceChoice(kept))) {
							return
						}
					}
				}
			}
		}
	})
}

// nextAlike returns the index of the next integer choice after index with
// the same bounds as the integer choice at index.
func nextAlike(nodes []node, index int) (int, bool) {
	if nodes[index].r.bounds.Kind() != choice.Integer {
		return 0, false
	}
	for j := index + 1; j < len(nodes); j++ {
		if nodes[j].r.bounds == nodes[index].r.bounds {
			return j, true
		}
	}
	return 0, false
}

// duplicates returns the groups of two or more choices that share one
// value and one bounds, and are away from their target.
func duplicates(nodes []node) [][]int {
	type shared struct {
		bounds choice.Bounds
		value  string
	}
	var order []shared
	groups := make(map[shared][]int)
	for index, n := range nodes {
		if n.key().Compare(atTarget(n).key()) == 0 {
			continue
		}
		s := shared{bounds: n.r.bounds, value: token.Encode([]choice.Choice{n.c})}
		if _, seen := groups[s]; !seen {
			order = append(order, s)
		}
		groups[s] = append(groups[s], index)
	}
	var out [][]int
	for _, s := range order {
		if len(groups[s]) > 1 {
			out = append(out, groups[s])
		}
	}
	return out
}

// findInteger returns a k with f(k) true and f(k + 1) false, given that
// f(0) is true and that f is false above limit. It probes 1 to 4 in turn,
// then doubles until f fails, then bisects. It probes the integers that the
// same search over unbounded integers probes, and takes a probe above limit
// as failed without calling f, so no probe overflows and the search ends
// after at most 64 doublings and 64 bisections.
func findInteger(limit uint64, f func(uint64) bool) uint64 {
	for k := uint64(1); k <= linearProbes; k++ {
		if k > limit || !f(k) {
			return k - 1
		}
	}
	// f accepts low and fails low + gap, which can lie past 2^64. Both
	// searches move gap and not that bound, so gap stays within limit.
	low, gap := uint64(linearProbes), uint64(1)
	for gap <= limit-low && f(low+gap) {
		low, gap = low+gap, low+gap
	}
	for gap > 1 {
		half := gap / 2
		if half <= limit-low && f(low+half) {
			low, gap = low+half, gap-half
		} else {
			gap = half
		}
	}
	return low
}

// search tries position 0, then bisects towards the smallest position that
// at accepts. at runs the candidate at one position of an order whose
// position 0 is the target, and reports whether it was accepted. position
// is the current one, which counts as accepted. The result reports whether
// any candidate was accepted.
func search(position uint64, at func(uint64) bool) bool {
	if position == 0 {
		return false
	}
	if at(0) {
		return true
	}
	low, high, improved := uint64(0), position, false
	for high-low > 1 {
		middle := low + (high-low)/2
		if at(middle) {
			high, improved = middle, true
		} else {
			low = middle
		}
	}
	return improved
}

// towards returns v moved amount towards target. v is not target.
func towards(v, target choice.Int, amount uint64) choice.Int {
	if v.Compare(target) == 1 {
		return v.Sub(amount)
	}
	return v.Add(amount)
}

// integerChoice returns the integer choice of v.
func integerChoice(v choice.Int) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: v}
}
