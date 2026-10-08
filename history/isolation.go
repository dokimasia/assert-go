// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"encoding/json"
	"runtime"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
)

// The assertions of the isolation checks, and the operations that name their
// faults.
const (
	serializableID      = "serializable"
	snapshotIsolationID = "snapshot-isolation"
	serializableOp      = "history.Serializable"
	snapshotIsolationOp = "history.HasSnapshotIsolation"
)

// The names of the detail fields of an isolation check's record, in the
// order the definition lists them.
const (
	// anomalyField is the first kind that the level forbids and the history
	// exhibits, an [Anomaly].
	anomalyField = "anomaly"
	// kindsField is every kind that the level forbids and the history
	// exhibits, a []Anomaly.
	kindsField = "kinds"
	// transactionsField is the transactions that the anomaly involves, a
	// []Transaction.
	transactionsField = "transactions"
	// cycleField is the cycle of a cycle anomaly, a []Link, and nil for any
	// other.
	cycleField = "cycle"
	// explanationField is the evidence of the anomaly, a []Evidence.
	explanationField = "explanation"
)

// kinds is a set of kinds of anomaly, a bit for each.
type kinds uint16

const (
	// everyKind is the set of every kind, which serializability forbids.
	everyKind kinds = 1<<(G2+1) - 1<<GarbageRead
	// snapshotKinds is the set of the kinds that snapshot isolation forbids:
	// every kind but G2, because snapshot isolation permits a cycle in which
	// two read-write dependencies are adjacent.
	snapshotKinds = everyKind &^ (1 << G2)
)

// has reports whether a is in s.
func (s kinds) has(a Anomaly) bool {
	return s&(1<<a) != 0
}

// Serializable checks that the list-append transactions of the history h
// exhibit no anomaly that serializability forbids, among the dependencies
// that their reads reveal, and fails tb with one record of the assertion
// serializable when they do. The record's contract is contract, and its
// location is the call of Serializable. Serializability forbids every kind
// of [Anomaly].
//
// A transaction is a call of the operation "txn" whose arguments are its
// micro-operations, each a []any of three parts: "append", a key and a
// value, or "read", a key and nil. An [OK] completion commits the
// transaction, and its output is a []any that repeats the micro-operations
// with each read's list, as a slice, or nil for the empty list. A [Fail]
// completion aborts it. A transaction whose outcome is unknown, or that is
// pending, is committed when a committed read observed one of its appends.
// Every value is appended to its key once. Keys and values compare by their
// typed literals, so the ints 1 and int64(1) are one value.
//
// The check reads each key's version order from its longest committed
// read, places every committed append that no read observed after that
// order, derives the write-write, write-read and read-write dependencies
// between committed transactions, and searches them for cycles, in the
// order and by the search that the definition fixes. The derivation and the
// searches for G0, G1c and G2 take time linear in the transactions and their
// dependencies. The searches for G-single and G-nonadjacent run a
// breadth-first search per read-write dependency until one closes a cycle,
// which is quadratic in the worst case.
//
// The record's detail states the five fields of the definition:
//
//   - anomaly, an [Anomaly]: the first kind that the history exhibits.
//   - kinds, a []Anomaly: every kind that the history exhibits.
//   - transactions, a []Transaction: the transactions that the anomaly
//     involves.
//   - cycle, a []Link for a cycle, and nil for any other anomaly.
//   - explanation, a []Evidence: an [Edge] for each link of a cycle, or one
//     [Observation].
//
// An [assert.Reporter] seat receives the record. Any other seat receives the
// record's sentence through Fatalf. The call record of a recorded run states
// the detail in the history's JSON form.
//
// # Errors
//
// The check ends the call with a fault for a nil history, and with a fault
// of the kind [ErrTransaction] at the call and the part of a call that is no
// list-append transaction.
//
// # Allocation contract
//
// A check allocates its transactions, the typed literal of each distinct key
// and value, the dependencies with their evidence, and the marks of its
// searches, each linear in the history. A passing check of three
// transactions over one key allocates 89 times.
func Serializable(tb assert.TB, h *History, contract string) {
	tb.Helper()
	isolated(tb, h, contract, serializableID, serializableOp, everyKind)
}

// HasSnapshotIsolation checks that the list-append transactions of the
// history h exhibit no anomaly that snapshot isolation forbids, among the
// dependencies that their reads reveal, and fails tb with one record of the
// assertion snapshot-isolation when they do. Snapshot isolation forbids
// every kind of [Anomaly] but [G2]: it permits a cycle in which two
// read-write dependencies are adjacent, such as a write skew.
//
// The workload, the derivation and its cost, the record and the faults are
// those of [Serializable], and the faults name the operation
// history.HasSnapshotIsolation.
//
// # Allocation contract
//
// A check allocates what a check of [Serializable] allocates. A passing check
// of a write skew allocates 86 times.
func HasSnapshotIsolation(tb assert.TB, h *History, contract string) {
	tb.Helper()
	isolated(tb, h, contract, snapshotIsolationID, snapshotIsolationOp, snapshotKinds)
}

