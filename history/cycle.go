// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"cmp"
	"slices"
)

// relations is a set of relations, a bit for each.
type relations uint8

const (
	// nonRW is the set of the write-write and the write-read dependencies.
	nonRW relations = 1<<WW | 1<<WR
	// allRelations is the set of every relation.
	allRelations relations = nonRW | 1<<RW
)

// has reports whether r is in s.
func (s relations) has(r Relation) bool {
	return s&(1<<r) != 0
}

// ordered returns the relations of s in the order WW, WR, RW.
func (s relations) ordered() []Relation {
	var out []Relation
	for r := WW; r <= RW; r++ {
		if s.has(r) {
			out = append(out, r)
		}
	}
	return out
}

// cycleSearch is how the check finds one kind of cycle. It takes the
// strongly connected components of the graph over the component relations.
// In each, it tries the edges of the closing relation in the order of their
// sources and then their targets, and an edge closes a cycle when its target
// leads back to its source over the path relations.
type cycleSearch struct {
	component relations
	closing   Relation
	path      relations
}

// searchOf returns the search of the cycle a: G0, G1c, G-single or G2.
func searchOf(a Anomaly) cycleSearch {
	switch a {
	case G0:
		return cycleSearch{component: 1 << WW, closing: WW, path: 1 << WW}
	case G1c:
		return cycleSearch{component: nonRW, closing: WR, path: nonRW}
	case GSingle:
		return cycleSearch{component: allRelations, closing: RW, path: nonRW}
	}
	return cycleSearch{component: allRelations, closing: RW, path: allRelations}
}

// hop is one step of a walk over the graph: the number of a transaction, and
// a relation of the edge that the step follows.
type hop struct {
	number   int32
	relation Relation
}

// cycle returns the first cycle of the kind a, as the kind's search finds
// it, and nil for none. The closing edge of a G-single cycle counts as its
// one read-write dependency, and each other edge by the relations that the
// search follows.
func (g *graph) cycle(a Anomaly) *finding {
	s := searchOf(a)
	set := g.componentsOver(s.component)
	for id, component := range set.members {
		for _, from := range component {
			for _, out := range g.arcsOf(from) {
				if set.of[out.to] != int32(id) || !out.relations.has(s.closing) {
					continue
				}
				path := g.shortestPath(out.to, from, s.path, set.of)
				if path == nil {
					continue
				}
				used := make([]relations, len(path))
				used[0] = s.component
				if a == GSingle {
					used[0] = 1 << RW
				}
				for i := 1; i < len(path); i++ {
					used[i] = s.path
				}
				return g.cycleFinding(a, append([]int32{from}, path[:len(path)-1]...), used)
			}
		}
	}
	return nil
}

// nonadjacent returns the first G-nonadjacent cycle, and nil for none. It
// tries each component's read-write edges in the order of their sources and
// then their targets. From an edge's target, it takes the shortest walk back
// to the edge's source on which no read-write edge follows another, that
// takes one more read-write edge at least, and whose last edge is no
// read-write edge. It reduces the closed walk to a simple cycle with no two
// adjacent read-write edges, and reports that cycle when it has two
// read-write edges or more.
func (g *graph) nonadjacent() *finding {
	set := g.componentsOver(allRelations)
	for id, component := range set.members {
		for _, from := range component {
			for _, out := range g.arcsOf(from) {
				if set.of[out.to] != int32(id) || !out.relations.has(RW) {
					continue
				}
				steps := g.alternating(out.to, from, set.of)
				if steps == nil {
					continue
				}
				walk := []hop{{number: from, relation: RW}}
				at := out.to
				for _, s := range steps {
					walk = append(walk, hop{number: at, relation: s.relation})
					at = s.number
				}
				cycle := simple(walk)
				numbers, used := make([]int32, len(cycle)), make([]relations, len(cycle))
				rw := 0
				for i, h := range cycle {
					numbers[i], used[i] = h.number, nonRW
					if h.relation == RW {
						used[i] = 1 << RW
						rw++
					}
				}
				if rw >= 2 {
					return g.cycleFinding(GNonadjacent, numbers, used)
				}
			}
		}
	}
	return nil
}

