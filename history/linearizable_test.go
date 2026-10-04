// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"encoding/json"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
)

// linearizableAllocs are the allocations of a passing check of the register
// over a write and a read, measured.
const linearizableAllocs = 38

// The size of the search that the checker cancels.
const (
	// cancelledWrites is the number of concurrent writes of the partition
	// whose search the checker cancels, whose whole search passes its budget.
	cancelledWrites = 20
	// cancelledBudget is the budget of the check whose search the checker
	// cancels.
	cancelledBudget = 2_000_000
	// cancelledSteps is more steps than a cancelled search spends.
	cancelledSteps = 1_000_000
)

// budgetWrites is the number of concurrent writes of distinct values of the
// history that spends the whole default budget.
const budgetWrites = 18

// The size of the history of the queue that spends a budget.
const (
	// queueEnqueues is the number of its concurrent enqueues of distinct
	// values.
	queueEnqueues = 10
	// queueBudget is the budget that it spends.
	queueBudget = 1_000_000
)

// The operations of the partitions of the tests of the workers, and of the
// queue.
const (
	first   = "first"
	second  = "second"
	put     = "put"
	enqueue = "enqueue"
	dequeue = "dequeue"
)

// TestLinearizable checks the check of a history: its verdict, its record,
// its faults, and its workers.
func TestLinearizable(t *testing.T) {
	t.Parallel()

	t.Run("Linearizable", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a linearizable history with a call record of a pass", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			history.Linearizable(rec, passingRead, register, contract)
			assert.False(t, rec.Failed(), "the read follows the write")
			var record struct {
				Assertion string `json:"assertion"`
				Verdict   string `json:"verdict"`
			}
			assert.NoError(t, json.Unmarshal([]byte(rec.Records()[0]), &record), "the call record is a JSON object")
			assert.Equal(t, []string{record.Assertion, record.Verdict}, []string{"linearizable", "pass"},
				"the call record of the pass")
		})

		t.Run("fails a violated history with one record of linearizable at the call", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			_, file, line, _ := runtime.Caller(0)
			history.Linearizable(rec, violatedRead(), register, contract)
			failures := rec.Failures()
			assert.Length(t, failures, 1, "one record")
			got := failures[0]
			assert.Equal(t, []any{got.Assertion, got.Contract, got.Where},
				[]any{"linearizable", contract, assert.Where{File: file, Line: line + 1}},
				"the assertion, the contract and the line of the call")
		})

		t.Run("reports the first violated partition after an undecided one, with the steps of both",
			func(t *testing.T) {
				t.Parallel()
				got := detailOf(threePartitions(), register, history.Budget(3))
				assert.Equal(
					t,
					[]any{got[outcomeField], got[partitionField], got[stepsField], got[partitionsField]},
					[]any{
						history.Violated,
						[]any{"b"},
						5,
						3,
					},
					"the partition of b, after 3 steps in the partition of a",
				)
			})

		t.Run("reports the first undecided partition when no partition is violated", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, "a")
			recordOK(h, 0, read, nil, 1, "a")
			one := h.Invoke(1, write, writeOne, "b")
			two := h.Invoke(2, write, []any{2}, "b")
			one.OK(nil)
			two.OK(nil)
			recordOK(h, 1, read, nil, 3, "b")
			recordOK(h, 3, read, nil, 0, "c")
			got := detailOf(h, register, history.Budget(2))
			assert.Equal(t, []any{got[outcomeField], got[partitionField], got[stepsField], got[partitionsField]},
				[]any{history.Undecided, []any{"b"}, 4, 3}, "the partition of b, with the steps of a and of b")
		})

		t.Run("sends the sentence of the record through Fatalf to a seat without Report", func(t *testing.T) {
			t.Parallel()
			seat := &fatalSeat{}
			history.Linearizable(seat, violatedRead(), register, contract)
			rec := assert.NewRecorder()
			history.Linearizable(rec, violatedRead(), register, contract)
			assert.Equal(t, seat.texts, []string{matcher.Render(rec.Failures()[0])}, "the sentence of the record")
		})

		tests := []struct {
			name    string
			history *history.History
			model   history.Model[int]
			want    string
		}{
			{name: "returns a fault for a nil history", model: register, want: "the history is nil"},
			{
				name:    "returns a fault for a model without Init",
				history: violatedRead(),
				model:   history.Model[int]{Step: register.Step},
				want:    "the model states no Init or no Step",
			},
			{
				name:    "returns a fault for a model without Step",
				history: violatedRead(),
				model:   history.Model[int]{Init: register.Init},
				want:    "the model states no Init or no Step",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, faultOf(t, tt.history, tt.model), fault.Error{Op: linearizableOp, Reason: tt.want})
			})
		}
	})

	t.Run("Workers", func(t *testing.T) {
		t.Parallel()

		t.Run("reports what one worker reports when a later partition ends first", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, first, nil, nil, "a")
			recordOK(h, 1, second, nil, nil, "b")
			got := detailOf(h, waitsForSecond(make(chan struct{})), history.Workers(2))
			assert.Equal(t, []any{got[outcomeField], got[partitionField], got[stepsField], got[partitionsField]},
				[]any{history.Violated, []any{"a"}, 1, 2}, "the partition of a, whose search ended last")
		})

		t.Run("reports what one worker reports for more partitions than workers", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, detailOf(threePartitions(), register, history.Budget(3), history.Workers(2)),
				detailOf(threePartitions(), register, history.Budget(3)), "the record of one worker")
		})

		t.Run("ignores the fault of a partition that one worker would not search", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, first, nil, nil, "a")
			recordOK(h, 1, second, nil, nil, "b")
			got := detailOf(h, panicsOnSecond(nil), history.Workers(2))
			assert.Equal(t, []any{got[outcomeField], got[partitionField]}, []any{history.Violated, []any{"a"}},
				"the violated partition of a, before the partition of b")
		})

		t.Run("returns the fault of a partition that one worker would search", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, first, nil, nil, "a")
			recordOK(h, 1, second, nil, nil, "b")
			got := faultOf(t, h, panicsOnSecond([]int{0}), history.Workers(2))
			expectFault(t, got, modelFault(fault.Path{fault.Field(callsField), fault.Index(2)},
				`the model's Step panics on "second" with boom`))
		})

		t.Run("cancels a search that it no longer needs", func(t *testing.T) {
			t.Parallel()
			started := make(chan struct{})
			var once sync.Once
			var puts atomic.Int64
			m := history.Model[int]{
				Init: register.Init,
				Step: func(s int, op history.Op) []int {
					if op.Operation == put {
						once.Do(func() { close(started) })
						puts.Add(1)
						return []int{op.Args[0].(int)}
					}
					<-started
					return register.Step(s, op)
				},
			}
			got := detailOf(cancelledHistory(), m, history.Budget(cancelledBudget), history.Workers(2))
			assert.Equal(t, []any{got[outcomeField], got[partitionField]}, []any{history.Violated, []any{"a"}},
				"the violated partition of a, whose search waits for the first step of b")
			assert.InRange(t, puts.Load(), 1, cancelledSteps, "the search of b stopped long before its budget")
		})
	})
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

