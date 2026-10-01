// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
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
	for index := 0; index < len(sh.nodes()); index++ {
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
	for index := 0; index < len(sh.nodes()); {
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
	for index := 0; index < len(sh.nodes()); index++ {
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
	for index := 0; index < len(sh.nodes()); index++ {
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
	moved := func(amount int) bool {
		if uint64(amount) > room {
			return false
		}
		a, b := towards(value, target, uint64(amount)), towards(other, target, uint64(amount))
		return sh.consider(replaced(replaced(nodes, index, integerChoice(a)), later, integerChoice(b)))
	}
	return findInteger(moved) > 0
}

// minimizeDuplicates moves every group of choices that share one value
// and one bounds, away from their target, towards the target together.
// Groups are visited by their first choice. Integers search their rank;
// floats and sequences try the target.
func (sh *shrinker) minimizeDuplicates() bool {
	improved := false
	for group := 0; ; group++ {
		groups := duplicates(sh.nodes())
		if group >= len(groups) {
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
// f(0) is true. It probes 1 to 4 in turn, then doubles until f fails, then
// bisects.
func findInteger(f func(int) bool) int {
	for k := 1; k <= linearProbes; k++ {
		if !f(k) {
			return k - 1
		}
	}
	// low stays below high, so the bisection narrows until they are adjacent.
	low, high := linearProbes, linearProbes+1
	for f(high) {
		low, high = high, high*2
	}
	for low+1 != high {
		middle := low + (high-low)/2
		if f(middle) {
			low = middle
		} else {
			high = middle
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
