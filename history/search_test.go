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
	"go.dokimi.dev/assert/prop"
)

// The time limit of the tests of the clock, and the operations of their
// spec.
const (
	// timeLimit is the time limit of a check whose spec waits past it.
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

// TestSearch checks the search of a partition through the record of a
// check: its frontier, its limits, its clock, and the faults of a spec.
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
					Operation:  history.Operation{Name: write, Args: writeOne, Known: true},
				}
				readZero := history.Span{
					Call:       2,
					Completion: 3,
					Process:    1,
					Operation:  history.Operation{Name: read, Known: true, Output: 0},
				}
				assert.Equal(t, []any{got[linearizedField], got[statesField], got[candidatesField]},
					[]any{[]history.Span{written}, []int{1}, []history.Span{readZero}},
					"the write, the state 1 that it leaves, and the read of 0 that 1 rejects")
			})

		t.Run("reports the candidates that the spec rejected before a limit stopped the search", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			five := h.Invoke(0, read, nil, "x")
			one := h.Invoke(1, write, writeOne, "x")
			five.OK(5)
			one.OK(nil)
			got := detailOf(h, register, history.Budget(1))
			readFive := history.Span{
				Call:       0,
				Completion: 2,
				Operation:  history.Operation{Name: read, Known: true, Output: 5},
			}
			assert.Equal(t, []any{got[linearizedField], got[statesField], got[candidatesField], got[stepsField]},
				[]any{[]history.Span{}, []int{0}, []history.Span{readFive}, 1},
				"the initial state rejects the read, and the write would pass the budget")
		})

		t.Run("keeps apart two configurations of one set of calls whose states differ", func(t *testing.T) {
			t.Parallel()
			m := history.Spec[int]{Initial: register.Initial, Next: register.Next, Equal: sameInt}
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
				m := history.Spec[int]{Initial: register.Initial, Next: growing, Equal: sameInt}
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

		t.Run("reports at each check of a growing history what a search of every call reports", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, "a grown search reports what a search of every call reports", func(c *prop.Case) {
				m := register
				if c.Draw(prop.Boolean(), "lossy") {
					m = lossy
				}
				h := history.New()
				var cp history.Checkpoint[int]
				open := map[int]history.Call{}
				for range c.Draw(prop.Integer(1, 30), "events") {
					client := c.Draw(prop.Integer(0, 2), "client")
					call, busy := open[client]
					switch {
					case busy:
						delete(open, client)
						complete(c, call)
					case c.Draw(prop.Boolean(), "writes"):
						open[client] = h.Invoke(client, write, []any{c.Draw(prop.Integer(1, 2), "value")})
					default:
						open[client] = h.Invoke(client, read, nil)
					}
					whole, wholeFinal := outcomeOf(h, m)
					grown, grownFinal := outcomeOf(h, m, history.Resume(&cp))
					assert.Equal(c, grown, whole, "the grown search reports the record of the search of every call")
					assert.Equal(c, grownFinal, wholeFinal, "the grown search leaves the same states")
				}
			}, prop.Seed(1), prop.Cases(300))
		})

		t.Run("continues a grown search into the calls before the growth, as a search of every call does",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				one := h.Invoke(0, write, writeOne)
				two := h.Invoke(1, write, []any{2})
				one.OK(nil)
				two.OK(nil)
				var cp history.Checkpoint[int]
				_, final := outcomeOf(h, register, history.Resume(&cp))
				assert.Equal(t, final, []int{2}, "the first order writes 1 and then 2")
				recordOK(h, 0, read, nil, 1)
				_, final = outcomeOf(h, register, history.Resume(&cp))
				assert.Equal(t, final, []int{1}, "the read of 1 takes the order that writes 2 and then 1")
				recordOK(h, 0, read, nil, 3)
				got, _ := outcomeOf(h, register, history.Resume(&cp))
				want, _ := outcomeOf(h, register)
				assert.Equal(t, got, want, "the violation of the read of 3, with the steps of the search of every call")
			})

		t.Run("finds a configuration that a grown search stored before its set of calls gained a word",
			func(t *testing.T) {
				t.Parallel()
				h := writes(62)
				one := h.Invoke(1, write, []any{7})
				two := h.Invoke(2, write, []any{7})
				one.OK(nil)
				two.OK(nil)
				var cp history.Checkpoint[int]
				assert.Nil(t, resumedDetail(h, register, &cp), "64 writes pass")
				recordOK(h, 0, read, nil, 9)
				got := resumedDetail(h, register, &cp)
				assert.Equal(t, got, detailOf(h, register, history.Whole()),
					"the memo finds the two writes of 7 in their second order, as the search of every call does")
				assert.Equal(t, []any{got[callsField], got[stepsField]}, []any{65, 67},
					"64 steps of the writes, the read's step, and the two writes of 7 in their second order")
			})

		atRead := fault.Path{fault.Field(callsField), fault.Index(2)}
		atWrite := fault.Path{fault.Field(callsField), fault.Index(0)}
		tests := []struct {
			name string
			spec history.Spec[int]
			opts []history.Option
			want fault.Error
		}{
			{
				name: "returns the fault of an Initial that panics",
				spec: history.Spec[int]{Initial: func() int { panic(boom) }, Next: register.Next},
				want: specFault(nil, "the spec's Initial panics with boom"),
			},
			{
				name: "returns the fault of a Next that panics, at the call that it steps",
				spec: history.Spec[int]{Initial: register.Initial, Next: panicsOnRead},
				want: specFault(atRead, `the spec's Next panics on "read" with boom`),
			},
			{
				name: "returns the fault of an Equal that panics, at the call whose states it compares",
				spec: history.Spec[int]{Initial: register.Initial, Next: spreading(0, 1, 3).Next, Equal: panicsOnEqual},
				want: specFault(atWrite, `the spec's Equal panics on "write" with boom`),
			},
			{
				name: "returns the fault of a Hash that panics, at the call whose state it hashes",
				spec: history.Spec[int]{
					Initial: register.Initial, Next: register.Next, Hash: func(int) uint64 { panic(boom) },
				},
				want: specFault(atWrite, `the spec's Hash panics on "write" with boom`),
			},
			{
				name: "returns the fault of a Next that ends its goroutine, at the call that it steps",
				spec: history.Spec[int]{Initial: register.Initial, Next: endsOnRead},
				opts: []history.Option{history.Workers(2)},
				want: specFault(atRead, `the spec's Next ends its goroutine on "read"`),
			},
			{
				name: "returns the fault of an Initial that ends its goroutine",
				spec: history.Spec[int]{Initial: func() int { runtime.Goexit(); return 0 }, Next: register.Next},
				opts: []history.Option{history.Workers(2)},
				want: specFault(nil, "the spec's Initial ends its goroutine"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, faultOf(t, violatedRead(), tt.spec, tt.opts...), tt.want)
			})
		}
	})
}

