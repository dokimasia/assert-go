// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
)

// transactionJSONAllocs are the allocations of the JSON form of committed,
// measured.
const transactionJSONAllocs = 55

// The typed literals of the micro-operations and of the outputs of the
// tests of the JSON form of a transaction.
const (
	// appendOneLiteral is the literal of appendOf("x", 1).
	appendOneLiteral = `{"type":"list","items":[{"type":"string","value":"append"},{"type":"string","value":"x"},` +
		`{"type":"int","value":1}]}`
	// committedJSON is the JSON form of committed.
	committedJSON = `{"call":0,"completion":1,"kind":"ok","process":0,"args":[` + appendOneLiteral +
		`],"output":{"type":"list","items":[` + appendOneLiteral + `]}}`
)

// committed is a transaction that appended 1 to x and committed.
var committed = history.Transaction{
	Call: 0, Completion: 1, Kind: history.OK, Args: []any{appendOf("x", 1)}, Output: []any{appendOf("x", 1)},
}

// TestTransaction checks the JSON form of a transaction, and the faults of
// a history whose calls are no list-append transactions.
func TestTransaction(t *testing.T) {
	t.Parallel()

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give history.Transaction
			want string
		}{
			{
				name: "states the completion, its kind and the output of a committed transaction",
				give: committed,
				want: committedJSON,
			},
			{
				name: "states the completion and its kind but no output of an aborted transaction",
				give: history.Transaction{
					Call:       2,
					Completion: 3,
					Kind:       history.Fail,
					Process:    1,
					Args:       []any{appendOf("x", 1)},
				},
				want: `{"call":2,"completion":3,"kind":"fail","process":1,"args":[` + appendOneLiteral + `]}`,
			},
			{
				name: "states no completion, no kind and no output of a pending transaction",
				give: history.Transaction{
					Call:       4,
					Completion: -1,
					Kind:       history.Invoke,
					Process:    2,
					Args:       []any{appendOf("x", 1)},
				},
				want: `{"call":4,"process":2,"args":[` + appendOneLiteral + `]}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give.MarshalJSON()
				assert.NoError(t, err, "a transaction marshals")
				assert.Equal(t, string(got), tt.want, "the transaction in the history's JSON form")
			})
		}
	})

	t.Run("Serializable", func(t *testing.T) {
		t.Parallel()

		at := func(segs ...fault.Segment) fault.Path {
			return append(fault.Path{fault.Field("calls")}, segs...)
		}
		tests := []struct {
			name       string
			give       func(h *history.History)
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "refuses a call whose operation is no transaction",
				give:       func(h *history.History) { h.Invoke(0, read, nil).OK(nil) },
				wantPath:   at(fault.Index(0), fault.Field("operation")),
				wantReason: `"read" is no transaction`,
			},
			{
				name:       "refuses an argument that is no list",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{"x"}) },
				wantPath:   at(fault.Index(0), fault.Field("args"), fault.Index(0)),
				wantReason: `"x" is no micro-operation`,
			},
			{
				name:       "refuses an argument of another function than append and read",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{[]any{"write", "x", 1}}) },
				wantPath:   at(fault.Index(0), fault.Field("args"), fault.Index(0)),
				wantReason: `[]interface {}{"write", "x", 1} is no micro-operation`,
			},
			{
				name:       "refuses an argument of two parts",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{[]any{"read", "x"}}) },
				wantPath:   at(fault.Index(0), fault.Field("args"), fault.Index(0)),
				wantReason: `[]interface {}{"read", "x"} is no micro-operation`,
			},
			{
				name:       "refuses a read that states a list before it ran",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{readOf("x")}) },
				wantPath:   at(fault.Index(0), fault.Field("args"), fault.Index(0)),
				wantReason: "the read states a list before it ran",
			},
			{
				name:       "refuses a key that no typed literal states",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{appendOf(make(chan int), 1)}) },
				wantPath:   at(fault.Index(0), fault.Field("args"), fault.Index(0)),
				wantReason: "chan int is no value that a typed literal states",
			},
			{
				name:       "refuses an appended value that no typed literal states",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{appendOf("x", make(chan int))}) },
				wantPath:   at(fault.Index(0), fault.Field("args"), fault.Index(0)),
				wantReason: "chan int is no value that a typed literal states",
			},
			{
				name:       "refuses an output that is no list",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{appendOf("x", 1)}).OK("done") },
				wantPath:   at(fault.Index(0), fault.Field("output")),
				wantReason: `the output "done" is no list of micro-operations`,
			},
			{
				name:       "refuses an output that repeats another number of micro-operations",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{appendOf("x", 1)}).OK([]any{}) },
				wantPath:   at(fault.Index(0), fault.Field("output")),
				wantReason: "the output repeats 0 of 1 micro-operations",
			},
			{
				name:       "refuses an output whose micro-operation appends another value",
				give:       func(h *history.History) { h.Invoke(0, "txn", []any{appendOf("x", 1)}).OK([]any{appendOf("x", 2)}) },
				wantPath:   at(fault.Index(0), fault.Field("output"), fault.Index(0)),
				wantReason: `[]interface {}{"append", "x", 2} does not repeat []interface {}{"append", "x", 1}`,
			},
			{
				name: "refuses an output whose micro-operation reads another key",
				give: func(h *history.History) {
					h.Invoke(0, "txn", []any{[]any{"read", "x", nil}}).OK([]any{readOf("y")})
				},
				wantPath:   at(fault.Index(0), fault.Field("output"), fault.Index(0)),
				wantReason: `[]interface {}{"read", "y", []interface {}{}} does not repeat []interface {}{"read", "x", interface {}(nil)}`,
			},
			{
				name: "refuses an output whose micro-operation states another function",
				give: func(h *history.History) {
					h.Invoke(0, "txn", []any{[]any{"read", "x", nil}}).OK([]any{appendOf("x", 1)})
				},
				wantPath:   at(fault.Index(0), fault.Field("output"), fault.Index(0)),
				wantReason: `[]interface {}{"append", "x", 1} does not repeat []interface {}{"read", "x", interface {}(nil)}`,
			},
			{
				name: "refuses a read whose list is no list",
				give: func(h *history.History) {
					h.Invoke(0, "txn", []any{[]any{"read", "x", nil}}).OK([]any{[]any{"read", "x", 7}})
				},
				wantPath:   at(fault.Index(0), fault.Field("output"), fault.Index(0)),
				wantReason: "the read returned 7, which is no list",
			},
			{
				name: "refuses a read whose list is a byte string",
				give: func(h *history.History) {
					h.Invoke(0, "txn", []any{[]any{"read", "x", nil}}).OK([]any{[]any{"read", "x", []byte{1}}})
				},
				wantPath:   at(fault.Index(0), fault.Field("output"), fault.Index(0)),
				wantReason: "the read returned []byte{0x1}, which is no list",
			},
			{
				name: "refuses a read of a value that no typed literal states",
				give: func(h *history.History) {
					h.Invoke(0, "txn", []any{[]any{"read", "x", nil}}).
						OK([]any{[]any{"read", "x", []any{make(chan int)}}})
				},
				wantPath:   at(fault.Index(0), fault.Field("output"), fault.Index(0)),
				wantReason: "chan int is no value that a typed literal states",
			},
			{
				name:       "refuses a value that one transaction appends to a key twice",
				give:       func(h *history.History) { transact(h, 0, history.OK, appendOf("x", 1), appendOf("x", 1)) },
				wantPath:   at(fault.Index(0), fault.Field("args"), fault.Index(1)),
				wantReason: `1 is appended to "x" twice by call 0`,
			},
			{
				name: "refuses a value that two transactions append to a key",
				give: func(h *history.History) {
					transact(h, 0, history.Fail, appendOf("x", 1))
					transact(h, 1, history.OK, appendOf("x", 1))
				},
				wantPath:   at(fault.Index(2), fault.Field("args"), fault.Index(0)),
				wantReason: `1 is appended to "x" by calls 0 and 2`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := history.New()
				tt.give(h)
				expectFault(t, isolationFault(t, history.Serializable, h), fault.Error{
					Op: serializableOp, Path: tt.wantPath, Kind: history.ErrTransaction, Reason: tt.wantReason,
				})
			})
		}

		t.Run("reads the ints 1 and int64(1) as one value", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1))
			transact(h, 1, history.OK, readOf("x", int64(1)))
			assert.Nil(t, isolationOf(history.Serializable, h), "the read observed the append")
		})

		t.Run("reads the list of a read from a slice of any element type", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1))
			h.Invoke(1, "txn", []any{[]any{"read", "x", nil}}, "x").OK([]any{[]any{"read", "x", []int{1, 1}}})
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[anomalyField], any(history.DuplicateAppend), "the read returned 1 twice")
		})

		t.Run("reads a read that returned nil as a read of the empty list", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, "txn", []any{[]any{"read", "x", nil}, appendOf("y", 1)}, "x", "y").
				OK([]any{[]any{"read", "x", nil}, appendOf("y", 1)})
			transact(h, 1, history.OK, readOf("y"), appendOf("x", 2))
			got := isolationOf(history.Serializable, h)
			explanation, _ := got[explanationField].([]history.Evidence)
			assert.Length(t, explanation, 2, "an edge for each link of the write skew")
			assert.Equal(t, explanation[0], history.Evidence(history.Edge{
				From: 0, To: 2, Relation: history.RW, Key: "x", Next: 2, Empty: true,
			}), "call 0 read the empty list of x")
		})
	})

	t.Run("HasSnapshotIsolation", func(t *testing.T) {
		t.Parallel()

		t.Run("names its own operation in the fault of a call that is no transaction", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, read, nil).OK(nil)
			expectFault(t, isolationFault(t, history.HasSnapshotIsolation, h), fault.Error{
				Op:     snapshotIsolationOp,
				Path:   fault.Path{fault.Field("calls"), fault.Index(0), fault.Field("operation")},
				Kind:   history.ErrTransaction,
				Reason: `"read" is no transaction`,
			})
		})
	})
}

// transactionAllocs are the cases of the allocation ceiling of a
// transaction's JSON form.
var transactionAllocs = []alloctest.Case{
	{Name: "MarshalJSON", Call: func(assert.TB) { _, _ = committed.MarshalJSON() }, Allocs: transactionJSONAllocs},
}

// TestTransactionAllocs checks the allocation ceiling of a transaction's
// JSON form.
func TestTransactionAllocs(t *testing.T) {
	alloctest.Check(t, transactionAllocs)
}

// BenchmarkTransaction measures the JSON form of a transaction under its
// ceiling.
func BenchmarkTransaction(b *testing.B) {
	for _, c := range transactionAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
