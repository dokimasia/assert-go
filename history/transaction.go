// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"encoding/json"
	"errors"
	"reflect"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/text"
)

// The workload that an isolation check reads: the operation of a
// transaction's call, and the functions of its micro-operations.
const (
	// transactionOperation is the operation of every call of a transaction.
	transactionOperation = "txn"
	// appendFunction appends a value to a key's list.
	appendFunction = "append"
	// readFunction reads a key's list.
	readFunction = "read"
	// microOperationParts is the number of parts of a micro-operation: its
	// function, its key and its value.
	microOperationParts = 3
)

// The members of a call that the path of a refused history names.
const (
	callsMember     = "calls"
	operationMember = "operation"
	argsMember      = "args"
	outputMember    = "output"
)

// ErrTransaction is the kind of the fault of a history that [Serializable]
// and [HasSnapshotIsolation] refuse: a call that is no list-append
// transaction.
var ErrTransaction = errors.New("history: a call that is no list-append transaction")

// Transaction is a transaction that an anomaly involves, as the record of an
// isolation check states it.
type Transaction struct {
	// Call is the index of the transaction's invocation event.
	Call int
	// Completion is the index of its completion event, and -1 for a pending
	// transaction.
	Completion int
	// Kind is its completion's kind: [OK], [Fail] or [Unknown], and [Invoke]
	// for a pending transaction.
	Kind Kind
	// Process is the process that made the call.
	Process int
	// Args are its micro-operations, as its invocation states them.
	Args []any
	// Output is its micro-operations with each read's list, for a transaction
	// that completed as OK, and nil for any other.
	Output any
}

// transactionJSON is a transaction in the history's JSON form.
type transactionJSON struct {
	Call       int               `json:"call"`
	Completion *int              `json:"completion,omitempty"`
	Kind       *Kind             `json:"kind,omitempty"`
	Process    int               `json:"process"`
	Args       []json.RawMessage `json:"args"`
	Output     json.RawMessage   `json:"output,omitempty"`
}

// MarshalJSON returns the transaction in the history's JSON form: the call,
// the completion and its kind, the process, the micro-operations of the
// invocation as typed literals, and the output as a typed literal. A pending
// transaction states no completion and no kind. A transaction that did not
// complete as OK states no output.
//
// # Allocation contract
//
// MarshalJSON allocates the literal of each value, and what encoding/json
// allocates for the transaction.
func (t Transaction) MarshalJSON() ([]byte, error) {
	out := transactionJSON{Call: t.Call, Process: t.Process, Args: literals(t.Args)}
	if t.Completion >= 0 {
		out.Completion, out.Kind = &t.Completion, &t.Kind
	}
	if t.Kind == OK {
		out.Output = literal.Detail(t.Output)
	}
	return json.Marshal(out)
}

// microOperation is one micro-operation of a transaction: an append of a
// value to a key, or a read of a key's list. A key and a value are known by
// the identity of their typed literal.
type microOperation struct {
	// key is the key. keyID is its identity.
	key   any
	keyID int32
	// read reports whether the micro-operation reads a list.
	read bool
	// value is the value that an append appends. valueID is its identity.
	value   any
	valueID int32
	// list is the list that a read of a transaction that completed as OK
	// returned. ids are the identities of its values.
	list []any
	ids  []int32
	// appenders are the numbers of the transactions that appended the values
	// of list to the key, and -1 for a value without an appender.
	appenders []int32
}

// transaction is a call of a history, read as a list-append transaction.
type transaction struct {
	// record is the transaction as the record of a check states it.
	record Transaction
	// microOperations are its micro-operations, each read with its list when
	// the transaction completed as OK.
	microOperations []microOperation
}

// identities numbers the distinct typed literals of the keys and the values
// of a history, so that two values with equal typed literals have one
// identity.
type identities struct {
	// byText maps the text of each typed literal to its identity.
	byText map[string]int32
	// byValue maps each value of a predeclared type without a float to its
	// identity, so that the typed literal of a value that recurs is encoded
	// once.
	byValue map[any]int32
}

