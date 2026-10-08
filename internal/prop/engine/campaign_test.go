// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"strconv"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/coverage"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
)

// The sizes of the campaigns of the tests.
const (
	// searchCases are the cases of a campaign that searches for a list that
	// starts with wanted.
	searchCases = 8000
	// shortCases are the cases of a campaign whose outcome needs few.
	shortCases = 30
	// shapeCases are the cases of a campaign over cases of four kinds.
	shapeCases = 400
	// campaignAllocs are the allocations of a campaign of shortCases cases
	// that draw one digit each, its clock included, measured: about seven
	// for each case.
	campaignAllocs = 220
)

// epoch is the time on the budget's clock when a campaign of the tests
// starts.
var epoch = time.Date(2026, time.October, 5, 9, 0, 0, 0, time.UTC)

// wanted is the start of the lists that the search of the tests fails on.
// A random case of the seed starts with it once in about 200,000.
var wanted = []int{7, 3, 9, 1, 5}

// TestCampaign checks a campaign: the cases that it runs, the pool that
// guides it, the failures that it concludes and the outcome that it
// reports.
func TestCampaign(t *testing.T) {
	t.Parallel()

	digits := engine.List(engine.Integer(0, 9), sizes(t, 0, 8))
	digit := engine.Integer(0, 9)

	t.Run("Campaign", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the random cases of the seed in order, past Cases, while the pool is empty", func(t *testing.T) {
			t.Parallel()
			var got []string
			s := settled()
			s.Cases = 10
			listed := func(c *engine.Case) { engine.Draw(c, digits, drawn) }
			r := campaigned(func(c *engine.Case) {
				listed(c)
				got = append(got, token.Encode(c.Choices()))
			}, s, shortCases)
			want := make([]string, shortCases)
			for i := range want {
				e := engine.Generate(listed, referenceSeed, uint64(i), nil)
				want[i] = token.Encode(e.Case.Choices())
			}
			assert.Equal(t, got, want, "random cases 0 to 29 of the seed")
			assert.Equal(t, summary(r), engine.Result{Outcome: engine.Passed, Cases: shortCases, Seed: referenceSeed},
				"a pass that counts every case")
		})

		guides := []struct {
			name  string
			guide func(c *engine.Case, matched int)
		}{
			{
				name:  "finds a failure that needs the pool when its cases count a label of their progress",
				guide: func(c *engine.Case, matched int) { c.Classify(strconv.Itoa(matched)) },
			},
			{
				name:  "finds a failure that needs the pool when its cases observe their progress",
				guide: func(c *engine.Case, matched int) { c.Observe(uint64(matched)) },
			},
			{
				name:  "finds a failure that needs the pool when its cases score their progress",
				guide: func(c *engine.Case, matched int) { c.Target("matched", float64(matched)) },
			},
		}
		for _, tt := range guides {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r := campaigned(searching(digits, tt.guide), settled(), searchCases)
				assert.Equal(t, r.Outcome, engine.Counterexample, "a list that starts with 7, 3, 9, 1 and 5")
			})
		}

		t.Run("finds no such failure in as many cases without a guide", func(t *testing.T) {
			t.Parallel()
			r := campaigned(searching(digits, func(*engine.Case, int) {}), settled(), searchCases)
			assert.Equal(t, r.Outcome, engine.Passed, "no list that starts with 7, 3, 9, 1 and 5")
		})

		t.Run("mutates three cases in four of pool members with and without spans", func(t *testing.T) {
			t.Parallel()
			shaped := func(c *engine.Case) {
				kind := c.Integer(choice.MustIntegerBounds(choice.Int{}, choice.UintOf(3))).Magnitude()
				c.Classify(strconv.FormatUint(kind, 10))
				if kind == 0 {
					engine.Draw(c, digit, drawn)
				}
				if kind == 1 {
					engine.Draw(c, digits, drawn)
				}
				if kind == 2 {
					c.Rand().Uint64()
				}
			}
			var tokens []string
			r := campaigned(func(c *engine.Case) {
				shaped(c)
				tokens = append(tokens, token.Encode(c.Choices()))
			}, settled(), shapeCases)
			random, mutated := 0, 0
			for _, tok := range tokens {
				e := engine.Generate(shaped, referenceSeed, uint64(random), nil)
				if tok == token.Encode(e.Case.Choices()) {
					random++
					continue
				}
				mutated++
			}
			assert.Equal(t, summary(r), engine.Result{Outcome: engine.Passed, Cases: shapeCases, Seed: referenceSeed},
				"a pass that counts every case")
			assert.InRange(t, float64(mutated)/shapeCases, 0.6, 0.8,
				"about three cases in four mutated, between the random cases of the seed in order")
		})

		t.Run("draws every request of a mutation of a member without choices from its own stream", func(t *testing.T) {
			t.Parallel()
			calls := 0
			r := campaigned(func(c *engine.Case) {
				calls++
				if calls == 1 {
					c.Classify("first")
					return
				}
				engine.Draw(c, digit, drawn)
			}, settled(), shortCases)
			assert.Equal(t, summary(r), engine.Result{Outcome: engine.Passed, Cases: shortCases, Seed: referenceSeed},
				"a pass that counts every case")
		})

		t.Run("concludes each failure of a new identity, and reports later ones among the others", func(t *testing.T) {
			t.Parallel()
			var concluded []string
			s := settled()
			s.Concluded = func(r engine.Result) { concluded = append(concluded, r.Failing.Identity.Assertion) }
			r := campaigned(func(c *engine.Case) {
				v := engine.Draw(c, engine.Integer(0, 1000), drawn)
				if v%2 == 1 {
					c.Report(assert.Failure{Assertion: "odd"}, false)
				}
				if v >= 900 {
					c.Report(assert.Failure{Assertion: "big"}, false)
				}
			}, s, shapeCases)
			others := make([]string, len(r.Others))
			for i, o := range r.Others {
				others[i] = o.Identity.Assertion
			}
			assert.Equal(t, []any{r.Outcome, r.Failing.Identity.Assertion, others},
				[]any{engine.Counterexample, "odd", []string{"big"}}, "the first failure, and the later one")
			assert.Equal(t, concluded, []string{"odd", "big"}, "each identity concluded once, as found")
			assert.Equal(t, drawValues(r.Failing.Case.Draws()), []any{1}, "the first failure, shrunk")
		})

		t.Run("reports the failures that the shrink of a failure finds among its others", func(t *testing.T) {
			t.Parallel()
			concluded := 0
			s := settled()
			s.Examples = []engine.Example{{Choices: integers(950)}}
			s.Concluded = func(engine.Result) { concluded++ }
			r := campaigned(func(c *engine.Case) {
				v := engine.Draw(c, engine.Integer(0, 1000), drawn)
				if v >= 900 {
					c.Report(assert.Failure{Assertion: "huge"}, false)
				}
				if v >= 100 {
					c.Report(assert.Failure{Assertion: "big"}, false)
				}
			}, s, shortCases)
			assert.Equal(t, []any{r.Failing.Identity.Assertion, len(r.Others), r.Others[0].Identity.Assertion},
				[]any{"huge", 1, "big"}, "the example's failure, and the one that its shrink found")
			assert.Equal(t, concluded, 1, "no conclusion of a later failure of big")
		})

		t.Run("concludes a failing example of values as found, and goes on", func(t *testing.T) {
			t.Parallel()
			var concluded []engine.Result
			s := settled()
			s.Examples = []engine.Example{{Values: []any{42}}}
			s.Concluded = func(r engine.Result) { concluded = append(concluded, r) }
			r := campaigned(func(c *engine.Case) {
				if engine.Draw(c, digit, drawn) >= 10 {
					c.Report(assert.Failure{Assertion: "beyond"}, false)
				}
			}, s, shortCases)
			assert.Length(t, concluded, 1, "the example's failure, concluded once")
			assert.Equal(t, []any{concluded[0].Token, concluded[0].Runs}, []any{"", 0}, "no token, and no shrink")
			assert.Equal(t, []any{r.Outcome, r.Cases}, []any{engine.Counterexample, shortCases - 1},
				"the example's failure, after the campaign's other cases")
			assert.True(t, r.Failing.Case.Valued(), "the example of values")
		})

		flaky := []struct {
			name   string
			adjust func(s *engine.Settings)
		}{
			{name: "ends as flaky at a random case whose replay passes", adjust: func(*engine.Settings) {}},
			{
				name:   "ends as flaky at an example whose replay passes",
				adjust: func(s *engine.Settings) { s.Examples = []engine.Example{{Choices: integers(3)}} },
			},
			{
				name:   "ends as flaky at a stored case whose replay passes",
				adjust: func(s *engine.Settings) { s.Stored = [][]choice.Choice{integers(3)} },
			},
		}
		for _, tt := range flaky {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				failed := false
				s := settled()
				tt.adjust(&s)
				r := campaigned(func(c *engine.Case) {
					engine.Draw(c, digit, drawn)
					if !failed {
						failed = true
						c.Report(assert.Failure{Assertion: "once"}, false)
					}
				}, s, shortCases)
				assert.Equal(t, []any{r.Outcome, r.Divergence.What, r.Cases},
					[]any{engine.Flaky, engine.VerdictDifference, 0}, "a replay that passes, of the first case")
			})
		}

		outcomes := []struct {
			name string
			body engine.Body
			give []engine.Requirement
			want engine.Result
		}{
			{
				name: "reports a campaign that rejects more than ten cases for every valid one as rejected",
				body: func(c *engine.Case) {
					engine.Draw(c, digit, drawn)
					c.Assume(false)
				},
				want: engine.Result{Outcome: engine.Rejected, Rejected: shortCases, Seed: referenceSeed},
			},
			{
				name: "reports a campaign whose cases request no input as vacuous",
				body: func(*engine.Case) {},
				want: engine.Result{Outcome: engine.Vacuous, Cases: shortCases, Seed: referenceSeed},
			},
			{
				name: "reports a requirement that its valid cases leave unmet",
				body: func(c *engine.Case) { engine.Draw(c, digit, drawn) },
				give: []engine.Requirement{{Label: "never", Share: 0.5}},
				want: engine.Result{
					Outcome: engine.CoverageUnmet, Cases: shortCases, Seed: referenceSeed,
					Shortfall: &engine.Shortfall{
						Requirement: engine.Requirement{Label: "never", Share: 0.5}, Valid: shortCases,
						Verdict: coverage.Unmet,
					},
				},
			},
		}
		for _, tt := range outcomes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Requirements = tt.give
				assert.Equal(t, summary(campaigned(tt.body, s, shortCases)), tt.want, "the outcome and its counts")
			})
		}

		t.Run("runs the case of the entries, the examples and the stored cases first", func(t *testing.T) {
			t.Parallel()
			var got []any
			s := settled(integers(6)...)
			s.Draws = []engine.Entry{{Label: drawn, Value: 4}}
			s.Examples = []engine.Example{{Choices: integers(5)}}
			r := campaigned(func(c *engine.Case) { got = append(got, engine.Draw(c, digit, drawn)) }, s, shortCases)
			assert.Equal(t, got[:3], []any{4, 5, 6}, "the entry's value, the example's, then the stored case's")
			assert.Length(t, r.Stored, 1, "the run of the stored case")
		})

		t.Run("ends with the refusal of an entry of Draws, before any other case", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "other", Value: 4}}
			r := campaigned(func(c *engine.Case) { engine.Draw(c, digit, drawn) }, s, shortCases)
			assert.Equal(t, r, engine.Result{Refused: r.Refused}, "the refusal, and nothing else")
			assert.NotNil(t, r.Refused, "a refusal")
		})
	})
}

