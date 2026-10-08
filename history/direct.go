// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"slices"
)

// readFinding returns the finding of an anomaly that the committed read r
// shows by its value at index i.
func (g *graph) readFinding(a Anomaly, r committedRead, i int) *finding {
	return &finding{
		anomaly: a, involved: []int32{r.number},
		explanation: []Evidence{Observation{
			Anomaly: a, Calls: []int{g.call(r.number)}, Key: r.key, Reads: [][]any{r.list}, Value: r.list[i],
		}},
	}
}

// garbageRead returns the first committed read of a value that no
// transaction appended to the key, and nil for none.
func (g *graph) garbageRead() *finding {
	for _, r := range g.reads {
		if i := slices.Index(r.appenders, -1); i >= 0 {
			return g.readFinding(GarbageRead, r, i)
		}
	}
	return nil
}

// duplicateAppend returns the first committed read that returned one value
// twice, at the second, and nil for none.
func (g *graph) duplicateAppend() *finding {
	// seen marks, with the position of a read plus 1, the identities of the
	// values that the read returned.
	seen := make([]int32, g.identities)
	for position, r := range g.reads {
		for i, id := range r.ids {
			if seen[id] == int32(position)+1 {
				return g.readFinding(DuplicateAppend, r, i)
			}
			seen[id] = int32(position) + 1
		}
	}
	return nil
}

// knowledge is what a transaction knows of a key's list: the whole list once
// it has read it, and before a read only that the list ends with its own
// appends. values are the values it knows, with their identities. owned
// reports whether values and ids are the knowledge's own, which an append
// grows in place, and not the lists of a read, which no append changes.
type knowledge struct {
	whole  bool
	owned  bool
	values []any
	ids    []int32
}

// internalInconsistency returns the first read of a transaction that
// completed as OK which contradicts the transaction, and nil for none.
func (g *graph) internalInconsistency() *finding {
	for number := range g.transactions {
		if g.transactions[number].record.Kind != OK {
			continue
		}
		if f := g.internal(int32(number)); f != nil {
			return f
		}
	}
	return nil
}

// internal returns the first read of the transaction of number that
// contradicts the transaction: a read that contains a value that the
// transaction appends to the key after it, or a read of a key that it
// touched before which is not what it knows of the key. It returns nil for
// none.
func (g *graph) internal(number int32) *finding {
	t := &g.transactions[number]
	known := make(map[int32]knowledge)
	for position, m := range t.microOperations {
		k, touched := known[m.keyID]
		if !m.read {
			if !k.owned {
				k.values, k.ids, k.owned = slices.Clip(k.values), slices.Clip(k.ids), true
			}
			k.values, k.ids = append(k.values, m.value), append(k.ids, m.valueID)
			known[m.keyID] = k
			continue
		}
		future := g.futureOf(number, position)
		if future < 0 && (!touched || agrees(m.ids, k)) {
			known[m.keyID] = knowledge{whole: true, values: m.list, ids: m.ids}
			continue
		}
		expected := k.values
		if expected == nil {
			expected = []any{}
		}
		o := Observation{
			Anomaly: InternalInconsistency, Calls: []int{t.record.Call}, Key: m.key, Reads: [][]any{m.list},
			Expected: expected, Whole: k.whole,
		}
		if future >= 0 {
			o.Future, o.HasFuture = m.list[future], true
		}
		return &finding{anomaly: InternalInconsistency, involved: []int32{number}, explanation: []Evidence{o}}
	}
	return nil
}

// futureOf returns the index of the first value of the read at position of
// the transaction of number that the transaction appends to the read's key
// after the read, and -1 for none.
func (g *graph) futureOf(number int32, position int) int {
	t := &g.transactions[number]
	read := &t.microOperations[position]
	for i, appender := range read.appenders {
		if appender != number {
			continue
		}
		for _, m := range t.microOperations[position+1:] {
			if !m.read && m.keyID == read.keyID && m.valueID == read.ids[i] {
				return i
			}
		}
	}
	return -1
}

// agrees reports whether a read whose values have the identities ids is
// what k knows: the whole list, or a list that ends with k's values.
func agrees(ids []int32, k knowledge) bool {
	if k.whole {
		return slices.Equal(ids, k.ids)
	}
	return len(ids) >= len(k.ids) && slices.Equal(ids[len(ids)-len(k.ids):], k.ids)
}

// abortedRead returns the first committed read of a value that an aborted
// transaction appended, and nil for none.
func (g *graph) abortedRead() *finding {
	for _, r := range g.reads {
		for i, appender := range r.appenders {
			if appender < 0 || g.transactions[appender].record.Kind != Fail {
				continue
			}
			return &finding{
				anomaly: AbortedRead, involved: []int32{r.number, appender},
				explanation: []Evidence{Observation{
					Anomaly: AbortedRead, Calls: []int{g.call(r.number)}, Key: r.key, Value: r.list[i],
					Appender: g.call(appender),
				}},
			}
		}
	}
	return nil
}

// intermediateRead returns the first committed read of a list that ends in
// a value that another transaction followed with an append of its own to
// the key, and nil for none. A transaction's read of its own appends is no
// intermediate read.
func (g *graph) intermediateRead() *finding {
	for _, r := range g.reads {
		last := len(r.ids) - 1
		if last < 0 || r.appenders[last] < 0 || r.appenders[last] == r.number {
			continue
		}
		next, followed := g.later[[2]int32{r.keyID, r.ids[last]}]
		if !followed {
			continue
		}
		appender := r.appenders[last]
		return &finding{
			anomaly: IntermediateRead, involved: []int32{r.number, appender},
			explanation: []Evidence{Observation{
				Anomaly: IntermediateRead, Calls: []int{g.call(r.number)}, Key: r.key, Value: r.list[last],
				Appender: g.call(appender), Next: next,
			}},
		}
	}
	return nil
}
