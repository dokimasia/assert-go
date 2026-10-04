// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
)

// optionAllocs are the allocations of an option that its caller keeps,
// measured: the closure of its setting.
const optionAllocs = 1

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

// TestOptionAllocs checks the allocation ceilings of the options.
func TestOptionAllocs(t *testing.T) {
	var kept history.Option
	assert.MaxAllocs(t, func() { kept = history.Budget(10) }, optionAllocs, "Budget allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.MemoLimit(10) }, optionAllocs, "MemoLimit allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.TimeLimit(time.Second) }, optionAllocs,
		"TimeLimit allocates its setting")
	assert.MaxAllocs(t, func() { kept = history.Workers(2) }, optionAllocs, "Workers allocates its setting")
	assert.NotEqual(t, kept, history.Option{}, "the kept option states a setting")
}

// BenchmarkOption measures each option that a caller keeps.
func BenchmarkOption(b *testing.B) {
	tests := []struct {
		name   string
		option func() history.Option
	}{
		{name: "Budget", option: func() history.Option { return history.Budget(10) }},
		{name: "MemoLimit", option: func() history.Option { return history.MemoLimit(10) }},
		{name: "TimeLimit", option: func() history.Option { return history.TimeLimit(time.Second) }},
		{name: "Workers", option: func() history.Option { return history.Workers(2) }},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var got history.Option
			c := bench.Start(b).MaxAllocs(optionAllocs)
			defer c.End()
			for c.Loop() {
				got = tt.option()
			}
			assert.NotEqual(b, got, history.Option{}, "the option states a setting")
		})
	}
}
