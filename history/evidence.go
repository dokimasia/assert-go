// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/literal"
)

// jsonNull is the JSON text of null, which an entry of an explanation states
// for a value that it does not state.
const jsonNull = "null"

// Link is one transaction of a cycle, with the relations of the edge from it
// to the next transaction of the cycle that the search followed. The last
// link's edge leads back to the first link's transaction.
type Link struct {
	// Call is the index of the transaction's invocation event.
	Call int `json:"call"`
	// Relations are the relations of the edge to the next transaction, in
	// the order WW, WR, RW.
	Relations []Relation `json:"relations"`
}

// Evidence is one entry of an anomaly's explanation: an [Edge] of a cycle,
// or the [Observation] of an anomaly that is no cycle.
type Evidence interface {
	json.Marshaler
	// evidence marks the type as evidence, so that the set of kinds of
	// evidence is closed.
	evidence()
}

// Edge is the evidence of one dependency of a cycle: the two transactions
// that it joins, its relation, and the key and the values that prove it.
type Edge struct {
	// From and To are the calls of the two transactions.
	From, To int
	// Relation is the dependency of To on From.
	Relation Relation
	// Register reports an edge of a register history.
	Register bool
	// Empty reports an RW edge whose From read the empty list, or the initial
	// value of a register, so Value states no value.
	Empty bool
	// Key is the key whose list or register proves the dependency.
	Key any
	// Value is a value of From in Key's version order for [WW], the last
	// value of the list that To read for [WR], and the last value of the
	// list that From read for [RW]. In a register history, it is the value
	// that From installed for WW, and the value that the reading transaction
	// read for WR and RW.
	Value any
	// Next is the value of To that follows Value in Key's version order, for
	// WW and RW, and nil for WR.
	Next any
}

// edgeJSON is an edge as the explanation of a record states it.
type edgeJSON struct {
	From     int             `json:"from"`
	To       int             `json:"to"`
	Relation Relation        `json:"relation"`
	Key      json.RawMessage `json:"key"`
	Value    json.RawMessage `json:"value"`
	Next     json.RawMessage `json:"next,omitempty"`
}

// MarshalJSON returns the edge as the explanation of a record states it: the
// two calls, the relation, and the key and the values as typed literals. The
// value of an RW edge whose From read the empty list is null, and a WR edge
// states no next value.
//
// # Allocation contract
//
// MarshalJSON allocates the literal of each value, and what encoding/json
// allocates for the edge.
func (e Edge) MarshalJSON() ([]byte, error) {
	out := edgeJSON{
		From:     e.From,
		To:       e.To,
		Relation: e.Relation,
		Key:      literal.Detail(e.Key),
		Value:    literal.Detail(e.Value),
	}
	if e.Empty {
		out.Value = json.RawMessage(jsonNull)
	}
	if e.Relation != WR {
		out.Next = literal.Detail(e.Next)
	}
	return json.Marshal(out)
}

// evidence marks an Edge as evidence.
func (Edge) evidence() {}

// Observation is the evidence of an anomaly that is no cycle: the reads and
// the appends that show it. Each anomaly states some fields, and the others
// keep their zero values:
//
//   - [GarbageRead] and [DuplicateAppend]: Calls and Reads of the one read,
//     Key, and Value, the value at fault.
//   - [InternalInconsistency]: Calls and Reads of the one read, Key,
//     Expected, Whole, and Future with HasFuture.
//   - [IncompatibleOrder]: Calls and Reads of the two reads, and Key.
//   - [AbortedRead]: Calls of the one read, Key, Value and Appender.
//   - [IntermediateRead]: Calls of the one read, Key, Value, Appender and
//     Next.
//
// An observation of a register history sets Register, and states the value
// that its read returned in Read rather than a list in Reads. Its internal
// inconsistency states the value the transaction knew in Known rather than
// a list in Expected. Appender is the call that wrote Value.
type Observation struct {
	// Anomaly is the anomaly that the observation shows.
	Anomaly Anomaly
	// Register reports an observation of a register history.
	Register bool
	// Whole reports whether the transaction knew the whole list before the
	// read. It is false when the transaction knew only its own appends at
	// the end of the list. For a register, it reports whether the
	// transaction knew the register's value.
	Whole bool
	// HasFuture reports whether the read contains a value that the
	// transaction appends to Key after the read, or for a register returned
	// a value that the transaction writes to Key after the read.
	HasFuture bool
	// Calls are the calls of the transactions that read.
	Calls []int
	// Key is the key that they read.
	Key any
	// Reads are the lists that they read, one for each of Calls.
	Reads [][]any
	// Value is the value at fault.
	Value any
	// Appender is the call that appended Value.
	Appender int
	// Next is the value that Appender appended to Key after Value.
	Next any
	// Expected is what the transaction knew of the list before the read.
	Expected []any
	// Future is the first value of the read that the transaction appends to
	// Key after the read, when HasFuture reports one.
	Future any
	// Read is the value that a register read returned, and nil for the
	// initial value.
	Read any
	// Known is the value of the register that the transaction knew before
	// the read, and nil when it knew none or knew the initial value.
	Known any
}

