// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"cmp"
	"slices"
)

// committedRead is one read of a committed transaction: the number of the
// transaction, and the micro-operation that read.
type committedRead struct {
	number int32
	*microOperation
}

// appendedValue is one append to a key: the number of the transaction that
// appended, and the value with its identity.
type appendedValue struct {
	number  int32
	valueID int32
	value   any
}

// versions is the version order of one key.
type versions struct {
	// key is the key, as the history first states it.
	key any
	// order are the values of the key's longest committed read, in order.
	order []any
	// writers are the numbers of the committed transactions that appended the
	// values of order, and -1 for a value that no committed transaction
	// appended.
	writers []int32
	// unobserved are the committed appends to the key that no committed read
	// observed, in the order of the transactions.
	unobserved []appendedValue
}

// edge is the dependency of one committed transaction on another: the
// numbers of the two, the relations of the dependency, and for each relation
// the position in the graph's evidence of its first evidence.
type edge struct {
	from, to  int32
	relations relations
	first     [RW + 1]int32
}

// arc is an edge in the list of the edges that leave a transaction. It
// states the number of its target, its relations, and its position in the
// graph's edges.
type arc struct {
	to        int32
	edge      int32
	relations relations
}

// graph is the derivation over the transactions of one history: which
// transactions are committed, each key's version order, and the dependencies
// between committed transactions with the evidence of each. A transaction's
// number is its position in the order of the invocations.
//
// A transaction is committed when it completed as OK, or when its outcome is
// unknown or it is pending and a committed read observed one of its
// appends. A key's version order is its longest committed read, and a key
// whose committed reads are not prefixes of one list has none. Every
// committed read is a prefix of the key's final list, so a committed append
// that no committed read observed follows the last value of the version
// order.
type graph struct {
	// transactions are the transactions, in the order of their invocations,
	// and identities is the number of the identities of their keys and
	// values.
	transactions []transaction
	identities   int
	// appender maps the identities of a key and a value to the number of the
	// transaction that appended the value to the key.
	appender map[[2]int32]int32
	// later maps the identities of a key and a value to the value that the
	// same transaction appended to the key next.
	later map[[2]int32]any
	// keys are the identities of the keys, in the order the history first
	// touches them. key states each key as the history first states it, and
	// appends the appends to it in the order of the transactions, by the
	// key's identity.
	keys    []int32
	key     []any
	appends [][]appendedValue
	// committed reports, by number, whether each transaction is committed.
	committed []bool
	// reads are the committed reads, in the order of their transactions and
	// of their micro-operations.
	reads []committedRead
	// incompatible is the first key whose committed reads are not prefixes
	// of one list, and nil for none.
	incompatible *finding
	// edgeOf maps the numbers of two committed transactions to the position
	// in edges of the dependency of the second on the first.
	edgeOf map[[2]int32]int32
	// edges are the dependencies, in the order the derivation finds them,
	// and evidence is the first evidence of each relation of each, in that
	// order.
	edges    []edge
	evidence []Edge
	// start and arcs are the edges by the transaction that they leave: the
	// edges of the transaction n are arcs[start[n]:start[n+1]], in the order
	// of their targets.
	start []int32
	arcs  []arc
	// components are the strongly connected components over each set of
	// relations that a search reads, computed once.
	components [allRelations + 1]*componentSet
	// seen, parent and via are the marks of the breadth-first searches, each
	// with an entry per state of a transaction. A search marks the states
	// that it visits with mark, which each search increments, so no search
	// clears them. queue is the queue of a search.
	seen   []uint32
	parent []int32
	via    []Relation
	mark   uint32
	queue  []int32
}

