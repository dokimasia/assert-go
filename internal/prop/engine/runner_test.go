// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"slices"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/coverage"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/internal/record"
)

// The allocations of a run, measured.
const (
	// runAllocs are the allocations of a passing run of 100 cases of an
	// integer in [0, 1000]: its 136 cases, its case tree and its executor.
	runAllocs = 485
	// replayAllocs are the allocations of a replay of one case that draws
	// one integer.
	replayAllocs = 9
	// concludeAllocs are the allocations of concluding the bridged case of
	// 10,000 for a body that fails from 1,001: the replay that confirms it,
	// with the list of where it made its request, the runs of its shrink and
	// of its explanation, and the result.
	concludeAllocs = 472
)

// largestBytes are the fuzzer's bytes that the bridge decodes as 10,000
// for an integer in [0, 10000]: two bytes, little-endian.
var largestBytes = []byte{0x10, 0x27}

// TestRunner checks the phases of a run, how a run ends, and the replay of
// one case, through runs pinned to what the definition's executable
// reference reports, with a digest of the choices of every call of the
// body.
func TestRunner(t *testing.T) {
	t.Parallel()

	digit, small := engine.Integer(0, 9), engine.Integer(0, 1000)
	tenThousand, whole := engine.Integer(0, 10_000), engine.Integer[uint64](0, math.MaxUint64)
	choiceOf := func(value int) func(*engine.Case) {
		return func(c *engine.Case) {
			if engine.Draw(c, tenThousand, drawn) >= value {
				c.Report(assert.Failure{Assertion: "big"}, false)
			}
		}
	}
	atLeast900 := func(c *engine.Case) {
		if engine.Draw(c, small, drawn) >= 900 {
			c.Report(assert.Failure{Assertion: "big"}, false)
		}
	}
	largest := func(c *engine.Case) {
		if engine.Draw(c, whole, drawn) == math.MaxUint64 {
			c.Report(assert.Failure{Assertion: "largest"}, false)
		}
	}
	classify := func(label string, holds func(int) bool) engine.Body {
		return func(c *engine.Case) {
			if holds(engine.Draw(c, small, drawn)) {
				c.Classify(label)
			}
		}
	}
	classifyDigit := func(label string, holds func(int) bool) engine.Body {
		return func(c *engine.Case) {
			if holds(engine.Draw(c, digit, drawn)) {
				c.Classify(label)
			}
		}
	}
	even := func(v int) bool { return v%2 == 0 }
	shortfall := func(label string, share float64, counted, valid int, verdict coverage.Verdict) *engine.Shortfall {
		requirement := engine.Requirement{Label: label, Share: share}
		return &engine.Shortfall{Requirement: requirement, Counted: counted, Valid: valid, Verdict: verdict}
	}
	digitBounds := choice.OfInteger(choice.MustIntegerBounds(choice.Int{}, choice.UintOf(9)))
	bitBounds := choice.OfInteger(choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1)))

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for the six outcomes and false past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t,
				[]engine.Outcome{
					engine.Passed, engine.Counterexample, engine.Flaky,
					engine.Rejected, engine.CoverageUnmet, engine.Vacuous,
				},
				[]engine.Outcome{invalidOutcome})
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			body   engine.Body
			adjust func(*engine.Settings)
			want   engine.Result
			calls  int
			digest string
		}{
			{
				name:   "returns Passed for a body that never fails",
				body:   func(c *engine.Case) { engine.Draw(c, small, drawn) },
				want:   engine.Result{Outcome: engine.Passed, Cases: 100, Seed: referenceSeed},
				calls:  136,
				digest: "998a64931654c18b988d88613d4f5868549493a6524e8053adc3a4302f8119da",
			},
			{
				name:   "returns Rejected for a body that rejects every case",
				body:   func(c *engine.Case) { engine.Draw(c, small, drawn); c.Assume(false) },
				want:   engine.Result{Outcome: engine.Rejected, Rejected: 458, Seed: referenceSeed},
				calls:  1005,
				digest: "f33c01aa8d0dd7a5dcd1d7b4efee31a86a00fc84b0fa6bd91b6f524757ba58c0",
			},
			{
				name:   "returns Passed for a body that rejects four cases in five",
				body:   func(c *engine.Case) { c.Assume(engine.Draw(c, small, drawn)%5 == 0) },
				want:   engine.Result{Outcome: engine.Passed, Cases: 98, Rejected: 360, Seed: referenceSeed},
				calls:  1005,
				digest: "f33c01aa8d0dd7a5dcd1d7b4efee31a86a00fc84b0fa6bd91b6f524757ba58c0",
			},
			{
				name: "rejects each case larger than MaxChoices",
				body: func(c *engine.Case) {
					if len(engine.Draw(c, engine.List(small, unbounded(t, 0)), drawn)) >= 2 {
						c.Report(assert.Failure{Assertion: "long"}, false)
					}
				},
				adjust: func(s *engine.Settings) { s.MaxChoices = 4 },
				want:   engine.Result{Outcome: engine.Passed, Cases: 100, Rejected: 604, Seed: referenceSeed},
				calls:  889,
				digest: "a383e2375443feaa985b9104bd9e0aac8b4f094c6b1bf5d57cc019856f8ae352",
			},
			{
				name:   "returns Passed for a run that rejects exactly ten cases for every valid one",
				body:   func(c *engine.Case) { c.Assume(engine.Draw(c, engine.Integer(0, 10), drawn) == 0) },
				adjust: func(s *engine.Settings) { s.Seed, s.Cases = 0, 2 },
				want:   engine.Result{Outcome: engine.Passed, Cases: 1, Rejected: 10},
				calls:  21,
				digest: "cb936f364a0e9426cdbd6f155dba448f0ae0a956b85363ed7b30f4ffd2106a6d",
			},
			{
				name: "runs a prefix case while at most ten of a hundred cases are valid",
				body: func(c *engine.Case) {
					engine.Draw(c, small, drawn)
					engine.Draw(c, small, "second")
				},
				want:   engine.Result{Outcome: engine.Passed, Cases: 100, Seed: referenceSeed},
				calls:  107,
				digest: "9937981906d46f4d4ba29eda420a4d66cd0be982867cb9bacd1d293fd0a3b7a3",
			},
			{
				name:   "returns Vacuous for a body that requests no input",
				body:   func(*engine.Case) {},
				want:   engine.Result{Outcome: engine.Vacuous, Cases: 1, Seed: referenceSeed},
				calls:  1,
				digest: "7a3e8573a5b0f78d78479509cde64ed061c20379be5b7dd3d6cbe996eef329ad",
			},
			{
				name:   "returns Passed for a body that draws a value without a choice",
				body:   func(c *engine.Case) { engine.Draw(c, engine.Just(1), drawn) },
				want:   engine.Result{Outcome: engine.Passed, Cases: 1, Seed: referenceSeed},
				calls:  1,
				digest: "7a3e8573a5b0f78d78479509cde64ed061c20379be5b7dd3d6cbe996eef329ad",
			},
			{
				name: "returns Passed once every input of a small domain is tested",
				body: func(c *engine.Case) {
					letters := anyOf(engine.SampledFrom("a", "b", "c"))
					engine.Draw(c, engine.OneOf(anyOf(engine.Boolean(1, 2)), letters), drawn)
				},
				want:   engine.Result{Outcome: engine.Passed, Cases: 5, Seed: referenceSeed},
				calls:  8,
				digest: "d30e367105e8912a760f18ec96f2d493d566c2d92dfac3c42858edfd236e58e9",
			},
			{
				name:   "counts each input of a domain once",
				body:   func(c *engine.Case) { engine.Draw(c, engine.Integer(0, 30), drawn) },
				want:   engine.Result{Outcome: engine.Passed, Cases: 31, Seed: referenceSeed},
				calls:  165,
				digest: "35b1a6b9211a53c28ea5cfc43397fe907c12c9fa6f6d58a9b2678f5268511ccc",
			},
			{
				name:   "returns Passed once a coverage requirement is met",
				body:   classify("even", even),
				adjust: require(engine.Requirement{Label: "even", Share: 0.3}),
				want:   engine.Result{Outcome: engine.Passed, Cases: 400, Seed: referenceSeed},
				calls:  819,
				digest: "5173db487c28445b7c03124a4301d841b417affe328759a36b920b91ab08e1ba",
			},
			{
				name: "returns Passed once every coverage requirement is met",
				body: func(c *engine.Case) {
					v := engine.Draw(c, small, drawn)
					if even(v) {
						c.Classify("even")
					}
					if v >= 500 {
						c.Classify("high")
					}
				},
				adjust: require(
					engine.Requirement{Label: "even", Share: 0.3},
					engine.Requirement{Label: "high", Share: 0.3},
				),
				want:   engine.Result{Outcome: engine.Passed, Cases: 800, Seed: referenceSeed},
				calls:  3091,
				digest: "946313970ebba67322c87712ce1684f36051c364f5272a452e32b493ef05af88",
			},
			{
				name:   "returns CoverageUnmet for a refuted requirement",
				body:   classify("even", even),
				adjust: require(engine.Requirement{Label: "even", Share: 0.9}),
				want: engine.Result{
					Outcome:   engine.CoverageUnmet,
					Cases:     100,
					Seed:      referenceSeed,
					Shortfall: shortfall("even", 0.9, 45, 100, coverage.Refuted),
				},
				calls:  136,
				digest: "998a64931654c18b988d88613d4f5868549493a6524e8053adc3a4302f8119da",
			},
			{
				name:   "returns CoverageUnmet for a requirement the last check leaves unmet",
				body:   classify("even", even),
				adjust: require(engine.Requirement{Label: "even", Share: 0.6}),
				want: engine.Result{
					Outcome:   engine.CoverageUnmet,
					Cases:     800,
					Seed:      referenceSeed,
					Shortfall: shortfall("even", 0.6, 397, 800, coverage.Unmet),
				},
				calls:  3091,
				digest: "946313970ebba67322c87712ce1684f36051c364f5272a452e32b493ef05af88",
			},
			{
				name:   "returns Passed for a share an exhausted domain meets exactly",
				body:   classifyDigit("small", func(v int) bool { return v < 5 }),
				adjust: require(engine.Requirement{Label: "small", Share: 0.5}),
				want:   engine.Result{Outcome: engine.Passed, Cases: 10, Seed: referenceSeed},
				calls:  31,
				digest: "b2f87c78dfd85a51a51647a66a7e20f99f2c976ee08ee38af85f8c5c224d515e",
			},
			{
				name:   "returns CoverageUnmet for a share an exhausted domain misses",
				body:   classifyDigit("nine", func(v int) bool { return v >= 9 }),
				adjust: require(engine.Requirement{Label: "nine", Share: 0.2}),
				want: engine.Result{
					Outcome:   engine.CoverageUnmet,
					Cases:     10,
					Seed:      referenceSeed,
					Shortfall: shortfall("nine", 0.2, 1, 10, coverage.Unmet),
				},
				calls:  31,
				digest: "b2f87c78dfd85a51a51647a66a7e20f99f2c976ee08ee38af85f8c5c224d515e",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				if tt.adjust != nil {
					tt.adjust(&s)
				}
				got, trace := recorded(tt.body, s)
				assert.Equal(t, summary(got), tt.want, "how the run ended, and its counts")
				assert.Equal(t, len(trace), tt.calls, "the calls of the body")
				assert.Equal(t, digestOf(trace), tt.digest, "the choices of every call, in order")
			})
		}

		t.Run("returns Flaky for a body whose requests diverge", func(t *testing.T) {
			t.Parallel()
			calls := 0
			got, trace := recorded(func(c *engine.Case) {
				calls++
				if calls == 1 {
					engine.Draw(c, digit, drawn)
					return
				}
				engine.Draw(c, engine.Boolean(1, 2), drawn)
			}, settled())
			divergence := &engine.Divergence{
				What: engine.RequestDifference, Recorded: digitBounds, Replayed: bitBounds,
				Where: engine.Where{Label: drawn, Drawing: true},
			}
			want := engine.Result{Outcome: engine.Flaky, Cases: 1, Seed: referenceSeed, Divergence: divergence}
			assert.Equal(t, summary(got), want, "the second case requests other bounds at its first choice")
			assert.Equal(t, digestOf(trace), "706d2435438fccdb677ab7ab2ec05973a15348f7c8e6478a2bbb4d406d53c5ae",
				"both calls")
		})

		counterexamples := []struct {
			name   string
			body   engine.Body
			adjust func(*engine.Settings)
			cases  int
			want   reference
		}{
			{
				name:  "returns the minimal counterexample with its nearest passing value",
				body:  choiceOf(1001),
				cases: 1,
				want: reference{
					explanation: []engine.Explained{
						{Label: drawn, Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
					},
					token:  "prop1:AOkH",
					runs:   27,
					calls:  30,
					digest: "24ca970e8145033ee62e61472d8d738be916eb0376738b1c3436ee87eb5b08f1",
				},
			},
			{
				name:  "returns a failure at the maximum that an edge case finds",
				body:  largest,
				cases: 3,
				want: reference{
					explanation: []engine.Explained{{
						Label:          drawn,
						Value:          uint64(math.MaxUint64),
						Relevance:      engine.ValueMatters,
						NearestPassing: uint64(math.MaxUint64 - 1),
					}},
					token:  "prop1:AP___________wE",
					runs:   68,
					calls:  74,
					digest: "229188e4a238caf16ce2e421dcce75f7820011cbce9c58c73f34e40392fce2aa",
				},
			},
			{
				name:   "returns a failure of a stored case before the simplest case runs",
				body:   atLeast900,
				adjust: func(s *engine.Settings) { s.Stored = [][]choice.Choice{integers(950)} },
				want: reference{
					explanation: []engine.Explained{
						{Label: drawn, Value: 900, Relevance: engine.ValueMatters, NearestPassing: 899},
					},
					token:  "prop1:AIQH",
					runs:   23,
					calls:  25,
					digest: "e11d9e7307c91df74c3ba1af820247f1508e6376b9d11b0fa88e81d8701c1033",
				},
			},
			{
				name:   "returns a failure of a later stored case after an earlier one passes",
				body:   atLeast900,
				adjust: func(s *engine.Settings) { s.Stored = [][]choice.Choice{integers(100), integers(950)} },
				cases:  1,
				want: reference{
					explanation: []engine.Explained{
						{Label: drawn, Value: 900, Relevance: engine.ValueMatters, NearestPassing: 899},
					},
					token:  "prop1:AIQH",
					runs:   23,
					calls:  26,
					digest: "8dc4896d121342d24918ac8c32de04b285521baa189ce245d214c5d413999589",
				},
			},
		}
		for _, tt := range counterexamples {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				if tt.adjust != nil {
					tt.adjust(&s)
				}
				got, trace := recorded(tt.body, s)
				matchesReference(t, got, trace, tt.want)
				assert.Equal(t, got.Cases, tt.cases, "the valid cases before the failure")
			})
		}

		firsts := []struct {
			name   string
			body   engine.Body
			cases  int
			values []any
			token  string
			digest string
		}{
			{
				name:   "returns the first failing case when shrinking is off",
				body:   choiceOf(1001),
				cases:  1,
				values: []any{8522},
				token:  "prop1:AMpC",
				digest: "f734e8abcfbace3a5b8d97fed937007dbcda93202d896a06a1ffcfc1371313e3",
			},
			{
				name: "returns a failure that the prefix case of random case 0 finds",
				body: func(c *engine.Case) {
					values := engine.Draw(c, engine.List(engine.Integer(0, 1_000_000_000), sizes(t, 2, 2)), drawn)
					if values[0] == 558560502 && values[1] == 0 {
						c.Report(assert.Failure{Assertion: "prefix"}, false)
					}
				},
				cases:  2,
				values: []any{[]int{558560502, 0}},
				token:  "prop1:AAEA9umrigIAAQAAAAA",
				digest: "29ea7f6e784080abacd38cfe160d250efcadcece551c283958bf2d7849ea93f4",
			},
		}
		for _, tt := range firsts {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Shrink = 0
				got, trace := recorded(tt.body, s)
				assert.Equal(t, drawValues(got.Failing.Case.Draws()), tt.values, "the first failing case's values")
				assert.Equal(t, got.Token, tt.token, "its token")
				assert.Equal(t, got.Cases, tt.cases, "the valid cases before it")
				assert.Equal(t, digestOf(trace), tt.digest, "the choices of every call, in order")
			})
		}

		t.Run("returns a failure of an edge case left after the random cases stop", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Cases, s.Shrink = 2, 0
			got, trace := recorded(largest, s)
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{uint64(math.MaxUint64)}, "the upper bound")
			assert.Equal(t, got.Cases, 2, "the simplest case and random case 0")
			assert.Equal(t, digestOf(trace), "b6d6925815b049e4f616ab48ba221451dae5c4dbe138a1eaadf8319c0001449f",
				"the simplest case, random case 0 and two edge cases")
		})

		t.Run("returns the runs of the stored cases up to the failing one", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored, s.Shrink = [][]choice.Choice{integers(100), integers(950), integers(300)}, 0
			got := engine.Run(atLeast900, s)
			assert.Length(t, got.Stored, 2, "the passing case and the failing one")
			assert.Equal(t, drawValues(got.Stored[1].Case.Draws()), []any{950}, "the failing case's value")
		})

		t.Run("returns the clock of the settings to every case", func(t *testing.T) {
			t.Parallel()
			start := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
			s := settled()
			s.Clock = assert.NewControlled(start)
			var seen []time.Time
			engine.Run(func(c *engine.Case) {
				engine.Draw(c, digit, drawn)
				seen = append(seen, c.Clock().Now())
			}, s)
			assert.Length(t, seen, 10, "one case for each digit, as a repeated case ends at its draw")
			assert.Equal(t, seen[0], start, "the controlled clock's time")
		})

		t.Run("returns the system clock to every case when the settings state none", func(t *testing.T) {
			t.Parallel()
			var got assert.Clock
			engine.Run(func(c *engine.Case) { got = c.Clock() }, settled())
			assert.Equal[assert.Clock](t, got, assert.System{}, "the runtime clock")
		})

		t.Run("records the calls of the stated cases first, each under its phase and run", func(t *testing.T) {
			t.Parallel()
			s := settled(integers(5)...)
			s.Examples = [][]choice.Choice{integers(4)}
			s.Draws = []engine.Entry{{Label: drawn, Value: 3}}
			got := recordedRun(t, s, func(s engine.Settings) {
				engine.Run(ended(func(c *engine.Case) { engine.Draw(c, digit, drawn) }), s)
			})
			assert.Equal(t, got[:4], []callRecord{
				{Run: 1, Phase: "example"},
				{Run: 2, Phase: "example"},
				{Run: 3, Phase: "stored"},
				{Run: 4, Phase: "simplest"},
			}, "the case of Draws, the example, the stored case and the simplest case")
			assert.Equal(t, got[4], callRecord{Run: 5, Phase: "random"}, "random case 0 after them")
		})

		t.Run("records random cases before the first coverage check and coverage cases after it", func(t *testing.T) {
			t.Parallel()
			s := settled()
			require(engine.Requirement{Label: "even", Share: 0.6})(&s)
			phases := phasesOf(recordedRun(t, s, func(s engine.Settings) {
				engine.Run(ended(func(c *engine.Case) {
					if engine.Draw(c, small, drawn)%2 == 0 {
						c.Classify("even")
					}
				}), s)
			}))
			lastRandom := -1
			for i, phase := range phases {
				if phase == "random" {
					lastRandom = i
				}
			}
			firstCoverage := slices.Index(phases, "coverage")
			ordered := lastRandom >= 0 && firstCoverage > lastRandom
			assert.True(t, ordered, "every random case before the first coverage case")
			assert.Contains(t, phases, "edge", "the edge cases")
		})

		t.Run("records the prefix case of a random case of two choices under the phase prefix", func(t *testing.T) {
			t.Parallel()
			phases := phasesOf(recordedRun(t, settled(), func(s engine.Settings) {
				engine.Run(ended(func(c *engine.Case) {
					engine.Draw(c, small, drawn)
					engine.Draw(c, small, "second")
				}), s)
			}))
			assert.Equal(t, phases[:3], []string{"simplest", "random", "prefix"}, "random case 0 and its prefix case")
		})

		t.Run("records the replay of a failing case, and each run of its shrink and explanation", func(t *testing.T) {
			t.Parallel()
			var r engine.Result
			phases := phasesOf(recordedRun(t, settled(), func(s engine.Settings) {
				r = engine.Run(ended(choiceOf(1001)), s)
			}))
			replay := slices.Index(phases, "replay")
			assert.True(t, replay > 0, "the replay follows the failing case")
			counts := map[string]int{}
			for _, phase := range phases[replay:] {
				counts[phase]++
			}
			assert.Equal(t, counts["replay"], 1, "one replay")
			assert.Equal(t, counts["shrink"]+counts["explain"], r.Runs, "a record for each run of the budget")
			assert.True(t, counts["shrink"] > 0 && counts["explain"] > 0, "runs of both")
			assert.Equal(t, phases[len(phases)-1], "explain", "the explanation last")
		})
	})

	t.Run("RunReplay", func(t *testing.T) {
		t.Parallel()

		atLeast5 := func(c *engine.Case) {
			if engine.Draw(c, digit, drawn) >= 5 {
				c.Report(assert.Failure{Assertion: "big"}, false)
			}
		}

		t.Run("returns the failure of the replayed case", func(t *testing.T) {
			t.Parallel()
			choices, err := token.Decode("prop1:AAc")
			assert.NoError(t, err, "the token decodes")
			got := engine.RunReplay(atLeast5, settled(), choices, record.Token)
			want := engine.Result{Outcome: engine.Counterexample, Seed: referenceSeed}
			assert.Equal(t, summary(got), want, "a counterexample")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{7}, "the replayed value")
			assert.Equal(t, got.Token, "prop1:AAc", "the replayed token")
			assert.Equal(t, got.Runs, 0, "nothing shrunk")
			assert.Empty(t, got.Explanation, "nothing explained")
		})

		tests := []struct {
			name    string
			body    engine.Body
			choices []choice.Choice
			want    engine.Result
		}{
			{
				name:    "returns Passed for a replayed case that passes",
				body:    atLeast5,
				choices: integers(3),
				want:    engine.Result{Outcome: engine.Passed, Cases: 1, Seed: referenceSeed},
			},
			{
				name:    "returns Rejected for a replayed case that is rejected",
				body:    func(c *engine.Case) { engine.Draw(c, digit, drawn); c.Assume(false) },
				choices: integers(3),
				want:    engine.Result{Outcome: engine.Rejected, Rejected: 1, Seed: referenceSeed},
			},
			{
				name: "returns Vacuous for a replayed case that requests no input",
				body: func(*engine.Case) {},
				want: engine.Result{Outcome: engine.Vacuous, Cases: 1, Seed: referenceSeed},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := engine.RunReplay(tt.body, settled(), tt.choices, record.Token)
				assert.Equal(t, summary(got), tt.want, "how the replay ended")
				assert.Nil(t, got.Failing, "no failing case")
			})
		}

		t.Run("returns the clock of the settings to the case", func(t *testing.T) {
			t.Parallel()
			start := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
			s := settled()
			s.Clock = assert.NewControlled(start)
			var got time.Time
			engine.RunReplay(func(c *engine.Case) { got = c.Clock().Now() }, s, nil, record.Token)
			assert.Equal(t, got, start, "the controlled clock's time")
		})

		t.Run("records the calls of the case under the phase that it states", func(t *testing.T) {
			t.Parallel()
			got := recordedRun(t, settled(), func(s engine.Settings) {
				engine.RunReplay(ended(atLeast5), s, integers(3), record.Stored)
			})
			assert.Equal(t, got, []callRecord{{Run: 1, Phase: "stored"}}, "the one case, as stored")
		})
	})

	t.Run("Conclude", func(t *testing.T) {
		t.Parallel()

		t.Run("shrinks and explains a failing case that the bridge decoded", func(t *testing.T) {
			t.Parallel()
			failing := engine.Bridge(choiceOf(1001), largestBytes, engine.Settings{})
			got := engine.Conclude(choiceOf(1001), settled(), failing)
			want := []engine.Explained{
				{Label: drawn, Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
			}
			assert.Equal(t, summary(got), engine.Result{Outcome: engine.Counterexample, Seed: referenceSeed},
				"a counterexample without a valid case")
			assert.Equal(t, got.Explanation, want, "the minimal value with its nearest passing value")
			assert.Equal(t, got.Token, "prop1:AOkH", "the minimal case's token")
		})

		t.Run("returns Flaky for a failing case whose replay passes", func(t *testing.T) {
			t.Parallel()
			calls := 0
			once := func(c *engine.Case) {
				calls++
				engine.Draw(c, digit, drawn)
				if calls == 1 {
					c.Report(assert.Failure{Assertion: "once"}, false)
				}
			}
			got := engine.Conclude(once, settled(), engine.Bridge(once, []byte{7}, engine.Settings{}))
			divergence := &engine.Divergence{
				What:     engine.VerdictDifference,
				Index:    1,
				Recorded: engine.Identity{Assertion: "once"},
			}
			want := engine.Result{Outcome: engine.Flaky, Seed: referenceSeed, Divergence: divergence}
			assert.Equal(t, summary(got), want, "the replay passes")
		})

		t.Run("records the bridged case, its replay, its shrink and its explanation", func(t *testing.T) {
			t.Parallel()
			body := ended(choiceOf(1001))
			var r engine.Result
			phases := phasesOf(recordedRun(t, settled(), func(s engine.Settings) {
				r = engine.Conclude(body, s, engine.Bridge(body, largestBytes, s))
			}))
			assert.Equal(t, phases[:2], []string{"fuzz", "replay"}, "the bridged case and its replay")
			counts := map[string]int{}
			for _, phase := range phases[2:] {
				counts[phase]++
			}
			assert.Equal(t, counts["shrink"]+counts["explain"], r.Runs, "a record for each run of the budget")
			assert.Equal(t, len(counts), 2, "runs of the shrink and of the explanation alone")
		})
	})
}

