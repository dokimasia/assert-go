// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
)

// contract is the contract of every check that the tests run.
const contract = "the register is linearizable"

// numbered returns a new map of the integers 0 to 15 to their decimal text,
// which iterates its entries in an order that the runtime chooses anew.
func numbered() map[int]string {
	m := make(map[int]string, 16)
	for i := range 16 {
		m[i] = strconv.Itoa(i)
	}
	return m
}

// The values past the members of each enumeration.
const (
	invalidKind     history.Kind     = 5
	invalidVerdict  history.Verdict  = 3
	invalidLimit    history.Limit    = 4
	invalidAnomaly  history.Anomaly  = 12
	invalidRelation history.Relation = 4
)

// The operations of the specs of the tests.
const (
	write = "write"
	read  = "read"
)

// boom is the value that the panicking functions of the tests raise.
const boom = "boom"

// writeOne are the args of a write of 1.
var writeOne = []any{1}

// errRefused is the error of a call that took no effect.
var errRefused = errors.New("history_test: the connection is refused")

// The names of the detail fields of a check's record.
const (
	outcomeField     = "outcome"
	partitionsField  = "partitions"
	stepsField       = "steps"
	partitionField   = "partition"
	callsField       = "calls"
	concurrencyField = "concurrency"
	linearizedField  = "linearized"
	statesField      = "states"
	candidatesField  = "candidates"
	limitField       = "limit"
)

// linearizableOp is the operation of the faults of a check.
const linearizableOp = "history.Linearizable"

// register is the spec of one value, initially 0. write(v) leaves v, and
// read() outputs the value and leaves it. A read whose outcome is unknown
// accepts any value.
var register = history.Spec[int]{
	Initial: func() int { return 0 },
	Next: func(s int, op history.Operation) []int {
		if op.Name == write {
			return []int{op.Args[0].(int)}
		}
		if op.Returned(s) {
			return []int{s}
		}
		return nil
	},
}

// TestHelpers checks the typed literals of the values of a history's JSON
// forms, which every form writes alike.
func TestHelpers(t *testing.T) {
	t.Parallel()

	t.Run("literals", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []any
			want string
		}{
			{
				name: "states each value as its typed literal, in order",
				give: []any{1, "x"},
				want: `[{"type":"int","value":1},{"type":"string","value":"x"}]`,
			},
			{
				name: "states an empty list for no value",
				give: nil,
				want: `[]`,
			},
			{
				name: "states a value without a typed literal as an opaque literal of its text",
				give: []any{complex(1, 2)},
				want: `[{"type":"opaque","text":"(1+2i)"}]`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				data, err := history.Span{
					Completion: -1,
					Operation:  history.Operation{Name: write, Args: tt.give},
				}.MarshalJSON()
				assert.NoError(t, err, "a span marshals")
				var got struct {
					Args json.RawMessage `json:"args"`
				}
				assert.NoError(t, json.Unmarshal(data, &got), "the span is a JSON object")
				assert.Equal(t, string(got.Args), tt.want, "the args of the span")
			})
		}
	})
}

// recordOK records a call of client of operation with args on keys, which
// completes as OK with output.
func recordOK(h *history.History, client int, operation string, args []any, output any, keys ...any) {
	h.Invoke(client, operation, args, keys...).OK(output)
}

// detailOf checks h against m on a recorder under opts, and returns the
// detail of the record of the check, or nil for a check that passed.
func detailOf[S any](h *history.History, m history.Spec[S], opts ...history.Option) map[string]any {
	rec := assert.NewRecorder()
	history.Linearizable(rec, h, m, contract, opts...)
	failures := rec.Failures()
	if len(failures) == 0 {
		return nil
	}
	return failures[0].Detail
}

// faultOf checks h against m on a seat of internal/matchertest under opts,
// and returns the fault that ends the check, which tb requires.
func faultOf[S any](tb testing.TB, h *history.History, m history.Spec[S], opts ...history.Option) *fault.Error {
	tb.Helper()
	seat := &matchertest.Seat{}
	history.Linearizable(seat, h, m, contract, opts...)
	faults := seat.Faults()
	assert.Length(tb, faults, 1, "one fault ends the check")
	assert.Empty(tb, seat.Records(), "no record of linearizable")
	got, ok := errors.AsType[*fault.Error](faults[0])
	assert.True(tb, ok, "the error is a fault")
	return got
}

// callRecord is the part of a call record that the tests read.
type callRecord struct {
	Assertion string          `json:"assertion"`
	Verdict   string          `json:"verdict"`
	Detail    json.RawMessage `json:"detail"`
}

// recordOf returns the call record of the one call that rec recorded, which
// tb requires.
func recordOf(tb testing.TB, rec *assert.Recorder) callRecord {
	tb.Helper()
	records := rec.Records()
	assert.Length(tb, records, 1, "the call record of the check")
	var got callRecord
	assert.NoError(tb, json.Unmarshal([]byte(records[0]), &got), "the call record is a JSON object")
	return got
}

// fatalSeat is a seat without Report, which keeps the text of each failure
// that it receives.
type fatalSeat struct {
	// texts are the texts of the failures, in order.
	texts []string
}

// Helper does nothing.
func (*fatalSeat) Helper() {}

// Fatalf keeps the text of a failure.
func (s *fatalSeat) Fatalf(format string, args ...any) {
	s.texts = append(s.texts, fmt.Sprintf(format, args...))
}

// Errorf keeps the text of a failure.
func (s *fatalSeat) Errorf(format string, args ...any) {
	s.texts = append(s.texts, fmt.Sprintf(format, args...))
}

