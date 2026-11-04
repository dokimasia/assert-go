// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"slices"
)

// readFinding returns the finding of an anomaly that the committed read r
// shows by its value at index i.
func (g *graph) readFinding(a Anomaly, r committedRead, i int) *finding {
	o := Observation{Anomaly: a, Calls: []int{g.call(r.number)}, Key: r.key, Reads: [][]any{r.list}, Value: r.list[i]}
	if g.registers {
		o.Register, o.Reads, o.Read = true, nil, r.list[i]
	}
	return &finding{anomaly: a, involved: []int32{r.number}, explanation: []Evidence{o}}
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
		internal := g.internal
		if g.registers {
			internal = g.internalRegister
		}
		if f := internal(int32(number)); f != nil {
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
					Anomaly: AbortedRead, Register: g.registers, Calls: []int{g.call(r.number)}, Key: r.key,
					Value: r.list[i], Appender: g.call(appender),
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
				Anomaly: IntermediateRead, Register: g.registers, Calls: []int{g.call(r.number)}, Key: r.key,
				Value: r.list[last], Appender: g.call(appender), Next: next,
			}},
		}
	}
	return nil
}

// lostUpdate returns the first two transactions that completed as OK, read
// one list or value of a key by their first micro-operation on the key, and
// append or write to the key, and nil for none. Such a write follows the
// read. The transactions are taken in the order of their invocations, so the
// finding is the first transaction whose read returned what the read of an
// earlier one returned, and that earlier one.
func (g *graph) lostUpdate() *finding {
	// touched and written mark, with the number of a transaction plus 1, the
	// keys that the transaction touched before a micro-operation, and the
	// keys that it appends or writes to.
	touched, written := make([]int32, g.identities), make([]int32, g.identities)
	// readers maps the identity of a key, the length of a read and the
	// identity of its last value, initialValue for none, to the reads with
	// that key, length and last value, each as the number of its transaction
	// and the position of its micro-operation.
	readers := make(map[[3]int32][][2]int32)
	for number := range g.transactions {
		t := &g.transactions[number]
		if t.record.Kind != OK {
			continue
		}
		stamp := int32(number) + 1
		for _, m := range t.microOperations {
			if !m.read {
				written[m.keyID] = stamp
			}
		}
		for position := range t.microOperations {
			m := &t.microOperations[position]
			first := touched[m.keyID] != stamp
			touched[m.keyID] = stamp
			if !m.read || !first || written[m.keyID] != stamp {
				continue
			}
			version := [3]int32{m.keyID, int32(len(m.ids)), initialValue}
			if len(m.ids) > 0 {
				version[2] = m.ids[len(m.ids)-1]
			}
			for _, r := range readers[version] {
				if slices.Equal(g.transactions[r[0]].microOperations[r[1]].ids, m.ids) {
					return g.lostUpdateOf(r[0], int32(number), m)
				}
			}
			readers[version] = append(readers[version], [2]int32{int32(number), int32(position)})
		}
	}
	return nil
}

// lostUpdateOf returns the lost update of the transactions of the numbers
// earlier and later, whose reads returned what the read m returned.
func (g *graph) lostUpdateOf(earlier, later int32, m *microOperation) *finding {
	o := Observation{
		Anomaly: LostUpdate, Register: g.registers, Calls: []int{g.call(earlier), g.call(later)}, Key: g.key[m.keyID],
	}
	switch {
	case !g.registers:
		o.Reads = [][]any{m.list}
	case len(m.list) > 0:
		o.Read = m.list[0]
	}
	return &finding{anomaly: LostUpdate, involved: []int32{earlier, later}, explanation: []Evidence{o}}
}