// of returns the identity of v. It returns a fault for a value that no typed
// literal states.
func (ids *identities) of(v any) (int32, error) {
	cached := false
	switch v.(type) {
	case bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		cached = true
		if id, ok := ids.byValue[v]; ok {
			return id, nil
		}
	}
	raw, ok := literal.Encode(v)
	if !ok {
		return 0, fault.Of(ErrTransaction, "%T is no value that a typed literal states", v)
	}
	text := identity(raw)
	id, known := ids.byText[text]
	if !known {
		id = int32(len(ids.byText))
		ids.byText[text] = id
	}
	if cached {
		ids.byValue[v] = id
	}
	return id, nil
}

// transactionsOf returns the transactions of events, in the order of their
// invocations, and the number of the identities of their keys and values. It
// returns a fault of the kind [ErrTransaction] at the call and the part of
// the first call that breaks the workload's contract: a call that is no
// transaction, an output that does not repeat its invocation's
// micro-operations, a key or a value without a typed literal, and a value
// that two appends append to one key.
func transactionsOf(events []Event) ([]transaction, int, error) {
	completion := make([]int, len(events))
	for _, e := range events {
		if e.Kind != Invoke {
			completion[e.Call] = e.Index
		}
	}
	ids := &identities{byText: map[string]int32{}, byValue: map[any]int32{}}
	appenders := make(map[[2]int32]int)
	var out []transaction
	for _, e := range events {
		if e.Kind != Invoke {
			continue
		}
		t, err := ids.transactionOf(e, events, completion[e.Index])
		if err == nil {
			err = t.register(appenders)
		}
		if err != nil {
			return nil, 0, fault.At(err, fault.Field(callsMember), fault.Index(e.Index))
		}
		out = append(out, t)
	}
	return out, len(ids.byText), nil
}

// transactionOf returns the transaction of the call that invocation starts,
// one of events, which completes at the event of index completed, and not
// at all for 0. It returns a fault at the part of the call that breaks the
// workload's contract.
func (ids *identities) transactionOf(invocation Event, events []Event, completed int) (transaction, error) {
	if invocation.Operation != transactionOperation {
		return transaction{}, fault.At(fault.Of(ErrTransaction, "%q is no transaction", invocation.Operation),
			fault.Field(operationMember))
	}
	t := transaction{
		record: Transaction{
			Call: invocation.Index, Completion: -1, Kind: Invoke, Process: invocation.Process, Args: invocation.Args,
		},
		microOperations: make([]microOperation, len(invocation.Args)),
	}
	for i, arg := range invocation.Args {
		m, err := ids.invoked(arg)
		if err != nil {
			return transaction{}, fault.At(err, fault.Field(argsMember), fault.Index(i))
		}
		t.microOperations[i] = m
	}
	if completed == 0 {
		return t, nil
	}
	completion := events[completed]
	t.record.Completion, t.record.Kind = completion.Index, completion.Kind
	if completion.Kind != OK {
		return t, nil
	}
	t.record.Output = completion.Output
	if err := ids.returned(&t, completion.Output); err != nil {
		return transaction{}, fault.At(err, fault.Field(outputMember))
	}
	return t, nil
}

// invoked returns the micro-operation that one argument of an invocation
// states. It returns a fault for an argument that is no micro-operation, a
// read that states a list, and a key or a value that no typed literal
// states.
func (ids *identities) invoked(arg any) (microOperation, error) {
	parts, ok := arg.([]any)
	if !ok || len(parts) != microOperationParts || (parts[0] != appendFunction && parts[0] != readFunction) {
		return microOperation{}, fault.Of(ErrTransaction, "%s is no micro-operation", text.Sprintf("%#v", arg))
	}
	keyID, err := ids.of(parts[1])
	if err != nil {
		return microOperation{}, err
	}
	m := microOperation{read: parts[0] == readFunction, key: parts[1], keyID: keyID}
	if m.read {
		if parts[2] != nil {
			return microOperation{}, fault.Of(ErrTransaction, "the read states a list before it ran")
		}
		return m, nil
	}
	if m.valueID, err = ids.of(parts[2]); err != nil {
		return microOperation{}, err
	}
	m.value = parts[2]
	return m, nil
}

