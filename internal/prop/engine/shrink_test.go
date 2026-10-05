// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"slices"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
)

// shrinkAllocs are the allocations of a run from a stored failing case of
// 2000 that shrinks to 1001 and explains it, measured: 26 cases of about 18
// allocations each, the record of the stored case's run, 5 for the
// delete-and-lower round that ends the shrink, whose one integer is at
// index 0, and the list of where the replay that confirms the failure made
// its request. The shrink's cases reuse the storage of spare cases, so each
// allocates its recorder, its goroutine and its candidate's choices, nodes
// and token, and not the growth of its record.
const shrinkAllocs = 479

// spentAllocs are the allocations of a run from a stored failing case of
// 512 choices whose budget of one run is spent by its first candidate,
// measured: the run of the stored case and the run of the candidate, each
// of 512 draws, and the list of where the replay that confirms the failure
// made its requests. No pass that starts after the budget is spent builds a
// candidate.
const spentAllocs = 4308

// TestShrink checks shrinking through whole runs: the minimal case of each
// failure, the shared budget of runs and time, and the order of every
// candidate, pinned by the digest of every call of the body.
func TestShrink(t *testing.T) {
	t.Parallel()

	percent := engine.List(engine.Integer(0, 100), unbounded(t, 0))
	digits := engine.List(engine.Integer(0, 9), unbounded(t, 0))
	small, wide := engine.Integer(0, 1000), engine.Integer(0, 1_000_000_000)
	start := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			p    property
			want reference
		}{
			{
				name: "shrinks a list whose sum is above 10 to the one element 11",
				p: func(c *engine.Case) string {
					return failsWhen(sum(engine.Draw(c, percent, "xs")) > 10, "sum")
				},
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{11}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEACwAA",
					runs:        21,
					calls:       24,
					digest:      "a8ff1f71c1ca1867d10d48156b1e46501eb5efb8b84651e2d2cef825c7c36592",
				},
			},
			{
				name: "shrinks a list of three or more digits to three zeros",
				p: func(c *engine.Case) string {
					return failsWhen(len(engine.Draw(c, digits, "xs")) >= 3, "length")
				},
				want: reference{
					explanation: []engine.Explained{
						{Label: "xs", Value: []int{0, 0, 0}, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEAAAABAAAAAQAAAAA",
					runs:   35,
					calls:  38,
					digest: "934d85d9eac9fb943b4dd3d5c5cea8f8bcfedb7ab4d899dc8c41dcc19fc25873",
				},
			},
			{
				name: "shrinks an unsorted list of digits to 1 then 0",
				p: func(c *engine.Case) string {
					return failsWhen(!sorted(engine.Draw(c, digits, "xs")), "unsorted")
				},
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{1, 0}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEAAQABAAAAAA",
					runs:        31,
					calls:       34,
					digest:      "042701cea9d5c90e84fba919d277a1b0749b4de9396fc867b5c6d93429addafc",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, trace := traced(tt.p, settled())
				matchesReference(t, got, trace, tt.want)
			})
		}

		t.Run("shrinks a second failure and reports it under the others", func(t *testing.T) {
			t.Parallel()
			got, trace := traced(func(c *engine.Case) string {
				v := engine.Draw(c, small, "x")
				if v%2 == 1 {
					return "odd"
				}
				return failsWhen(v > 50, "big")
			}, settled())
			matchesReference(t, got, trace, reference{
				explanation: []engine.Explained{
					{Label: "x", Value: 1, Relevance: engine.ValueMatters, NearestPassing: 0},
				},
				token:  "prop1:AAE",
				runs:   66,
				calls:  69,
				digest: "91b51ff8621ebe89ea94488277489b5bc754df7a4fd1a3a5b5de4d03e93390c2",
			})
			assert.Equal(t, others(got), []other{{"big", []any{52}, "prop1:ADQ"}}, "the smallest even value above 50")
		})

		t.Run("shrinks the failures found while shrinking, the smallest first", func(t *testing.T) {
			t.Parallel()
			got, trace := traced(func(c *engine.Case) string {
				v := engine.Draw(c, small, "x")
				if v > 600 {
					return "big"
				}
				if v > 300 {
					return "mid"
				}
				return failsWhen(v > 100, "low")
			}, settled(integers(900)...))
			matchesReference(t, got, trace, reference{
				explanation: []engine.Explained{{Label: "x", Value: 601, Relevance: engine.ValueMatters}},
				token:       "prop1:ANkE",
				runs:        47,
				calls:       49,
				digest:      "f8ef4dbfa536d4116b676958ce9221099dec7aca241786dea24bac97fc36c992",
			})
			assert.Equal(t, others(got), []other{{"mid", []any{301}, "prop1:AK0C"}, {"low", []any{101}, "prop1:AGU"}},
				"the other failures, in the order they were found")
		})

		t.Run("stops at the smallest case found when the budget is spent", func(t *testing.T) {
			t.Parallel()
			s := settled(integers(2000)...)
			s.Shrink = 10
			got, trace := traced(above(wide), s)
			matchesReference(t, got, trace, reference{
				explanation: []engine.Explained{{Label: "n", Value: 1007, Relevance: engine.Untested}},
				token:       "prop1:AO8H",
				runs:        10,
				calls:       12,
				digest:      "ff30a3b1c8ac3259a117f8f1e8e290d364e616d62a3a25595442790ff83b4f4f",
			})
		})

		t.Run("stops at the smallest case found when the time is spent", func(t *testing.T) {
			t.Parallel()
			clock := assert.NewControlled(start)
			s := settled(integers(2000)...)
			s.ShrinkClock, s.ShrinkTime = clock, time.Second
			runs := 0
			got := engine.Run(func(c *engine.Case) {
				if engine.Draw(c, wide, "n") > 1000 {
					c.Report(assert.Failure{Assertion: "above"}, false)
				}
				runs++
				if runs == 4 {
					clock.Advance(time.Second + time.Nanosecond)
				}
			}, s)
			assert.Equal(t, got.Runs, 2, "the empty case and the target, before the clock passed the deadline")
			assert.Equal(t, got.Explanation, []engine.Explained{{Label: "n", Value: 2000, Relevance: engine.Untested}},
				"the stored case, which no explanation run tested")
		})

		t.Run("runs to the budget when the time is not spent", func(t *testing.T) {
			t.Parallel()
			s := settled(integers(2000)...)
			s.ShrinkClock, s.ShrinkTime = assert.NewControlled(start), time.Second
			got, _ := traced(above(wide), s)
			assert.Equal(t, got.Runs, 24, "the runs of the whole shrink and explanation")
		})

		t.Run("measures the time on the shrink clock and not on the test clock", func(t *testing.T) {
			t.Parallel()
			clock := assert.NewControlled(start)
			s := settled(integers(2000)...)
			s.Clock, s.ShrinkClock, s.ShrinkTime = clock, assert.NewControlled(start), time.Second
			got := engine.Run(func(c *engine.Case) {
				if engine.Draw(c, wide, "n") > 1000 {
					c.Report(assert.Failure{Assertion: "above"}, false)
				}
				clock.Advance(time.Hour)
			}, s)
			assert.Equal(t, got.Runs, 24, "the runs of the whole shrink and explanation, as if no time had passed")
		})

		t.Run("measures the time on the platform clock when no shrink clock is stated", func(t *testing.T) {
			t.Parallel()
			s := settled(integers(2000)...)
			s.Clock, s.ShrinkTime = assert.NewControlled(start), time.Hour
			got, _ := traced(above(wide), s)
			assert.Equal(t, got.Runs, 24, "the runs of the whole shrink and explanation, well within an hour")
		})

		t.Run("returns the first failing case when shrinking is off", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Shrink = 0
			got, trace := traced(above(wide), s)
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{558560502}, "random case 0 of seed 7")
			assert.Equal(t, got.Token, "prop1:APbpq4oC", "its token")
			assert.Equal(t, got.Runs, 0, "no run spent")
			assert.Empty(t, got.Explanation, "no explanation")
			assert.Equal(t, digestOf(trace), "095bad80b00f5a47692bf3842457d181c720e4dc0f90d205d959f3b2c9decaed",
				"the simplest case, then random case 0")
		})

		workers := []struct {
			name   string
			p      property
			stored []choice.Choice
			budget int
		}{
			{name: "reports on four workers the bisection of one", p: above(wide), stored: integers(2000)},
			{
				name: "reports on four workers the chunks that one deletes",
				p: func(c *engine.Case) string {
					values := engine.Draw(c, percent, "xs")
					return failsWhen(len(values) > 0 && values[0] > 10, "first")
				},
				stored: integers(1, 50, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 0),
			},
			{
				name: "reports on four workers the siblings that one sorts",
				p: func(c *engine.Case) string {
					values := engine.Draw(c, percent, "xs")
					return failsWhen(slices.Contains(values, 3) && slices.Contains(values, 5), "both")
				},
				stored: integers(1, 5, 1, 3, 0),
			},
			{
				name: "reports on four workers the rounding that one accepts",
				p: func(c *engine.Case) string {
					return failsWhen(engine.Draw(c, engine.Float(0.0, 10.0, choice.ExcludeNaN), "v") > 2.5, "bigger")
				},
				stored: []choice.Choice{float(2.875)},
			},
			{
				name: "reports on four workers the other failures of one",
				p: func(c *engine.Case) string {
					v := engine.Draw(c, small, "x")
					if v > 600 {
						return "big"
					}
					return failsWhen(v > 300, "mid")
				},
				stored: integers(900),
			},
			{
				name: "reports on four workers the explanation of one",
				p: func(c *engine.Case) string {
					value := engine.Draw(c, wide, "n")
					engine.Draw(c, wide, "noise")
					return failsWhen(value > 1000, "above")
				},
			},
			{
				name:   "reports on four workers the budget that one spends",
				p:      above(wide),
				stored: integers(2000),
				budget: 10,
			},
			{
				name: "reports on four workers a budget that runs out among the fillings",
				p: func(c *engine.Case) string {
					value := engine.Draw(c, wide, "n")
					engine.Draw(c, wide, "noise")
					return failsWhen(value > 1000, "above")
				},
				budget: 54,
			},
		}
		for _, tt := range workers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				one, four := settled(tt.stored...), settled(tt.stored...)
				if tt.budget > 0 {
					one.Shrink, four.Shrink = tt.budget, tt.budget
				}
				four.Workers = fourWorkers
				assert.Equal(t, reportOf(engine.Run(failing(tt.p), four)), reportOf(engine.Run(failing(tt.p), one)),
					"what four workers report, against one")
			})
		}

		t.Run("returns a token that replays the minimal case", func(t *testing.T) {
			t.Parallel()
			got, _ := traced(above(wide), settled(integers(2000)...))
			choices, err := token.Decode(got.Token)
			assert.NoError(t, err, "the token decodes")
			var value int
			engine.Replay(func(c *engine.Case) { value = engine.Draw(c, wide, "n") }, choices, nil)
			assert.Equal(t, value, 1001, "the minimal value")
		})
	})
}