// newGraph returns the derivation over transactions, the transactions of a
// history in the order of their invocations, whose keys and values have
// identities identities.
func newGraph(transactions []transaction, identities int) *graph {
	n := len(transactions)
	g := &graph{
		transactions: transactions, identities: identities, appender: make(map[[2]int32]int32),
		later: make(map[[2]int32]any), key: make([]any, identities), appends: make([][]appendedValue, identities),
		committed: make([]bool, n), edgeOf: make(map[[2]int32]int32), seen: make([]uint32, states*n),
		parent: make([]int32, states*n), via: make([]Relation, states*n),
	}
	g.index()
	for number := range transactions {
		t := &transactions[number]
		if t.record.Kind != OK {
			continue
		}
		g.committed[number] = true
		for i := range t.microOperations {
			m := &t.microOperations[i]
			if !m.read {
				continue
			}
			m.appenders = make([]int32, len(m.ids))
			for j, id := range m.ids {
				m.appenders[j] = -1
				if appender, appended := g.appender[[2]int32{m.keyID, id}]; appended {
					m.appenders[j] = appender
				}
			}
			g.reads = append(g.reads, committedRead{number: int32(number), microOperation: m})
		}
	}
	for _, r := range g.reads {
		for _, appender := range r.appenders {
			if appender < 0 {
				continue
			}
			if kind := transactions[appender].record.Kind; kind == Unknown || kind == Invoke {
				g.committed[appender] = true
			}
		}
	}
	g.derive()
	g.indexEdges()
	return g
}

// call returns the call of the transaction of number.
func (g *graph) call(number int32) int {
	return g.transactions[number].record.Call
}

// index records the keys that each transaction touches, and the appends
// that it makes.
func (g *graph) index() {
	touched := make([]bool, g.identities)
	// last is the position of the last append of each key in the transaction
	// of number stamp - 1.
	stamp, last := make([]int32, g.identities), make([]int32, g.identities)
	for number := range g.transactions {
		t := &g.transactions[number]
		for position, m := range t.microOperations {
			if !touched[m.keyID] {
				touched[m.keyID], g.key[m.keyID] = true, m.key
				g.keys = append(g.keys, m.keyID)
			}
			if m.read {
				continue
			}
			g.appender[[2]int32{m.keyID, m.valueID}] = int32(number)
			g.appends[m.keyID] = append(g.appends[m.keyID],
				appendedValue{number: int32(number), value: m.value, valueID: m.valueID})
			if stamp[m.keyID] == int32(number)+1 {
				g.later[[2]int32{m.keyID, t.microOperations[last[m.keyID]].valueID}] = m.value
			}
			stamp[m.keyID], last[m.keyID] = int32(number)+1, int32(position)
		}
	}
}

// derive derives the dependencies of each key, in key order. A key whose
// committed reads are not prefixes of one list has no version order. derive
// skips it, and keeps the first such key as the incompatible order.
func (g *graph) derive() {
	byKey := make([][]committedRead, g.identities)
	for _, r := range g.reads {
		byKey[r.keyID] = append(byKey[r.keyID], r)
	}
	// present marks, with the identity of a key plus 1, the identities of the
	// values of the key's version order.
	present := make([]int32, g.identities)
	for _, id := range g.keys {
		reads := byKey[id]
		longest, clash := longestOf(reads)
		if clash != nil {
			if g.incompatible == nil {
				g.incompatible = g.incompatibleOf(g.key[id], clash[0], clash[1])
			}
			continue
		}
		g.deriveKey(g.versionsOf(id, longest, present), reads)
	}
}

// longestOf returns the longest of a key's committed reads, and nil, or the
// first two distinct reads, shortest first, of which the shorter is no
// prefix of the longer. Of two reads of one list, the first counts.
func longestOf(reads []committedRead) (*committedRead, *[2]committedRead) {
	ranked := slices.Clone(reads)
	slices.SortStableFunc(ranked, func(a, b committedRead) int { return cmp.Compare(len(a.ids), len(b.ids)) })
	var longest *committedRead
	for i := range ranked {
		r := &ranked[i]
		if longest != nil && slices.Equal(r.ids, longest.ids) {
			continue
		}
		if longest != nil && !slices.Equal(r.ids[:len(longest.ids)], longest.ids) {
			return nil, &[2]committedRead{*longest, *r}
		}
		longest = r
	}
	return longest, nil
}

// incompatibleOf returns the incompatible order of key that the reads
// shorter and longer show.
func (g *graph) incompatibleOf(key any, shorter, longer committedRead) *finding {
	return &finding{
		anomaly: IncompatibleOrder, involved: []int32{shorter.number, longer.number},
		explanation: []Evidence{Observation{
			Anomaly: IncompatibleOrder, Calls: []int{g.call(shorter.number), g.call(longer.number)}, Key: key,
			Reads: [][]any{shorter.list, longer.list},
		}},
	}
}

