// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

// initialValue is the identity of a register's initial value, which no
// write writes.
const initialValue = -1

// externalRead is a committed transaction's first read of a register before
// it writes the register: the number of the transaction, and the value it
// read with its identity, initialValue for the initial value.
type externalRead struct {
	number  int32
	valueID int32
	value   any
}

// versionPair is a value of a register that precedes a later value in every
// version order that the history allows: the value with its identity,
// initialValue for the initial value, and the transaction that installed the
// later value with that value.
type versionPair struct {
	beforeID int32
	number   int32
	before   any
	value    any
}

// deriveRegisters derives the dependencies of each register, in key order.
// The initial value precedes each installed value, the value of a committed
// transaction's last write of a register. A committed transaction's external
// read of a register precedes its own installed value. Each pair gives a
// write-write edge from the transaction that installed its first value, and a
// read-write edge from each transaction whose external read returned it. Each
// external read gives a write-read edge from the transaction that installed
// its value.
func (g *graph) deriveRegisters() {
	reads := make([][]externalRead, g.identities)
	installs := make([][]appendedValue, g.identities)
	for number := range g.transactions {
		if !g.committed[number] {
			continue
		}
		t := &g.transactions[number]
		touched := make(map[int32]bool, len(t.microOperations))
		last := make(map[int32]int, len(t.microOperations))
		for position, m := range t.microOperations {
			if m.read && !touched[m.keyID] && t.record.Kind == OK {
				r := externalRead{number: int32(number), valueID: initialValue}
				if len(m.ids) > 0 {
					r.valueID, r.value = m.ids[0], m.list[0]
				}
				reads[m.keyID] = append(reads[m.keyID], r)
			}
			touched[m.keyID] = true
			if m.write {
				last[m.keyID] = position
			}
		}
		for position, m := range t.microOperations {
			if m.write && last[m.keyID] == position {
				installs[m.keyID] = append(installs[m.keyID],
					appendedValue{number: int32(number), valueID: m.valueID, value: m.value})
			}
		}
	}
	for _, id := range g.keys {
		g.deriveRegister(id, pairsOf(installs[id], reads[id]), reads[id])
	}
}

// pairsOf returns the version pairs of one register, whose installed values
// are installs and whose external reads are reads: for each installed value,
// in the order of the transactions, the initial value, and the value that
// its transaction's external read returned when that is another value.
func pairsOf(installs []appendedValue, reads []externalRead) []versionPair {
	readBy := make(map[int32]externalRead, len(reads))
	for _, r := range reads {
		readBy[r.number] = r
	}
	var pairs []versionPair
	for _, a := range installs {
		pairs = append(pairs, versionPair{beforeID: initialValue, number: a.number, value: a.value})
		if r, read := readBy[a.number]; read && r.valueID != initialValue {
			pairs = append(pairs, versionPair{beforeID: r.valueID, before: r.value, number: a.number, value: a.value})
		}
	}
	return pairs
}

// deriveRegister derives the edges of the register whose identity is id from
// its version pairs and its external reads: the write-write edges of the
// pairs in order, then the write-read and read-write edges of each read.
func (g *graph) deriveRegister(id int32, pairs []versionPair, reads []externalRead) {
	key := g.key[id]
	for _, p := range pairs {
		if p.beforeID != initialValue {
			g.edge(g.installer(id, p.beforeID), p.number,
				Edge{Relation: WW, Register: true, Key: key, Value: p.before, Next: p.value})
		}
	}
	for _, r := range reads {
		if r.valueID != initialValue {
			g.edge(g.installer(id, r.valueID), r.number, Edge{Relation: WR, Register: true, Key: key, Value: r.value})
		}
		for _, p := range pairs {
			if p.beforeID == r.valueID {
				g.edge(r.number, p.number, Edge{
					Relation: RW, Register: true, Key: key, Value: r.value, Next: p.value,
					Empty: r.valueID == initialValue,
				})
			}
		}
	}
}

// installer returns the number of the committed transaction that installed
// the value whose identity is valueID on the register whose identity is id,
// and -1 for a value that no committed transaction installed. A value that
// its writer wrote again later in the same transaction was never installed.
func (g *graph) installer(id, valueID int32) int32 {
	name := [2]int32{id, valueID}
	writer, written := g.appender[name]
	if _, overwritten := g.later[name]; !written || overwritten || !g.committed[writer] {
		return -1
	}
	return writer
}

// internalRegister returns the first read of the register transaction of
// number that contradicts the transaction: a read that returned a value that
// the transaction writes to the register after it, or a read of a register
// that it wrote or read before which returned another value. It returns nil
// for none.
func (g *graph) internalRegister(number int32) *finding {
	t := &g.transactions[number]
	type knowledge struct {
		valueID int32
		value   any
	}
	known := make(map[int32]knowledge)
	for position, m := range t.microOperations {
		if m.write {
			known[m.keyID] = knowledge{valueID: m.valueID, value: m.value}
			continue
		}
		read := knowledge{valueID: initialValue}
		if len(m.ids) > 0 {
			read = knowledge{valueID: m.ids[0], value: m.list[0]}
		}
		future := read.valueID != initialValue && g.writesLater(number, position, read.valueID)
		k, touched := known[m.keyID]
		if !future && (!touched || k.valueID == read.valueID) {
			known[m.keyID] = read
			continue
		}
		o := Observation{
			Anomaly: InternalInconsistency, Register: true, Calls: []int{t.record.Call}, Key: m.key,
			Read: read.value, Known: k.value, Whole: touched,
		}
		if future {
			o.Future, o.HasFuture = read.value, true
		}
		return &finding{anomaly: InternalInconsistency, involved: []int32{number}, explanation: []Evidence{o}}
	}
	return nil
}

// writesLater reports whether the transaction of number writes the value
// whose identity is valueID to the register of its read at position, after
// that read.
func (g *graph) writesLater(number int32, position int, valueID int32) bool {
	t := &g.transactions[number]
	keyID := t.microOperations[position].keyID
	for _, m := range t.microOperations[position+1:] {
		if m.write && m.keyID == keyID && m.valueID == valueID {
			return true
		}
	}
	return false
}
