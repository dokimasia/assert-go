// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"encoding/json"
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
)

// contract is the contract of every check that the tests run.
const contract = "the register is linearizable"

// The values past the members of each enumeration.
const (
	invalidKind    history.Kind    = 5
	invalidVerdict history.Verdict = 3
	invalidLimit   history.Limit   = 4
)

// The operations of the models of the tests.
const (
	write = "write"
	read  = "read"
)

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

// register is the model of one value, initially 0. write(v) leaves v, and
// read() outputs the value and leaves it. A read whose outcome is unknown
// accepts any value.
var register = history.Model[int]{
	Init: func() int { return 0 },
	Step: func(s int, op history.Op) []int {
		if op.Operation == write {
			return []int{op.Args[0].(int)}
		}
		if !op.Known || op.Output == s {
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
				data, err := history.Span{Completion: -1, Op: history.Op{Operation: write, Args: tt.give}}.MarshalJSON()
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
func detailOf[S any](h *history.History, m history.Model[S], opts ...history.Option) map[string]any {
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
func faultOf[S any](tb testing.TB, h *history.History, m history.Model[S], opts ...history.Option) *fault.Error {
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

// spreading returns the model whose write leaves states, from any state,
// and whose read every state rejects. Its initial state is init.
func spreading[S any](init S, states ...S) history.Model[S] {
	return history.Model[S]{
		Init: func() S { return init },
		Step: func(_ S, op history.Op) []S {
			if op.Operation == write {
				return states
			}
			return nil
		},
	}
}