// waiting is the spec of one value, initially 0, whose wait and spread wait
// past the time limit on a timer of the platform clock, which the limit
// reads. A wait leaves the state, a spread leaves the states 0 to 1,999, and
// every state rejects a read.
var waiting = history.Spec[int]{
	Initial: func() int { return 0 },
	Next: func(s int, op history.Operation) []int {
		switch op.Name {
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

// lossy is the register whose write may be lost: it leaves the written value
// and the value before it. A read is the register's read.
var lossy = history.Spec[int]{
	Initial: register.Initial,
	Next: func(s int, op history.Operation) []int {
		if op.Name == write {
			return []int{op.Args[0].(int), s}
		}
		return register.Next(s, op)
	},
}

// complete completes call by the case's choices: as Fail or Unknown, or as
// OK with an output from 0 to 2, which a read reports and a write ignores.
func complete(c *prop.Case, call history.Call) {
	switch c.Draw(prop.Integer(0, 9), "outcome") {
	case 0:
		call.Fail(errRefused)
	case 1:
		call.Unknown(errRefused)
	default:
		call.OK(c.Draw(prop.Integer(0, 2), "output"))
	}
}

// outcomeOf checks h against m under Whole, Final and opts on a recorder, and
// returns the detail of each record of the check and the states that it
// stored.
func outcomeOf(h *history.History, m history.Spec[int], opts ...history.Option) ([]map[string]any, []int) {
	var final []int
	rec := assert.NewRecorder()
	all := append([]history.Option{history.Whole(), history.Final(&final)}, opts...)
	history.Linearizable(rec, h, m, contract, all...)
	var details []map[string]any
	for _, f := range rec.Failures() {
		details = append(details, f.Detail)
	}
	return details, final
}

// sameInt reports whether a equals b, as a stated Equal beside which the
// search hashes no state.
func sameInt(a, b int) bool {
	return a == b
}

// growing steps the register, and keeps the old value beside a greater
// value that a write leaves over a value other than the initial one.
func growing(s int, op history.Operation) []int {
	if op.Name != write {
		return register.Next(s, op)
	}
	v := op.Args[0].(int)
	if s != 0 && v > s {
		return []int{v, s}
	}
	return []int{v}
}

// specFault returns the fault of the kind ErrSpec of a check at path for
// reason.
func specFault(path fault.Path, reason string) fault.Error {
	return fault.Error{Op: linearizableOp, Path: path, Kind: history.ErrSpec, Reason: reason}
}

// panicsOnRead steps the register, and panics on a read.
func panicsOnRead(s int, op history.Operation) []int {
	if op.Name == read {
		panic(boom)
	}
	return register.Next(s, op)
}

// endsOnRead steps the register, and ends its goroutine on a read.
func endsOnRead(s int, op history.Operation) []int {
	if op.Name == read {
		runtime.Goexit()
	}
	return register.Next(s, op)
}

// panicsOnEqual panics, as an Equal that cannot compare two states does.
func panicsOnEqual(int, int) bool {
	panic(boom)
}