// The states of a transaction in the G-nonadjacent search, each an offset
// from four times the transaction's number. A state states whether the walk
// entered the transaction by a read-write edge, and whether the walk has
// taken one.
const (
	// takenState is the offset of a state whose walk has taken a read-write
	// edge.
	takenState = 1
	// afterRWState is the offset of a state that the walk entered by a
	// read-write edge.
	afterRWState = 2
	// states is the number of states of a transaction.
	states = 4
)

// shortestPath returns the numbers of the transactions of the shortest path
// from start to goal over the edges of a relation of s, within the component
// of start, which component states for each number, and nil for none. The
// search visits each transaction's targets in ascending order, so of two
// shortest paths it returns the one that comes first in that order.
func (g *graph) shortestPath(start, goal int32, s relations, component []int32) []int32 {
	g.mark++
	g.seen[start], g.parent[start] = g.mark, start
	g.queue = append(g.queue[:0], start)
	for head := 0; head < len(g.queue); head++ {
		at := g.queue[head]
		if at == goal {
			path := []int32{at}
			for at != start {
				at = g.parent[at]
				path = append(path, at)
			}
			slices.Reverse(path)
			return path
		}
		for _, out := range g.arcsOf(at) {
			if out.relations&s == 0 || component[out.to] != component[start] || g.seen[out.to] == g.mark {
				continue
			}
			g.seen[out.to], g.parent[out.to] = g.mark, at
			g.queue = append(g.queue, out.to)
		}
	}
	return nil
}

// alternating returns the steps of the shortest walk from start to goal
// within the component of start, which component states for each number, on
// which no read-write edge follows another, that takes a read-write edge,
// and whose last edge is no read-write edge, and nil for none. A read-write
// edge leads into start. Each step is the transaction that the step enters,
// and the relation of the edge that it follows. The search tries each
// transaction's edges in the order of their targets and of the relations.
func (g *graph) alternating(start, goal int32, component []int32) []hop {
	g.mark++
	origin := states*start + afterRWState
	g.seen[origin] = g.mark
	g.queue = append(g.queue[:0], origin)
	for head := 0; head < len(g.queue); head++ {
		state := g.queue[head]
		at, afterRW, taken := state/states, state%states&afterRWState != 0, state%states&takenState != 0
		if at == goal && !afterRW && taken {
			var steps []hop
			for ; state != origin; state = g.parent[state] {
				steps = append(steps, hop{number: state / states, relation: g.via[state]})
			}
			slices.Reverse(steps)
			return steps
		}
		for _, out := range g.arcsOf(at) {
			if component[out.to] != component[start] {
				continue
			}
			for r := WW; r <= RW; r++ {
				if !out.relations.has(r) || r == RW && afterRW {
					continue
				}
				next := states * out.to
				if r == RW {
					next += afterRWState
				}
				if taken || r == RW {
					next += takenState
				}
				if g.seen[next] != g.mark {
					g.seen[next], g.parent[next], g.via[next] = g.mark, state, r
					g.queue = append(g.queue, next)
				}
			}
		}
	}
	return nil
}

// simple returns a simple cycle of the closed walk, each hop a transaction
// and the relation of the edge that leaves it, with no two adjacent
// read-write edges. While a transaction occurs twice, the walk splits at the
// first repeat into the part between the two occurrences and the rest. It
// keeps the part between them when that part has no two adjacent read-write
// edges, and the rest otherwise, and one of the two always has none. The
// cycle starts at its first read-write edge.
func simple(walk []hop) []hop {
	for {
		first, second, repeated := firstRepeat(walk)
		if !repeated {
			break
		}
		inner := walk[first:second]
		if alternates(inner) {
			walk = inner
		} else {
			walk = append(slices.Clip(walk[:first]), walk[second:]...)
		}
	}
	start := 0
	for i, h := range walk {
		if h.relation == RW {
			start = i
			break
		}
	}
	return append(slices.Clip(walk[start:]), walk[:start]...)
}

// firstRepeat returns the two positions of the first transaction that the
// walk visits twice, and false when it visits none twice.
func firstRepeat(walk []hop) (int, int, bool) {
	seen := make(map[int32]int, len(walk))
	for position, h := range walk {
		if first, again := seen[h.number]; again {
			return first, position, true
		}
		seen[h.number] = position
	}
	return 0, 0, false
}