// isolated checks h for the anomalies of forbids, and reports the verdict of
// the assertion id, whose faults name op, to tb.
func isolated(tb assert.TB, h *History, contract, id, op string, forbids kinds) {
	tb.Helper()
	run := matcher.Begin(tb)
	if h == nil {
		run.Fault(matcher.Fatal, id, contract, fault.In(op, fault.New("the history is nil")))
		return
	}
	transactions, identities, err := transactionsOf(h.Events())
	if err != nil {
		run.Fault(matcher.Fatal, id, contract, fault.In(op, err))
		return
	}
	d := newGraph(transactions, identities).check(forbids)
	if d.anomaly == 0 {
		run.Pass(matcher.Fatal, id, contract)
		return
	}
	var pcs [callerFrames]uintptr
	where := matcher.CallerWhere(pcs[:runtime.Callers(1, pcs[:])])
	run.FailRun(matcher.Fatal, assert.Failure{Assertion: id, Contract: contract, Detail: d.fields(), Where: where}, d)
}

// finding is one instance of an anomaly: the numbers of the transactions
// that it involves, in the order the record lists them, its cycle, which is
// nil for an anomaly that is no cycle, and its explanation.
type finding struct {
	anomaly     Anomaly
	involved    []int32
	cycle       []Link
	explanation []Evidence
}

// find returns the first instance of the anomaly a, and nil for none.
func (g *graph) find(a Anomaly) *finding {
	switch a {
	case GarbageRead:
		return g.garbageRead()
	case DuplicateAppend:
		return g.duplicateAppend()
	case InternalInconsistency:
		return g.internalInconsistency()
	case IncompatibleOrder:
		return g.incompatible
	case AbortedRead:
		return g.abortedRead()
	case IntermediateRead:
		return g.intermediateRead()
	case GNonadjacent:
		return g.nonadjacent()
	default:
		return g.cycle(a)
	}
}

// check returns the detail of the record of a check for the kinds of
// forbids: every kind that the history exhibits, in report order, and the
// first instance of the first of them. A history that exhibits none states
// no anomaly.
func (g *graph) check(forbids kinds) isolationDetail {
	var d isolationDetail
	for a := GarbageRead; a <= G2; a++ {
		if !forbids.has(a) {
			continue
		}
		f := g.find(a)
		if f == nil {
			continue
		}
		d.kinds = append(d.kinds, a)
		if d.anomaly != 0 {
			continue
		}
		d.anomaly, d.cycle, d.explanation = a, f.cycle, f.explanation
		d.transactions = make([]Transaction, len(f.involved))
		for i, number := range f.involved {
			d.transactions[i] = g.transactions[number].record
		}
	}
	return d
}

// isolationDetail is the detail of the record of a failing isolation check:
// the five fields of the definition.
type isolationDetail struct {
	anomaly      Anomaly
	kinds        []Anomaly
	transactions []Transaction
	cycle        []Link
	explanation  []Evidence
}

// MarshalJSON returns the detail as the call record of a check states it, in
// the form of the definition's vectors: the kinds by their spellings, each
// transaction in the history's JSON form, null for the cycle of an anomaly
// that is no cycle, and each entry of the explanation with its values as
// typed literals.
func (d isolationDetail) MarshalJSON() ([]byte, error) {
	return json.Marshal(isolationJSON{
		Anomaly: d.anomaly, Kinds: d.kinds, Transactions: d.transactions, Cycle: d.cycle,
		Explanation: d.explanation,
	})
}

// isolationJSON is the detail of a failing isolation check as its call
// record states it, in the order that the definition lists its fields.
type isolationJSON struct {
	Anomaly      Anomaly       `json:"anomaly"`
	Kinds        []Anomaly     `json:"kinds"`
	Transactions []Transaction `json:"transactions"`
	Cycle        []Link        `json:"cycle"`
	Explanation  []Evidence    `json:"explanation"`
}

// fields returns the detail as a failure record states it: each field of
// the definition by its name, with its Go value, and nil for the cycle of an
// anomaly that is no cycle.
func (d isolationDetail) fields() map[string]any {
	fields := map[string]any{
		anomalyField:      d.anomaly,
		kindsField:        d.kinds,
		transactionsField: d.transactions,
		cycleField:        nil,
		explanationField:  d.explanation,
	}
	if d.cycle != nil {
		fields[cycleField] = d.cycle
	}
	return fields
}
