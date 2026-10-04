// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"runtime"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
)

// The time limit of the tests of the clock, and the operations of their
// model.
const (
	// timeLimit is the time limit of a check whose model waits past it.
	timeLimit = 50 * time.Millisecond
	// wait is the operation whose step waits past the time limit.
	wait = "wait"
	// spread is the operation whose step waits past the time limit and then
	// leaves spreadStates states.
	spread = "spread"
	// spreadStates is the number of states that spread leaves, more than the
	// steps between two readings of the clock.
	spreadStates = 2000
)

// boom is the value that the panicking functions of the tests raise.
const boom = "boom"

// TestSearch checks the search of a partition through the record of a
// check: its frontier, its limits, its clock, and the faults of a model.
func TestSearch(t *testing.T) {
	t.Parallel()

	t.Run("Linearizable", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the order of calls, the states and the candidates of the frontier of a violated search",
			func(t *testing.T) {
				t.Parallel()
				got := detailOf(violatedRead(), register)
				written := history.Span{
					Call:       0,
					Completion: 1,
					Op:         history.Op{Operation: write, Args: writeOne, Known: true},
				}
				readZero := history.Span{
					Call: 2, Completion: 3, Process: 1, Op: history.Op{Operation: read, Known: true, Output: 0},
				}
				assert.Equal(t, []any{got[linearizedField], got[statesField], got[candidatesField]},
					[]any{[]history.Span{written}, []int{1}, []history.Span{readZero}},
					"the write, the state 1 that it leaves, and the read of 0 that 1 rejects")
			})

		t.Run("reports the candidates that the model rejected before a limit stopped the search", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			five := h.Invoke(0, read, nil, "x")
			one := h.Invoke(1, write, writeOne, "x")
			five.OK(5)
			one.OK(nil)
			got := detailOf(h, register, history.Budget(1))
			readFive := history.Span{Call: 0, Completion: 2, Op: history.Op{Operation: read, Known: true, Output: 5}}
			assert.Equal(t, []any{got[linearizedField], got[statesField], got[candidatesField], got[stepsField]},
				[]any{[]history.Span{}, []int{0}, []history.Span{readFive}, 1},
				"the initial state rejects the read, and the write would pass the budget")
		})

		t.Run("keeps apart two configurations of one set of calls whose states differ", func(t *testing.T) {
			t.Parallel()
			m := history.Model[int]{Init: register.Init, Step: register.Step, Equal: sameInt}
			got := detailOf(parityWrites(), m)
			assert.Equal(t, got[stepsField], any(6), "the state 1 after 3 then 1 is new beside the state 3")
		})

		t.Run(
			"finds a configuration of a partition of more than 64 calls by every word of its set",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				for v := range 64 {
					recordOK(h, 0, write, []any{v + 1}, nil, "x")
				}
				one := h.Invoke(1, write, []any{7}, "x")
				two := h.Invoke(2, write, []any{7}, "x")
				one.OK(nil)
				two.OK(nil)
				recordOK(h, 0, read, nil, 9, "x")
				got := detailOf(h, register)
				assert.Equal(t, []any{got[callsField], got[stepsField], got[statesField]}, []any{67, 69, []int{7}},
					"the memo finds the two writes of 7 in their second order, as the reference does")
			},
		)

		t.Run("keeps apart two configurations of one set of calls whose lists of states differ in length",
			func(t *testing.T) {
				t.Parallel()
				m := history.Model[int]{Init: register.Init, Step: growing, Equal: sameInt}
				got := detailOf(parityWrites(), m)
				assert.Equal(
					t,
					got[stepsField],
					any(7),
					"the states 1 after 3 then 1 are new beside the states 3 and 1",
				)
			})

		t.Run("ends a partition's search before its first step once the time limit has passed", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, wait, nil, nil, "a")
			recordOK(h, 0, read, nil, 0, "b")
			got := detailOf(h, waiting, history.TimeLimit(timeLimit))
			assert.Equal(t,
				[]any{got[outcomeField], got[limitField], got[partitionField], got[stepsField], got[linearizedField]},
				[]any{history.Undecided, history.LimitTime, []any{"b"}, 1, []history.Span{}},
				"the partition of b takes no step after the wait of the partition of a")
		})

		t.Run("reads the clock every 1,024 steps after the first", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, spread, nil, nil, "a")
			recordOK(h, 0, read, nil, 0, "a")
			got := detailOf(h, waiting, history.TimeLimit(timeLimit))
			assert.Equal(t, []any{got[outcomeField], got[limitField], got[stepsField]},
				[]any{history.Undecided, history.LimitTime, 1024}, "the read's steps from the spread states")
			assert.Length(t, got[statesField], spreadStates, "the frontier lists every state of the spread")
		})

		atRead := fault.Path{fault.Field(callsField), fault.Index(2)}
		atWrite := fault.Path{fault.Field(callsField), fault.Index(0)}
		tests := []struct {
			name  string
			model history.Model[int]
			opts  []history.Option
			want  fault.Error
		}{
			{
				name:  "returns the fault of an Init that panics",
				model: history.Model[int]{Init: func() int { panic(boom) }, Step: register.Step},
				want:  modelFault(nil, "the model's Init panics with boom"),
			},
			{
				name:  "returns the fault of a Step that panics, at the call that it steps",
				model: history.Model[int]{Init: register.Init, Step: panicsOnRead},
				want:  modelFault(atRead, `the model's Step panics on "read" with boom`),
			},
			{
				name:  "returns the fault of an Equal that panics, at the call whose states it compares",
				model: history.Model[int]{Init: register.Init, Step: spreading(0, 1, 3).Step, Equal: panicsOnEqual},
				want:  modelFault(atWrite, `the model's Equal panics on "write" with boom`),
			},
			{
				name: "returns the fault of a Hash that panics, at the call whose state it hashes",
				model: history.Model[int]{
					Init: register.Init, Step: register.Step, Hash: func(int) uint64 { panic(boom) },
				},
				want: modelFault(atWrite, `the model's Hash panics on "write" with boom`),
			},
			{
				name:  "returns the fault of a Step that ends its goroutine, at the call that it steps",
				model: history.Model[int]{Init: register.Init, Step: endsOnRead},
				opts:  []history.Option{history.Workers(2)},
				want:  modelFault(atRead, `the model's Step ends its goroutine on "read"`),
			},
			{
				name:  "returns the fault of an Init that ends its goroutine",
				model: history.Model[int]{Init: func() int { runtime.Goexit(); return 0 }, Step: register.Step},
				opts:  []history.Option{history.Workers(2)},
				want:  modelFault(nil, "the model's Init ends its goroutine"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, faultOf(t, violatedRead(), tt.model, tt.opts...), tt.want)
			})
		}
	})
}

