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
	// readFunction reads a key's list, or a register's value.
	readFunction = "read"
	// writeFunction writes a value to a register.
	writeFunction = "write"
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
// and [HasSnapshotIsolation] refuse: a call that is no list-append or
// register transaction, or a history that both appends and writes.
var ErrTransaction = errors.New("history: a call that is no list-append or register transaction")

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
	// Output is its micro-operations with each read's list or value, for a
	// transaction that completed as OK, and nil for any other.
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
// value to a key, a write of a value to a register, or a read of a key's
// list or a register's value. A key and a value are known by the identity
// of their typed literal.
type microOperation struct {
	// key is the key. keyID is its identity.
	key   any
	keyID int32
	// read reports whether the micro-operation reads, and write whether it
	// writes a register. A micro-operation that does neither appends.
	read, write bool
	// value is the value that an append appends or a write writes. valueID
	// is its identity.
	value   any
	valueID int32
	// returned is what a read of a transaction that completed as OK
	// returned, before the workload of the history decides how to read it.
	returned any
	// list is the list that a read of a transaction that completed as OK
	// returned. ids are the identities of its values. A register read
	// returned the list of its one value, or the empty list for the initial
	// value.
	list []any
	ids  []int32
	// appenders are the numbers of the transactions that appended the values
	// of list to the key, and -1 for a value without an appender.
	appenders []int32
}

// transaction is a call of a history, read as a list-append or a register
// transaction.
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

// workload is what a history of transactions is: its transactions, the
// number of the identities of their keys and values, and whether it is a
// register history.
type workload struct {
	transactions []transaction
	identities   int
	registers    bool
}

// transactionsOf returns the workload of events, with the transactions in
// the order of their invocations. A history whose micro-operations write is
// a register history, and any other history is a list-append history. It
// returns a fault of the kind [ErrTransaction] at the call and the part of
// the first call that breaks the workload's contract: a call that is no
// transaction, an output that does not repeat its invocation's
// micro-operations, a key or a value without a typed literal, a value that
// two appends or two writes produce on one key, a history that both appends
// and writes, and a list-append read that returned no list.
func transactionsOf(events []Event) (workload, error) {
	completion := make([]int, len(events))
	for _, e := range events {
		if e.Kind != Invoke {
			completion[e.Call] = e.Index
		}
	}
	ids := &identities{byText: map[string]int32{}, byValue: map[any]int32{}}
	producers := make(map[[2]int32]int)
	var out []transaction
	for _, e := range events {
		if e.Kind != Invoke {
			continue
		}
		t, err := ids.transactionOf(e, events, completion[e.Index])
		if err == nil {
			err = t.register(producers)
		}
		if err != nil {
			return workload{}, fault.At(err, fault.Field(callsMember), fault.Index(e.Index))
		}
		out = append(out, t)
	}
	registers, err := registersOf(out)
	if err != nil {
		return workload{}, err
	}
	for i := range out {
		if err := ids.readsOf(&out[i], registers); err != nil {
			return workload{}, fault.At(err, fault.Field(callsMember), fault.Index(out[i].record.Call))
		}
	}
	return workload{transactions: out, identities: len(ids.byText), registers: registers}, nil
}

// registersOf reports whether transactions write, which makes their history a
// register history. It returns a fault at the argument that first both
// appends and writes in a history: the first append after a write, or the
// first write after an append.
func registersOf(transactions []transaction) (bool, error) {
	appends, writes := -1, -1
	for _, t := range transactions {
		for i, m := range t.microOperations {
			if m.read {
				continue
			}
			if m.write && writes < 0 {
				writes = t.record.Call
			}
			if !m.write && appends < 0 {
				appends = t.record.Call
			}
			if appends >= 0 && writes >= 0 {
				return false, fault.At(
					fault.Of(ErrTransaction, "call %d appends and call %d writes, and a history is either "+
						"list-append or registers", appends, writes),
					fault.Field(callsMember), fault.Index(t.record.Call), fault.Field(argsMember), fault.Index(i))
			}
		}
	}
	return writes >= 0, nil
}