// returned fills in the list of each read of t from output, the output of
// t's OK completion, which repeats t's micro-operations. It returns a fault
// for an output that is no list, or does not repeat them.
func (ids *identities) returned(t *transaction, output any) error {
	parts, ok := output.([]any)
	if !ok {
		return fault.Of(ErrTransaction, "the output %s is no list of micro-operations", text.Sprintf("%#v", output))
	}
	if len(parts) != len(t.microOperations) {
		return fault.Of(ErrTransaction, "the output repeats %d of %d micro-operations", len(parts),
			len(t.microOperations))
	}
	for i, part := range parts {
		if err := ids.repeated(&t.microOperations[i], part); err != nil {
			return fault.At(err, fault.Index(i))
		}
	}
	return nil
}

// repeated fills in m's list from part, the micro-operation of the output
// that repeats m. A read that returned nil returned the empty list. It
// returns a fault for a part that does not repeat m's function, key and
// appended value, and for a read whose list is no list.
func (ids *identities) repeated(m *microOperation, part any) error {
	parts, ok := part.([]any)
	if !ok || len(parts) != microOperationParts || parts[0] != m.function() || !ids.same(parts[1], m.keyID) ||
		(!m.read && !ids.same(parts[2], m.valueID)) {

		return fault.Of(ErrTransaction, "%s does not repeat %s", text.Sprintf("%#v", part),
			text.Sprintf("%#v", []any{m.function(), m.key, m.value}))
	}
	if !m.read {
		return nil
	}
	switch list := parts[2].(type) {
	case nil:
		m.list = []any{}
	case []any:
		m.list = list
	default:
		values := reflect.ValueOf(list)
		if values.Kind() != reflect.Slice && values.Kind() != reflect.Array || literal.IsBytes(values.Type()) {
			return fault.Of(ErrTransaction, "the read returned %s, which is no list", text.Sprintf("%#v", list))
		}
		m.list = make([]any, values.Len())
		for i := range values.Len() {
			m.list[i] = values.Index(i).Interface()
		}
	}
	m.ids = make([]int32, len(m.list))
	for i, value := range m.list {
		id, err := ids.of(value)
		if err != nil {
			return err
		}
		m.ids[i] = id
	}
	return nil
}

// same reports whether v has a typed literal whose identity is id.
func (ids *identities) same(v any, id int32) bool {
	got, err := ids.of(v)
	return err == nil && got == id
}

// function returns the function of m.
func (m *microOperation) function() string {
	if m.read {
		return readFunction
	}
	return appendFunction
}

// register records in appenders, by the identities of its key and its
// value, the call of each value that t appends. It returns a fault at the
// argument of an append of a value that an append of t or of an earlier
// transaction already appended to the key.
func (t transaction) register(appenders map[[2]int32]int) error {
	for i, m := range t.microOperations {
		if m.read {
			continue
		}
		name := [2]int32{m.keyID, m.valueID}
		first, appended := appenders[name]
		if !appended {
			appenders[name] = t.record.Call
			continue
		}
		value, key := text.Sprintf("%#v", m.value), text.Sprintf("%#v", m.key)
		reason := fault.Of(ErrTransaction, "%s is appended to %s by calls %d and %d", value, key, first, t.record.Call)
		if first == t.record.Call {
			reason = fault.Of(ErrTransaction, "%s is appended to %s twice by call %d", value, key, first)
		}
		return fault.At(reason, fault.Field(argsMember), fault.Index(i))
	}
	return nil
}
