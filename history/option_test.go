// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
)

// optionAllocs is the ceiling of the allocations of an option that its
// caller keeps: the closure of its setting.
const optionAllocs = 2

// TestOption checks each option of a check.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("Budget", func(t *testing.T) {
		t.Parallel()

		t.Run("stops a search before the step that would pass the budget", func(t *testing.T) {
			t.Parallel()
			got := detailOf(violatedRead(), register, history.Budget(1))
			assert.Equal(t, []any{got[outcomeField], got[stepsField], got[limitField]},
				[]any{history.Undecided, 1, history.LimitSteps}, "the read would be the second step")
		})

		t.Run("panics for steps below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { history.Budget(0) }, "a budget of no step")
			assert.Equal(t, got, any("history: Budget(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("MemoLimit", func(t *testing.T) {
		t.Parallel()

		t.Run("stops a search before a configuration that would pass the limit", func(t *testing.T) {
			t.Parallel()
			got := detailOf(violatedRead(), register, history.MemoLimit(1))
			assert.Equal(t, []any{got[outcomeField], got[stepsField], got[limitField]},
				[]any{history.Undecided, 1, history.LimitMemo}, "a configuration counts a bit for each of two calls")
		})

		t.Run("stores a configuration of a bit for each call within the limit", func(t *testing.T) {
			t.Parallel()
			got := detailOf(violatedRead(), register, history.MemoLimit(2))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the one configuration fits two bits")
		})

		t.Run("panics for bits below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { history.MemoLimit(0) }, "a memo of no bit")
			assert.Equal(t, got, any("history: MemoLimit(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("TimeLimit", func(t *testing.T) {
		t.Parallel()

		t.Run("sets no limit for a time of 0", func(t *testing.T) {
			t.Parallel()
			got := detailOf(violatedRead(), register, history.TimeLimit(0))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the search ends")
		})

		t.Run("panics for a negative time", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { history.TimeLimit(-time.Second) }, "a negative time")
			assert.Equal(t, got, any("history: TimeLimit(-1s) is below 0"), "the panic names the option")
		})
	})

	t.Run("Workers", func(t *testing.T) {
		t.Parallel()

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { history.Workers(0) }, "no worker")
			assert.Equal(t, got, any("history: Workers(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("Whole", func(t *testing.T) {
		t.Parallel()

		t.Run("searches every call as one partition, whatever keys the calls declare", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, "x")
			recordOK(h, 1, read, nil, 0, "y")
			assert.Nil(t, detailOf(h, register), "x and y pass apart")
			got := detailOf(h, register, history.Whole())
			assert.Equal(t, []any{got[outcomeField], got[partitionsField], got[partitionField]},
				[]any{history.Violated, 1, []any{}},
				"the read of 0 follows the write of 1 in one partition of every key")
		})
	})

	t.Run("Final", func(t *testing.T) {
		t.Parallel()

		t.Run("stores the states that the order of a passing check leaves", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, finalOf(passingRead, register), []int{1}, "the write left 1, and the read kept it")
		})

		t.Run("stores the initial state for a history without calls", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, finalOf(history.New(), register), []int{0}, "the register starts at 0")
		})

		t.Run("stores nil for a check that does not pass", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, finalOf(violatedRead(), register), "no order passes")
		})

		t.Run("stores nil for a check of two partitions", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, "x")
			recordOK(h, 1, read, nil, 0, "y")
			assert.Nil(t, finalOf(h, register), "each partition leaves states of its own")
		})

		t.Run("stores the states of one partition of every key under Whole", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, "x")
			recordOK(h, 1, read, nil, 1, "y")
			assert.Equal(t, finalOf(h, register, history.Whole()), []int{1}, "the write and the read in one order")
		})

		t.Run("returns the fault of an Initial that panics on a history without calls", func(t *testing.T) {
			t.Parallel()
			m := history.Spec[int]{Initial: func() int { panic(boom) }, Next: register.Next}
			got := faultOf(t, history.New(), m, history.Final(new([]int)))
			expectFault(t, got, fault.Error{
				Op: linearizableOp, Kind: history.ErrSpec, Reason: "the spec's Initial panics with boom",
			})
		})
	})

	t.Run("Resume", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the search of a passing check, which the next check of a grown history continues",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := writes(5)
				var cp history.Checkpoint[int]
				assert.Nil(t, resumedDetail(h, m, &cp), "five writes pass")
				recordOK(h, 0, read, nil, 5)
				assert.Nil(t, resumedDetail(h, m, &cp), "the read of the last value passes")
				assert.Equal(t, steps, 6, "five steps of the writes, and the step of the read")
			})

		t.Run("changes nothing for a nil checkpoint", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, detailOf(violatedRead(), register, history.Resume[int](nil)),
				detailOf(violatedRead(), register), "the record of the defaults, with no Whole required")
		})
	})

	t.Run("Option", func(t *testing.T) {
		t.Parallel()

		t.Run("changes nothing as the zero option", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, detailOf(violatedRead(), register, history.Option{}), detailOf(violatedRead(), register),
				"the record of the defaults")
		})

		t.Run("applies a later option over an earlier one", func(t *testing.T) {
			t.Parallel()
			got := detailOf(violatedRead(), register, history.Budget(1), history.Budget(2))
			assert.Equal(t, got[outcomeField], any(history.Violated), "a budget of two steps")
		})
	})
}

