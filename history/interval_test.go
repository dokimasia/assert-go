// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
)

// fromIntervalsAllocs are the allocations of FromIntervals for two entries
// of two clients on one string key, measured.
const fromIntervalsAllocs = 42

// fromIntervalsOp is the operation of the faults of FromIntervals.
const fromIntervalsOp = "history.FromIntervals"

// TestInterval checks the history that FromIntervals builds from calls
// recorded with a start and an end on one clock.
func TestInterval(t *testing.T) {
	t.Parallel()

	t.Run("FromIntervals", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the events in time order, an invocation before a completion at one time", func(t *testing.T) {
			t.Parallel()
			h, err := history.FromIntervals([]history.Interval{
				{Client: 0, Operation: write, Args: writeOne, Keys: []any{"x"}, Start: 10, End: 20, Kind: history.OK},
				{Client: 1, Operation: read, Keys: []any{"x"}, Start: 20, End: 30, Kind: history.OK, Output: 1},
			})
			assert.NoError(t, err, "the entries are a history")
			want := []history.Event{
				{Index: 0, Kind: history.Invoke, Call: 0, Operation: write, Args: writeOne, Keys: []any{"x"}},
				{Index: 1, Kind: history.Invoke, Call: 1, Client: 1, Process: 1, Operation: read, Keys: []any{"x"}},
				{Index: 2, Kind: history.OK, Call: 0},
				{Index: 3, Kind: history.OK, Call: 1, Client: 1, Process: 1, Output: 1},
			}
			assert.Equal(t, h.Events(), want, "the read starts before the write ends")
		})

		t.Run("keeps the order of the entries among the invocations and among the completions at one time",
			func(t *testing.T) {
				t.Parallel()
				h, err := history.FromIntervals([]history.Interval{
					{Client: 1, Operation: read, Start: 5, End: 9, Kind: history.OK},
					{Client: 0, Operation: write, Args: writeOne, Start: 5, End: 9, Kind: history.OK},
				})
				assert.NoError(t, err, "the entries are a history")
				assert.Equal(t, clientsOf(h), []int{1, 0, 1, 0}, "client 1, as the first entry, first in each phase")
			})

		t.Run("records an entry of the kind Invoke as pending without reading its end", func(t *testing.T) {
			t.Parallel()
			h, err := history.FromIntervals([]history.Interval{
				{Client: 0, Operation: write, Args: writeOne, Start: 5, End: 1, Kind: history.Invoke},
			})
			assert.NoError(t, err, "a pending entry states no end")
			assert.Equal(t, h.Events(), []history.Event{
				{Kind: history.Invoke, Operation: write, Args: writeOne},
			}, "an invocation without a completion")
		})

		t.Run("records the output of an ok entry and the error of a fail or an unknown entry", func(t *testing.T) {
			t.Parallel()
			h, err := history.FromIntervals([]history.Interval{
				{Client: 0, Operation: read, Start: 0, End: 1, Kind: history.OK, Output: 7},
				{Client: 0, Operation: write, Args: writeOne, Start: 2, End: 3, Kind: history.Fail, Error: errRefused},
				{
					Client:    0,
					Operation: write,
					Args:      writeOne,
					Start:     4,
					End:       5,
					Kind:      history.Unknown,
					Error:     errRefused,
				},
				{Client: 0, Operation: read, Start: 6, End: 7, Kind: history.OK, Output: 1},
			})
			assert.NoError(t, err, "the entries are a history")
			events := h.Events()
			assert.Equal(t, []any{events[1].Output, events[3].Error, events[5].Error}, []any{7, errRefused, errRefused},
				"the output and the errors")
			assert.Equal(t, events[6].Process, 1, "the client continues on a new process after the unknown entry")
		})

		t.Run("compares times at the ends of the range of int64 without subtracting them", func(t *testing.T) {
			t.Parallel()
			h, err := history.FromIntervals([]history.Interval{
				{
					Client:    0,
					Operation: write,
					Args:      writeOne,
					Start:     math.MaxInt64 - 1,
					End:       math.MaxInt64,
					Kind:      history.OK,
				},
				{Client: 0, Operation: read, Start: math.MinInt64, End: math.MinInt64 + 1, Kind: history.OK},
			})
			assert.NoError(t, err, "the two entries do not overlap")
			assert.Equal(t, callsOf(h), []int{0, 0, 2, 2}, "the read, at the earliest times, first")
		})

		t.Run("accepts overlapping entries of two clients", func(t *testing.T) {
			t.Parallel()
			_, err := history.FromIntervals([]history.Interval{
				{Client: 0, Operation: write, Args: writeOne, Start: 0, End: 10, Kind: history.OK},
				{Client: 1, Operation: read, Start: 0, End: 10, Kind: history.OK},
			})
			assert.NoError(t, err, "each client has one call")
		})

		t.Run("accepts an entry of a client that ends before its pending entry starts", func(t *testing.T) {
			t.Parallel()
			h, err := history.FromIntervals([]history.Interval{
				{Client: 0, Operation: write, Args: writeOne, Start: 5, Kind: history.Invoke},
				{Client: 0, Operation: read, Start: 0, End: 3, Kind: history.OK},
			})
			assert.NoError(t, err, "a pending entry overlaps only the later entries of its client")
			assert.Equal(t, callsOf(h), []int{0, 0, 2}, "the read and then the pending write")
		})

		tests := []struct {
			name string
			give []history.Interval
			want fault.Error
		}{
			{
				name: "returns a fault at an entry that states no kind",
				give: []history.Interval{
					{Client: 0, Operation: read, Start: 0, End: 1, Kind: history.OK},
					{Client: 1, Operation: read, Start: 0, End: 1},
				},
				want: refused(1, "states no kind of event"),
			},
			{
				name: "returns a fault at an entry that ends before it starts",
				give: []history.Interval{{Client: 0, Operation: read, Start: 9, End: 5, Kind: history.OK}},
				want: refused(0, "ends at 5, before it starts at 9"),
			},
			{
				name: "returns a fault at an entry that states a key without a typed literal",
				give: []history.Interval{{Client: 0, Operation: read, Keys: []any{make(chan int)}, Kind: history.OK}},
				want: refused(0, "states the key chan int, which no typed literal states"),
			},
			{
				name: "returns a fault at an entry that shares an instant with an earlier entry of its client",
				give: []history.Interval{
					{Client: 0, Operation: write, Args: writeOne, Start: 0, End: 10, Kind: history.OK},
					{Client: 1, Operation: read, Start: 10, End: 20, Kind: history.OK},
					{Client: 0, Operation: read, Start: 10, End: 20, Kind: history.OK},
				},
				want: refused(2, "overlaps entry 0 of client 0"),
			},
			{
				name: "returns a fault at an entry that ends where an earlier entry of its client starts",
				give: []history.Interval{
					{Client: 0, Operation: write, Args: writeOne, Start: 10, End: 20, Kind: history.OK},
					{Client: 0, Operation: read, Start: 0, End: 10, Kind: history.OK},
				},
				want: refused(1, "overlaps entry 0 of client 0"),
			},
			{
				name: "returns a fault at an entry that starts after a pending entry of its client",
				give: []history.Interval{
					{Client: 0, Operation: write, Args: writeOne, Start: 5, Kind: history.Invoke},
					{Client: 0, Operation: read, Start: 100, End: 200, Kind: history.OK},
				},
				want: refused(1, "overlaps entry 0 of client 0"),
			},
			{
				name: "returns a fault at a pending entry that starts before an earlier entry of its client ends",
				give: []history.Interval{
					{Client: 0, Operation: read, Start: 0, End: 10, Kind: history.OK},
					{Client: 0, Operation: write, Args: writeOne, Start: 10, Kind: history.Invoke},
				},
				want: refused(1, "overlaps entry 0 of client 0"),
			},
			{
				name: "returns a fault at the first entry in the given order that breaks a rule",
				give: []history.Interval{
					{Client: 0, Operation: read, Start: 0, End: 1, Kind: history.OK},
					{Client: 1, Operation: read, Start: 9, End: 5, Kind: history.OK},
					{Client: 2, Operation: read, Start: 0, End: 1},
				},
				want: refused(1, "ends at 5, before it starts at 9"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h, err := history.FromIntervals(tt.give)
				assert.Nil(t, h, "no history")
				f := assert.ErrorAs[*fault.Error](t, err, "a fault")
				assert.Equal(t, fault.Error{Op: f.Op, Path: f.Path, Kind: f.Kind, Reason: f.Reason}, tt.want,
					"the operation, the entry, the kind and the reason of the fault")
			})
		}
	})
}