// TestRunnerAllocs checks the allocation ceilings of a run, of a replay
// and of concluding a bridged case, and that Valid allocates nothing.
func TestRunnerAllocs(t *testing.T) {
	small := engine.Integer(0, 1000)
	body := func(c *engine.Case) { engine.Draw(c, small, drawn) }
	s, seven := settled(), integers(7)
	big := fromThousand()
	failing := engine.Bridge(big, largestBytes, engine.Settings{})
	assert.MaxAllocs(t, func() { engine.Run(body, s) }, runAllocs, "a run of 100 cases")
	assert.MaxAllocs(t, func() { engine.RunReplay(body, s, seven, record.Token) }, replayAllocs, "a replay of one case")
	assert.MaxAllocs(t, func() { engine.Conclude(big, s, failing) }, concludeAllocs, "the conclusion of a case")
	assert.MaxAllocs(t, func() { _ = engine.Vacuous.Valid() }, 0, "Valid allocates nothing")
}

// BenchmarkRunner measures a run of 100 cases, a replay of one case, the
// conclusion of a bridged case, and Valid.
func BenchmarkRunner(b *testing.B) {
	small := engine.Integer(0, 1000)
	body := func(c *engine.Case) { engine.Draw(c, small, drawn) }

	b.Run("Run", func(b *testing.B) {
		var got engine.Result
		s := settled()
		c := bench.Start(b).MaxAllocs(runAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Run(body, s)
		}
		assert.Equal(b, got.Cases, 100, "the valid cases")
	})

	b.Run("RunReplay", func(b *testing.B) {
		var got engine.Result
		s, seven := settled(), integers(7)
		c := bench.Start(b).MaxAllocs(replayAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.RunReplay(body, s, seven, record.Token)
		}
		assert.Equal(b, got.Outcome, engine.Passed, "the replayed case passes")
	})

	b.Run("Conclude", func(b *testing.B) {
		var got engine.Result
		s, big := settled(), fromThousand()
		failing := engine.Bridge(big, largestBytes, engine.Settings{})
		c := bench.Start(b).MaxAllocs(concludeAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Conclude(big, s, failing)
		}
		assert.Equal(b, got.Token, "prop1:AOkH", "the minimal case's token")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.Vacuous.Valid()
		}
		assert.True(b, got, "Vacuous is an outcome")
	})
}

// fromThousand returns the body that draws an integer in [0, 10000] and
// fails from 1,001.
func fromThousand() engine.Body {
	tenThousand := engine.Integer(0, 10_000)
	return func(c *engine.Case) {
		if engine.Draw(c, tenThousand, drawn) >= 1001 {
			c.Report(assert.Failure{Assertion: "big"}, false)
		}
	}
}