// readJSON is the observation of a garbage read or a duplicate append.
type readJSON struct {
	Call  int             `json:"call"`
	Key   json.RawMessage `json:"key"`
	Read  json.RawMessage `json:"read"`
	Value json.RawMessage `json:"value"`
}

// inconsistencyJSON is the observation of an internal inconsistency.
type inconsistencyJSON struct {
	Call     int             `json:"call"`
	Key      json.RawMessage `json:"key"`
	Read     json.RawMessage `json:"read"`
	Expected json.RawMessage `json:"expected"`
	Whole    bool            `json:"whole"`
	Future   json.RawMessage `json:"future"`
}

// orderJSON is the observation of an incompatible order.
type orderJSON struct {
	Calls []int             `json:"calls"`
	Key   json.RawMessage   `json:"key"`
	Reads []json.RawMessage `json:"reads"`
}

// dirtyJSON is the observation of an aborted read or an intermediate read.
type dirtyJSON struct {
	Call     int             `json:"call"`
	Key      json.RawMessage `json:"key"`
	Value    json.RawMessage `json:"value"`
	Appender int             `json:"appender"`
	Next     json.RawMessage `json:"next,omitempty"`
}

// MarshalJSON returns the observation as the explanation of a record states
// it: the fields that its anomaly states, each value and each list as a
// typed literal, in the order the definition lists them.
//
// # Allocation contract
//
// MarshalJSON allocates the literal of each value, and what encoding/json
// allocates for the observation.
func (o Observation) MarshalJSON() ([]byte, error) {
	key := literal.Detail(o.Key)
	switch o.Anomaly {
	case InternalInconsistency:
		out := inconsistencyJSON{Call: o.Calls[0], Key: key, Whole: o.Whole, Future: json.RawMessage(jsonNull)}
		if o.Register {
			out.Read, out.Expected = orNull(o.Read), orNull(o.Known)
		} else {
			out.Read, out.Expected = literal.Detail(o.Reads[0]), literal.Detail(o.Expected)
		}
		if o.HasFuture {
			out.Future = literal.Detail(o.Future)
		}
		return json.Marshal(out)
	case IncompatibleOrder:
		return json.Marshal(orderJSON{Calls: o.Calls, Key: key, Reads: literals(o.Reads)})
	case AbortedRead, IntermediateRead:
		out := dirtyJSON{Call: o.Calls[0], Key: key, Value: literal.Detail(o.Value), Appender: o.Appender}
		if o.Anomaly == IntermediateRead {
			out.Next = literal.Detail(o.Next)
		}
		return json.Marshal(out)
	default:
		out := readJSON{Call: o.Calls[0], Key: key, Value: literal.Detail(o.Value)}
		if o.Register {
			out.Read = orNull(o.Read)
		} else {
			out.Read = literal.Detail(o.Reads[0])
		}
		return json.Marshal(out)
	}
}

// orNull returns the typed literal of a register's value, and null for its
// initial value, which nil states.
func orNull(v any) json.RawMessage {
	if v == nil {
		return json.RawMessage(jsonNull)
	}
	return literal.Detail(v)
}

// evidence marks an Observation as evidence.
func (Observation) evidence() {}