// TestShrinkAllocs checks the allocation ceiling of a run that shrinks
// and explains a failing case.
func TestShrinkAllocs(t *testing.T) {
	wide := engine.Integer(0, 1_000_000_000)
	s := settled(integers(2000)...)
	body := func(c *engine.Case) {
		if engine.Draw(c, wide, "n") > 1000 {
			c.Report(assert.Failure{Assertion: "above"}, false)
		}
	}
	assert.MaxAllocs(t, func() { engine.Run(body, s) }, shrinkAllocs, "a run that shrinks 2000 to 1001")

	digit := engine.Integer(0, 9)
	nines := make([]int64, 512)
	for i := range nines {
		nines[i] = 9
	}
	spent := settled(integers(nines...)...)
	spent.Shrink = 1
	sum := func(c *engine.Case) {
		total := 0
		for range nines {
			total += engine.Draw(c, digit, "n")
		}
		if total > 0 {
			c.Report(assert.Failure{Assertion: "positive"}, false)
		}
	}
	assert.MaxAllocs(t, func() { engine.Run(sum, spent) }, spentAllocs,
		"a run of 512 choices whose budget of one run its first candidate spends")
}

// BenchmarkShrink measures a run from a stored failing case of 2000 that
// shrinks to 1001 and explains it.
func BenchmarkShrink(b *testing.B) {
	wide := engine.Integer(0, 1_000_000_000)
	s := settled(integers(2000)...)
	body := func(c *engine.Case) {
		if engine.Draw(c, wide, "n") > 1000 {
			c.Report(assert.Failure{Assertion: "above"}, false)
		}
	}

	b.Run("Run", func(b *testing.B) {
		var got engine.Result
		c := bench.Start(b).MaxAllocs(shrinkAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Run(body, s)
		}
		assert.Equal(b, got.Token, "prop1:AOkH", "the minimal case of 1001")
	})
}

// failing returns the body that reports the failure p returns, which is
// safe to run concurrently with itself.
func failing(p property) engine.Body {
	return func(c *engine.Case) {
		if failed := p(c); failed != "" {
			c.Report(assert.Failure{Assertion: failed}, false)
		}
	}
}

// sorted reports whether values are in ascending order.
func sorted(values []int) bool {
	for i := 1; i < len(values); i++ {
		if values[i] < values[i-1] {
			return false
		}
	}
	return true
}