// waiting is the model of one value, initially 0, whose wait and spread wait
// past the time limit on a timer of the platform clock, which the limit
// reads. A wait leaves the state, a spread leaves the states 0 to 1,999, and
// every state rejects a read.
var waiting = history.Model[int]{
	Init: func() int { return 0 },
	Step: func(s int, op history.Op) []int {
		switch op.Operation {
		case wait:
			<-time.After(2 * timeLimit)
			return []int{s}
		case spread:
			<-time.After(2 * timeLimit)
			states := make([]int, spreadStates)
			for i := range states {
				states[i] = i
			}
			return states
		}
		return nil
	},
}

// sameInt reports whether a equals b, as a stated Equal beside which the
// search hashes no state.
func sameInt(a, b int) bool {
	return a == b
}

// growing steps the register, and keeps the old value beside a greater
// value that a write leaves over a value other than the initial one.
func growing(s int, op history.Op) []int {
	if op.Operation != write {
		return register.Step(s, op)
	}
	v := op.Args[0].(int)
	if s != 0 && v > s {
		return []int{v, s}
	}
	return []int{v}
}

// modelFault returns the fault of the kind ErrModel of a check at path for
// reason.
func modelFault(path fault.Path, reason string) fault.Error {
	return fault.Error{Op: linearizableOp, Path: path, Kind: history.ErrModel, Reason: reason}
}

// panicsOnRead steps the register, and panics on a read.
func panicsOnRead(s int, op history.Op) []int {
	if op.Operation == read {
		panic(boom)
	}
	return register.Step(s, op)
}

// endsOnRead steps the register, and ends its goroutine on a read.
func endsOnRead(s int, op history.Op) []int {
	if op.Operation == read {
		runtime.Goexit()
	}
	return register.Step(s, op)
}

// panicsOnEqual panics, as an Equal that cannot compare two states does.
func panicsOnEqual(int, int) bool {
	panic(boom)
}