// campaigned runs a campaign of body under s for cases seconds of a
// controlled clock, which each case advances by one second before its body
// runs, so the campaign runs cases cases.
func campaigned(body engine.Body, s engine.Settings, cases int) engine.Result {
	clock := assert.NewControlled(epoch)
	s.Budget, s.BudgetClock = time.Duration(cases)*time.Second, clock
	return engine.Campaign(func(c *engine.Case) {
		clock.Advance(time.Second)
		body(c)
	}, s)
}

// searching returns a body that draws a list of digits, guides the
// campaign with the length of the start of the list that matches wanted,
// and fails once the list starts with wanted.
func searching(digits engine.Generator[[]int], guide func(c *engine.Case, matched int)) engine.Body {
	return func(c *engine.Case) {
		list := engine.Draw(c, digits, drawn)
		matched := 0
		for matched < min(len(list), len(wanted)) && list[matched] == wanted[matched] {
			matched++
		}
		guide(c, matched)
		if matched == len(wanted) {
			c.Report(assert.Failure{Assertion: "found"}, false)
		}
	}
}

// oneDigit is the body of the measured campaign.
var oneDigit = func(c *engine.Case) { c.Integer(digitRange) }

// TestCampaignAllocs checks the ceiling of a campaign.
func TestCampaignAllocs(t *testing.T) {
	s := settled()
	assert.MaxAllocs(t, func() { campaigned(oneDigit, s, shortCases) }, campaignAllocs,
		"a campaign allocates its cases")
}

// BenchmarkCampaign measures a campaign of shortCases cases that draw one
// digit each.
func BenchmarkCampaign(b *testing.B) {
	b.Run("Campaign", func(b *testing.B) {
		var got engine.Result
		s := settled()
		c := bench.Start(b).MaxAllocs(campaignAllocs)
		defer c.End()
		for c.Loop() {
			got = campaigned(oneDigit, s, shortCases)
		}
		assert.Equal(b, got.Cases, shortCases, "every case of the campaign")
	})
}