// passingRead is a history in which client 0 writes 1 and client 1 then
// reads 1.
var passingRead = func() *history.History {
	h := history.New()
	recordOK(h, 0, write, writeOne, nil, "x")
	recordOK(h, 1, read, nil, 1, "x")
	return h
}()

// threePartitions returns a history of three partitions. Two writes of a
// run at once, and a read of 3 follows them. A write of b and then a read of
// 2 follow, and then a read of 0 on c.
func threePartitions() *history.History {
	h := history.New()
	one := h.Invoke(0, write, writeOne, "a")
	two := h.Invoke(1, write, []any{2}, "a")
	one.OK(nil)
	two.OK(nil)
	recordOK(h, 2, read, nil, 3, "a")
	recordOK(h, 0, write, writeOne, nil, "b")
	recordOK(h, 0, read, nil, 2, "b")
	recordOK(h, 0, read, nil, 0, "c")
	return h
}

// waitsForSecond returns the model whose step of first waits until the step
// of second has closed secondEnded, and which rejects both.
func waitsForSecond(secondEnded chan struct{}) history.Model[int] {
	return history.Model[int]{
		Init: register.Init,
		Step: func(_ int, op history.Op) []int {
			if op.Operation == second {
				close(secondEnded)
				return nil
			}
			<-secondEnded
			return nil
		},
	}
}