// alternates reports whether no read-write edge of the closed walk follows
// another.
func alternates(walk []hop) bool {
	for i, h := range walk {
		if h.relation == RW && walk[(i+1)%len(walk)].relation == RW {
			return false
		}
	}
	return true
}

// cycleFinding returns the finding of the cycle a through the transactions
// of numbers in order, back to the first. used states, for the edge from
// each transaction to the next, the relations that the search followed it
// by. A link states the edge's relations among them, and the explanation the
// evidence of them that the derivation found first.
func (g *graph) cycleFinding(a Anomaly, numbers []int32, used []relations) *finding {
	links, explanation := make([]Link, len(numbers)), make([]Evidence, len(numbers))
	for i, number := range numbers {
		e := g.edges[g.edgeOf[[2]int32{number, numbers[(i+1)%len(numbers)]}]]
		present := e.relations & used[i]
		first := int32(len(g.evidence))
		for _, r := range present.ordered() {
			first = min(first, e.first[r])
		}
		links[i] = Link{Call: g.call(number), Relations: present.ordered()}
		explanation[i] = g.evidence[first]
	}
	return &finding{anomaly: a, involved: numbers, cycle: links, explanation: explanation}
}

// componentSet is the strongly connected components of two transactions or
// more over one set of relations.
type componentSet struct {
	// members are the numbers of the transactions of each component, in
	// ascending order, and the components come in the order of their lowest
	// numbers.
	members [][]int32
	// of is the position in members of the component of each transaction,
	// by its number, and -1 for a transaction in no component.
	of []int32
}

// tarjanFrame is a transaction whose edges Tarjan's search is visiting, and
// the position of the next of them.
type tarjanFrame struct {
	number int32
	next   int32
}

// componentsOver returns the strongly connected components of the graph
// over the edges of a relation of s. The search is Tarjan's, without
// recursion, from each transaction in the order of the numbers.
func (g *graph) componentsOver(s relations) *componentSet {
	if set := g.components[s]; set != nil {
		return set
	}
	n := int32(len(g.transactions))
	order, low := make([]int32, n), make([]int32, n)
	onStack := make([]bool, n)
	for i := range order {
		order[i] = -1
	}
	var stack []int32
	var work []tarjanFrame
	set := &componentSet{of: make([]int32, n)}
	visited := int32(0)
	enter := func(number int32) {
		order[number], low[number] = visited, visited
		visited++
		stack = append(stack, number)
		onStack[number] = true
		work = append(work, tarjanFrame{number: number, next: g.start[number]})
	}
	for root := range n {
		if order[root] >= 0 {
			continue
		}
		enter(root)
		for len(work) > 0 {
			top := &work[len(work)-1]
			if top.next == g.start[top.number+1] {
				number := top.number
				work = work[:len(work)-1]
				if len(work) > 0 {
					parent := work[len(work)-1].number
					low[parent] = min(low[parent], low[number])
				}
				if low[number] == order[number] {
					set.close(&stack, onStack, number)
				}
				continue
			}
			out := g.arcs[top.next]
			top.next++
			if out.relations&s == 0 {
				continue
			}
			if order[out.to] < 0 {
				enter(out.to)
			} else if onStack[out.to] {
				low[top.number] = min(low[top.number], order[out.to])
			}
		}
	}
	slices.SortFunc(set.members, func(a, b []int32) int { return cmp.Compare(a[0], b[0]) })
	for i := range set.of {
		set.of[i] = -1
	}
	for id, members := range set.members {
		for _, number := range members {
			set.of[number] = int32(id)
		}
	}
	g.components[s] = set
	return set
}

// close pops the component of root, the transaction that Tarjan's search
// found to be the root of a component, off stack, and keeps it when it has
// two transactions or more.
func (set *componentSet) close(stack *[]int32, onStack []bool, root int32) {
	var component []int32
	for {
		number := (*stack)[len(*stack)-1]
		*stack = (*stack)[:len(*stack)-1]
		onStack[number] = false
		component = append(component, number)
		if number == root {
			break
		}
	}
	if len(component) > 1 {
		slices.Sort(component)
		set.members = append(set.members, component)
	}
}