// expectFault checks that got has the operation, the path, the kind and the
// reason of want.
func expectFault(tb testing.TB, got *fault.Error, want fault.Error) {
	tb.Helper()
	assert.Equal(tb, fault.Error{Op: got.Op, Path: got.Path, Kind: got.Kind, Reason: got.Reason}, want,
		"the operation, the path, the kind and the reason of the fault")
}

// violatedRead returns a history in which client 0 writes 1 and then client
// 1 reads 0, which no order of the register explains, on the key x.
func violatedRead() *history.History {
	h := history.New()
	recordOK(h, 0, write, []any{1}, nil, "x")
	recordOK(h, 1, read, nil, 0, "x")
	return h
}

// parityWrites returns a history in which two clients write 1 and 3 at once,
// and client 0 then reads 2, which no order explains.
func parityWrites() *history.History {
	h := history.New()
	one := h.Invoke(0, write, []any{1}, "x")
	three := h.Invoke(1, write, []any{3}, "x")
	one.OK(nil)
	three.OK(nil)
	recordOK(h, 0, read, nil, 2, "x")
	return h
}

// The errors that end the transactions of the tests that do not commit.
var (
	// errAborted is the error of a transaction that the store aborted.
	errAborted = errors.New("history_test: the store aborted the transaction")
	// errLost is the error of a transaction whose reply was lost.
	errLost = errors.New("history_test: the reply of the store was lost")
)

// isolationContract is the contract of every isolation check that the tests
// run.
const isolationContract = "the store is isolated"

// The operations of the faults of the isolation checks.
const (
	serializableOp      = "history.Serializable"
	snapshotIsolationOp = "history.HasSnapshotIsolation"
)

// The names of the detail fields of an isolation check's record.
const (
	anomalyField      = "anomaly"
	kindsField        = "kinds"
	transactionsField = "transactions"
	cycleField        = "cycle"
	explanationField  = "explanation"
)

// appendOf returns the micro-operation that appends value to key.
func appendOf(key, value any) []any {
	return []any{"append", key, value}
}

// readOf returns the micro-operation that reads key, with values as the list
// that the read returned.
func readOf(key any, values ...any) []any {
	return []any{"read", key, append([]any{}, values...)}
}

// transact records a transaction of client whose micro-operations ran as
// operations, each read with the list that it returned, and completes it as
// kind: OK with operations as its output, Fail or Unknown, and not at all
// for Invoke. Its invocation states each read with nil, and touches the key
// of each micro-operation.
func transact(h *history.History, client int, kind history.Kind, operations ...[]any) {
	args, output := statedBy(operations)
	keys := make([]any, len(operations))
	for i, m := range operations {
		keys[i] = m[1]
	}
	call := h.Invoke(client, "txn", args, keys...)
	switch kind {
	case history.OK:
		call.OK(output)
	case history.Fail:
		call.Fail(errAborted)
	case history.Unknown:
		call.Unknown(errLost)
	}
}

// statedBy returns the arguments of the invocation of a transaction whose
// micro-operations ran as operations, each read with nil, and the output of
// its OK completion, which repeats operations.
func statedBy(operations [][]any) (args, output []any) {
	args, output = make([]any, len(operations)), make([]any, len(operations))
	for i, m := range operations {
		args[i], output[i] = m, m
		if m[0] == "read" {
			args[i] = []any{"read", m[1], nil}
		}
	}
	return args, output
}

// committedAt returns the transaction that transact records with its
// invocation at the event call, on process, and its OK completion at the
// event after it.
func committedAt(call, process int, operations ...[]any) history.Transaction {
	args, output := statedBy(operations)
	return history.Transaction{
		Call: call, Completion: call + 1, Kind: history.OK, Process: process, Args: args, Output: output,
	}
}

// isolationOf checks h with level on a recorder, and returns the detail of
// the record of the check, or nil for a check that passed.
func isolationOf(level func(assert.TB, *history.History, string), h *history.History) map[string]any {
	rec := assert.NewRecorder()
	level(rec, h, isolationContract)
	failures := rec.Failures()
	if len(failures) == 0 {
		return nil
	}
	return failures[0].Detail
}

// isolationFault checks h with level on a seat of internal/matchertest, and
// returns the fault that ends the check, which tb requires.
func isolationFault(tb testing.TB, level func(assert.TB, *history.History, string), h *history.History) *fault.Error {
	tb.Helper()
	seat := &matchertest.Seat{}
	level(seat, h, isolationContract)
	faults := seat.Faults()
	assert.Length(tb, faults, 1, "one fault ends the check")
	assert.Empty(tb, seat.Records(), "no record of the check")
	got, ok := errors.AsType[*fault.Error](faults[0])
	assert.True(tb, ok, "the error is a fault")
	return got
}

// writeSkew returns the history of a write skew: each of two transactions
// reads the key that the other appends to, and finds it empty.
func writeSkew() *history.History {
	h := history.New()
	transact(h, 0, history.OK, readOf("x"), appendOf("y", 1))
	transact(h, 1, history.OK, readOf("y"), appendOf("x", 2))
	return h
}

// spreading returns the spec whose write leaves states, from any state, and
// whose read every state rejects. Its initial state is initial.
func spreading[S any](initial S, states ...S) history.Spec[S] {
	return history.Spec[S]{
		Initial: func() S { return initial },
		Next: func(_ S, op history.Operation) []S {
			if op.Name == write {
				return states
			}
			return nil
		},
	}
}
