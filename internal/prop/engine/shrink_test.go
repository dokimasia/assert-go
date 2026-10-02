// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
)

// referenceSeed is the seed of every run whose outcome the tests pin from
// the definition's executable reference.
const referenceSeed = 7

// shrinkAllocs are the allocations of a run from a stored failing case of
// 2000 that shrinks to 1001 and explains it, measured: 26 cases of about 27
// allocations each, and the record of the stored case's run.
const shrinkAllocs = 704

// property is a body that draws its values and returns the assertion of
// the failure they make, or "" for none.
type property func(c *engine.Case) string

// reference is what the definition's executable reference reports for a
// run: the explanation of each draw of the minimal case, the minimal case's
// token, the runs that shrinking and explaining spent, and the number of
// calls of the body with a SHA-256 digest of the token of each call's
// choices, joined by newlines.
type reference struct {
	// explanation is the explanation of each draw of the minimal case.
	explanation []engine.Explained
	// token is the minimal case's replay token.
	token string
	// runs are the runs that shrinking and explaining spent.
	runs int
	// calls is the number of calls of the body.
	calls int
	// digest is the digest of every call's choices.
	digest string
}

// other is a further failure of a run: its assertion, the values of its
// minimal case, and that case's token.
type other struct {
	// assertion is the failure's assertion.
	assertion string
	// values are the values the minimal case drew.
	values []any
	// token is the minimal case's token.
	token string
}

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
					runs:   34,
					calls:  37,
					digest: "cf9fcfa6a070fb9fb4f00a10ed4aa2cbc04874b2dc26c11ef24a7aa963756e1b",
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
					runs:        29,
					calls:       32,
					digest:      "f8a9d1d6714bcd48a5d322cb0aaf6a6e67f4d183cd1b7177576e1a93b1b7f5af",
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
				"the stored case, which no explanation run reached")
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

// TestShrinkZeroAlloc checks the allocation ceiling of a run that shrinks
// and explains a failing case.
func TestShrinkZeroAlloc(t *testing.T) {
	wide := engine.Integer(0, 1_000_000_000)
	s := settled(integers(2000)...)
	body := func(c *engine.Case) {
		if engine.Draw(c, wide, "n") > 1000 {
			c.Report(assert.Failure{Assertion: "above"}, false)
		}
	}
	assert.MaxAllocs(t, func() { engine.Run(body, s) }, shrinkAllocs, "a run that shrinks 2000 to 1001")
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

// others returns the further failures of a run, in the order it found
// them.
func others(r engine.Result) []other {
	out := make([]other, len(r.Others))
	for i, e := range r.Others {
		out[i] = other{e.Identity.Assertion, drawValues(e.Case.Draws()), token.Encode(e.Case.Choices())}
	}
	return out
}

// settled returns the settings of a run of the reference seed that shrinks
// and explains with the default budget, and tries the stored case of
// choices first when there are any.
func settled(choices ...choice.Choice) engine.Settings {
	s := engine.Settings{
		Seed:       referenceSeed,
		Cases:      engine.DefaultCases,
		MaxChoices: engine.MaxChoices,
		Shrink:     engine.DefaultShrink,
		Explain:    true,
	}
	if len(choices) > 0 {
		s.Stored = [][]choice.Choice{choices}
	}
	return s
}

// traced runs p under s, and returns the result and the token of the
// choices of each call of the body, in call order. The entry of a call is
// taken as the call ends, so a call that a rejection or a repeat ends has
// one too.
func traced(p property, s engine.Settings) (engine.Result, []string) {
	var trace []string
	result := engine.Run(func(c *engine.Case) {
		defer func() { trace = append(trace, token.Encode(c.Choices())) }()
		if failed := p(c); failed != "" {
			c.Report(assert.Failure{Assertion: failed}, false)
		}
	}, s)
	return result, trace
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

// matchesReference checks a counterexample and the trace of its run
// against what the reference reports for the same run.
func matchesReference(tb testing.TB, got engine.Result, trace []string, want reference) {
	tb.Helper()
	assert.Equal(tb, got.Outcome, engine.Counterexample, "a counterexample")
	assert.Equal(tb, got.Explanation, want.explanation, "each draw of the minimal case, explained", assert.EquateNaNs())
	assert.Equal(tb, got.Token, want.token, "the minimal case's token")
	assert.Equal(tb, got.Runs, want.runs, "the runs that shrinking and explaining spent")
	assert.Equal(tb, len(trace), want.calls, "the calls of the body")
	assert.Equal(tb, digestOf(trace), want.digest, "the choices of every call, in order")
}

// digestOf returns the SHA-256 digest of the trace joined by newlines, in
// hexadecimal.
func digestOf(trace []string) string {
	sum := sha256.Sum256([]byte(strings.Join(trace, "\n")))
	return hex.EncodeToString(sum[:])
}

// drawValues returns the values of draws, in order.
func drawValues(draws []engine.Drawn) []any {
	out := make([]any, len(draws))
	for i, d := range draws {
		out[i] = d.Value
	}
	return out
}

// failsWhen returns assertion when failing is true, and "" otherwise.
func failsWhen(failing bool, assertion string) string {
	if failing {
		return assertion
	}
	return ""
}

// above returns the property that fails with "above" for a value of g
// above 1000.
func above(g engine.Generator[int]) property {
	return func(c *engine.Case) string {
		return failsWhen(engine.Draw(c, g, "n") > 1000, "above")
	}
}

// sum returns the sum of values.
func sum(values []int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
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