// versionsOf returns the version order of the key whose identity is id,
// whose longest committed read is longest, or nil for a key that no
// committed transaction read. It marks the values of the order in present.
func (g *graph) versionsOf(id int32, longest *committedRead, present []int32) versions {
	v := versions{key: g.key[id]}
	if longest != nil {
		v.order, v.writers = longest.list, make([]int32, len(longest.ids))
		for i, valueID := range longest.ids {
			present[valueID] = id + 1
			v.writers[i] = -1
			if appender := longest.appenders[i]; appender >= 0 && g.committed[appender] {
				v.writers[i] = appender
			}
		}
	}
	for _, a := range g.appends[id] {
		if g.committed[a.number] && present[a.valueID] != id+1 {
			v.unobserved = append(v.unobserved, a)
		}
	}
	return v
}

// deriveKey derives the dependencies that one key's version order v and its
// committed reads reveal. The transaction that appended the last value of
// the order precedes each unobserved append, as does every read of the whole
// order.
func (g *graph) deriveKey(v versions, reads []committedRead) {
	for i := 1; i < len(v.order); i++ {
		g.edge(v.writers[i-1], v.writers[i], Edge{Relation: WW, Key: v.key, Value: v.order[i-1], Next: v.order[i]})
	}
	if last := len(v.order) - 1; last >= 0 {
		for _, a := range v.unobserved {
			g.edge(v.writers[last], a.number, Edge{Relation: WW, Key: v.key, Value: v.order[last], Next: a.value})
		}
	}
	for _, r := range reads {
		g.readEdges(r, v)
	}
}

// readEdges derives the write-read and the read-write dependencies of the
// committed read r of the key whose version order is v.
func (g *graph) readEdges(r committedRead, v versions) {
	seen := len(r.list)
	var last any
	if seen > 0 {
		last = r.list[seen-1]
		g.edge(v.writers[seen-1], r.number, Edge{Relation: WR, Key: v.key, Value: last})
	}
	if seen < len(v.order) {
		g.edge(
			r.number,
			v.writers[seen],
			Edge{Relation: RW, Key: v.key, Value: last, Next: v.order[seen], Empty: seen == 0},
		)
		return
	}
	for _, a := range v.unobserved {
		g.edge(r.number, a.number, Edge{Relation: RW, Key: v.key, Value: last, Next: a.value, Empty: seen == 0})
	}
}

// edge adds e as evidence of a dependency of the transaction of number to on
// that of number from, two committed transactions that differ, and adds
// nothing for a transaction that is not committed, which -1 states, or for
// one transaction. The edge keeps the first evidence of each relation.
func (g *graph) edge(from, to int32, e Edge) {
	if from < 0 || to < 0 || from == to {
		return
	}
	pair := [2]int32{from, to}
	position, known := g.edgeOf[pair]
	if !known {
		position = int32(len(g.edges))
		g.edgeOf[pair] = position
		g.edges = append(g.edges, edge{from: from, to: to})
	}
	d := &g.edges[position]
	if d.relations.has(e.Relation) {
		return
	}
	e.From, e.To = g.call(from), g.call(to)
	d.relations |= 1 << e.Relation
	d.first[e.Relation] = int32(len(g.evidence))
	g.evidence = append(g.evidence, e)
}

// indexEdges indexes the edges by the transaction that they leave, each
// transaction's in the order of their targets.
func (g *graph) indexEdges() {
	g.start = make([]int32, len(g.transactions)+1)
	for _, e := range g.edges {
		g.start[e.from+1]++
	}
	for n := 1; n < len(g.start); n++ {
		g.start[n] += g.start[n-1]
	}
	next := slices.Clone(g.start)
	g.arcs = make([]arc, len(g.edges))
	for position, e := range g.edges {
		g.arcs[next[e.from]] = arc{to: e.to, edge: int32(position), relations: e.relations}
		next[e.from]++
	}
	for n := range g.transactions {
		slices.SortFunc(g.arcsOf(int32(n)), func(a, b arc) int { return cmp.Compare(a.to, b.to) })
	}
}

// arcsOf returns the edges that leave the transaction of number, in the
// order of their targets.
func (g *graph) arcsOf(number int32) []arc {
	return g.arcs[g.start[number]:g.start[number+1]]
}