// panicsOnSecond returns the model whose step of first leaves accepted,
// which rejects the call for none, and whose step of second panics.
func panicsOnSecond(accepted []int) history.Model[int] {
	return history.Model[int]{
		Init: register.Init,
		Step: func(_ int, op history.Op) []int {
			if op.Operation == second {
				panic(boom)
			}
			return accepted
		},
	}
}

// cancelledHistory returns a history of two partitions: a write of 1 and a
// read of 0 on a, and on b cancelledWrites puts of distinct values at once
// with a read of a value that no put wrote.
func cancelledHistory() *history.History {
	h := history.New()
	recordOK(h, 0, write, writeOne, nil, "a")
	recordOK(h, 0, read, nil, 0, "a")
	var calls []history.Call
	for client := range cancelledWrites {
		calls = append(calls, h.Invoke(client+1, put, []any{client + 100}, "b"))
	}
	calls = append(calls, h.Invoke(cancelledWrites+1, read, nil, "b"))
	for _, c := range calls[:cancelledWrites] {
		c.OK(nil)
	}
	calls[cancelledWrites].OK(99)
	return h
}

// budgetHistory returns a history of budgetWrites concurrent writes of
// distinct values and a read of a value that none wrote, whose search spends
// the whole default budget.
func budgetHistory() *history.History {
	h := history.New()
	var calls []history.Call
	for client := range budgetWrites {
		calls = append(calls, h.Invoke(client, write, []any{client + 100}, "x"))
	}
	for _, c := range calls {
		c.OK(nil)
	}
	recordOK(h, budgetWrites, read, nil, 99, "x")
	return h
}

// queueHistory returns a history of queueEnqueues concurrent enqueues of
// distinct values, and a dequeue of a value that none enqueued, whose search
// spends a budget of queueBudget steps.
func queueHistory() *history.History {
	h := history.New()
	var calls []history.Call
	for client := range queueEnqueues {
		calls = append(calls, h.Invoke(client, enqueue, []any{client + 100}, "q"))
	}
	for _, c := range calls {
		c.OK(nil)
	}
	recordOK(h, queueEnqueues, dequeue, nil, 99, "q")
	return h
}

// queue is the model of a queue of ints, initially empty, head first, whose
// states the defaults compare through reflection. enqueue(v) adds v at the
// tail, and dequeue() removes the head and outputs it.
var queue = history.Model[[]int]{
	Init: func() []int { return []int{} },
	Step: func(s []int, op history.Op) [][]int {
		if op.Operation == enqueue {
			return [][]int{append(slices.Clip(s), op.Args[0].(int))}
		}
		if len(s) > 0 && (!op.Known || op.Output == s[0]) {
			return [][]int{s[1:]}
		}
		return nil
	},
}

// passingCheck is the case of the allocation ceiling of a passing check.
var passingCheck = alloctest.Case{
	Name:   "Linearizable",
	Call:   func(tb assert.TB) { history.Linearizable(tb, passingRead, register, contract) },
	Allocs: linearizableAllocs,
}

// TestLinearizableAllocs checks the allocation ceiling of a passing check.
func TestLinearizableAllocs(t *testing.T) {
	alloctest.Check(t, []alloctest.Case{passingCheck})
}

// BenchmarkLinearizable measures a passing check under its ceiling, and the
// time of two checks that spend a budget: the register over budgetHistory and
// the default budget, and the queue over queueHistory. The allocations of a
// check that fails count the setup of the module's first failure in a run of
// one iteration, so those two state no ceiling.
func BenchmarkLinearizable(b *testing.B) {
	b.Run("Linearizable", func(b *testing.B) {
		b.Run("pass", func(b *testing.B) { alloctest.Measure(b, passingCheck) })
		b.Run("register", func(b *testing.B) {
			spend(b, budgetHistory(), register)
		})
		b.Run("queue", func(b *testing.B) {
			spend(b, queueHistory(), queue, history.Budget(queueBudget))
		})
	})
}

// spend measures the check of h against m under opts, which spends its
// budget.
func spend[S any](b *testing.B, h *history.History, m history.Model[S], opts ...history.Option) {
	b.Helper()
	rec := assert.NewRecorder()
	c := bench.Start(b)
	defer c.End()
	for c.Loop() {
		history.Linearizable(rec, h, m, contract, opts...)
	}
	assert.Equal(b, rec.Failures()[0].Detail[limitField], any(history.LimitSteps), "the check spends its budget")
}