// refused returns the fault of FromIntervals at the entry i for reason.
func refused(i int, reason string) fault.Error {
	return fault.Error{Op: fromIntervalsOp, Path: fault.Path{fault.Index(i)}, Kind: history.ErrInterval, Reason: reason}
}

// clientsOf returns the client of each event of h, in recording order.
func clientsOf(h *history.History) []int {
	var out []int
	for _, e := range h.Events() {
		out = append(out, e.Client)
	}
	return out
}

// callsOf returns the call of each event of h, in recording order.
func callsOf(h *history.History) []int {
	var out []int
	for _, e := range h.Events() {
		out = append(out, e.Call)
	}
	return out
}

// twoEntries are the entries of the measurement of FromIntervals.
var twoEntries = []history.Interval{
	{Client: 0, Operation: write, Args: writeOne, Keys: []any{"x"}, Start: 10, End: 20, Kind: history.OK},
	{Client: 1, Operation: read, Keys: []any{"x"}, Start: 20, End: 30, Kind: history.OK, Output: 1},
}

// intervalAllocs are the cases of the allocation ceiling of FromIntervals.
var intervalAllocs = []alloctest.Case{
	{
		Name:   "FromIntervals",
		Call:   func(assert.TB) { _, _ = history.FromIntervals(twoEntries) },
		Allocs: fromIntervalsAllocs,
	},
}

// TestIntervalAllocs checks the allocation ceiling of FromIntervals.
func TestIntervalAllocs(t *testing.T) {
	alloctest.Check(t, intervalAllocs)
}

// BenchmarkInterval measures FromIntervals under its ceiling.
func BenchmarkInterval(b *testing.B) {
	for _, c := range intervalAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