// finalOf checks h against m on a recorder under opts and Final, and
// returns the states that the check stored, which start as an empty list.
func finalOf[S any](h *history.History, m history.Spec[S], opts ...history.Option) []S {
	states := []S{}
	history.Linearizable(assert.NewRecorder(), h, m, contract, append(opts, history.Final(&states))...)
	return states
}

// The values that the options of the allocation ceilings of Final and
// Resume keep.
var (
	finalStates []int
	checkpoint  history.Checkpoint[int]
)

// TestOptionAllocs checks the allocation ceilings of the options.
func TestOptionAllocs(t *testing.T) {
	var kept history.Option
	assert.MaxAllocs(t, func() { kept = history.Budget(10) }, optionAllocs, "Budget allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.MemoLimit(10) }, optionAllocs, "MemoLimit allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.TimeLimit(time.Second) }, optionAllocs,
		"TimeLimit allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.Workers(2) }, optionAllocs, "Workers allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.Whole() }, 0, "Whole states a setting that captures nothing")
	assert.MaxAllocs(t, func() { kept = history.Final(&finalStates) }, optionAllocs, "Final allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.Resume(&checkpoint) }, optionAllocs, "Resume allocates its setting")
	assert.NotEqual(t, kept, history.Option{}, "the kept option states a setting")
}

// BenchmarkOption measures each option that a caller keeps.
func BenchmarkOption(b *testing.B) {
	tests := []struct {
		name   string
		option func() history.Option
		allocs uint64
	}{
		{name: "Budget", option: func() history.Option { return history.Budget(10) }, allocs: optionAllocs},
		{name: "MemoLimit", option: func() history.Option { return history.MemoLimit(10) }, allocs: optionAllocs},
		{
			name:   "TimeLimit",
			option: func() history.Option { return history.TimeLimit(time.Second) },
			allocs: optionAllocs,
		},
		{name: "Workers", option: func() history.Option { return history.Workers(2) }, allocs: optionAllocs},
		{name: "Whole", option: history.Whole},
		{name: "Final", option: func() history.Option { return history.Final(&finalStates) }, allocs: optionAllocs},
		{name: "Resume", option: func() history.Option { return history.Resume(&checkpoint) }, allocs: optionAllocs},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var got history.Option
			c := bench.Start(b).MaxAllocs(tt.allocs)
			defer c.End()
			for c.Loop() {
				got = tt.option()
			}
			assert.NotEqual(b, got, history.Option{}, "the option states a setting")
		})
	}
}