// readsOf fills in what each read of t returned, as the workload reads it:
// the list of the one value of a register read, or the empty list for its
// initial value, and the list that a list-append read returned. It returns a
// fault at the output of a list-append read that returned no list.
func (ids *identities) readsOf(t *transaction, registers bool) error {
	if t.record.Kind != OK {
		return nil
	}
	for i := range t.microOperations {
		m := &t.microOperations[i]
		if !m.read {
			continue
		}
		var err error
		if registers {
			err = ids.registerRead(m)
		} else {
			err = ids.listRead(m)
		}
		if err != nil {
			return fault.At(fault.At(err, fault.Index(i)), fault.Field(outputMember))
		}
	}
	return nil
}

// registerRead fills in m's list with the value that the register read
// returned, and leaves it empty for the initial value, which nil states. It
// returns a fault for a value that no typed literal states.
func (ids *identities) registerRead(m *microOperation) error {
	if m.returned == nil {
		m.list, m.ids = []any{}, []int32{}
		return nil
	}
	id, err := ids.of(m.returned)
	if err != nil {
		return err
	}
	m.list, m.ids = []any{m.returned}, []int32{id}
	return nil
}

// listRead fills in m's list with the list that the list-append read
// returned. A read that returned nil returned the empty list. It returns a
// fault for a read whose list is no list, and for a value that no typed
// literal states.
func (ids *identities) listRead(m *microOperation) error {
	switch list := m.returned.(type) {
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
// read that states a value, a write of nil, and a key or a value that no
// typed literal states.
func (ids *identities) invoked(arg any) (microOperation, error) {
	parts, ok := arg.([]any)
	if !ok || len(parts) != microOperationParts ||
		(parts[0] != appendFunction && parts[0] != readFunction && parts[0] != writeFunction) {

		return microOperation{}, fault.Of(ErrTransaction, "%s is no micro-operation", text.Sprintf("%#v", arg))
	}
	keyID, err := ids.of(parts[1])
	if err != nil {
		return microOperation{}, err
	}
	m := microOperation{read: parts[0] == readFunction, write: parts[0] == writeFunction, key: parts[1], keyID: keyID}
	if m.read {
		if parts[2] != nil {
			return microOperation{}, fault.Of(ErrTransaction, "the read states a value before it ran")
		}
		return m, nil
	}
	if m.write && parts[2] == nil {
		return microOperation{}, fault.Of(ErrTransaction, "the write writes nil, the initial value of every key")
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

// repeated keeps what a read returned from part, the micro-operation of the
// output that repeats m. It returns a fault for a part that does not repeat
// m's function, key and appended or written value.
func (ids *identities) repeated(m *microOperation, part any) error {
	parts, ok := part.([]any)
	if !ok || len(parts) != microOperationParts || parts[0] != m.function() || !ids.same(parts[1], m.keyID) ||
		(!m.read && !ids.same(parts[2], m.valueID)) {

		return fault.Of(ErrTransaction, "%s does not repeat %s", text.Sprintf("%#v", part),
			text.Sprintf("%#v", []any{m.function(), m.key, m.value}))
	}
	if m.read {
		m.returned = parts[2]
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
	switch {
	case m.read:
		return readFunction
	case m.write:
		return writeFunction
	default:
		return appendFunction
	}
}

// register records in producers, by the identities of its key and its
// value, the call of each value that t appends or writes. It returns a fault
// at the argument of an append or a write of a value that t or an earlier
// transaction already produced on the key.
func (t transaction) register(producers map[[2]int32]int) error {
	for i, m := range t.microOperations {
		if m.read {
			continue
		}
		name := [2]int32{m.keyID, m.valueID}
		first, produced := producers[name]
		if !produced {
			producers[name] = t.record.Call
			continue
		}
		verb := "appended to"
		if m.write {
			verb = "written to"
		}
		value, key := text.Sprintf("%#v", m.value), text.Sprintf("%#v", m.key)
		reason := fault.Of(ErrTransaction, "%s is %s %s by calls %d and %d", value, verb, key, first, t.record.Call)
		if first == t.record.Call {
			reason = fault.Of(ErrTransaction, "%s is %s %s twice by call %d", value, verb, key, first)
		}
		return fault.At(reason, fault.Field(argsMember), fault.Index(i))
	}
	return nil
}
