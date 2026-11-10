// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"context"
	"math"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/coverage"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/internal/record"
)

// The allocation ceilings of a run.
const (
	// runAllocs is the ceiling of the allocations of a passing run of 100
	// cases of an integer in [0, 1000]: its 136 cases, its case tree and
	// its executor.
	runAllocs = 610
	// replayAllocs is the ceiling of the allocations of a replay of one
	// case that draws one integer.
	replayAllocs = 12
	// storedAllocs is the ceiling of the allocations of a run of one stored
	// case that draws one integer and passes: the case, and the list of the
	// runs.
	storedAllocs = 13
	// concludeAllocs is the ceiling of the allocations of concluding the
	// bridged case of 10,000 for a body that fails from 1,001: the replay
	// that confirms it, with the list of where it made its request, the
	// runs of its shrink and of its explanation, and the result.
	concludeAllocs = 590
	// countAllocs is the ceiling of the allocations of counting a bridged
	// case that draws one integer and passes under a Confirm of 1: the tally,
	// and the replay with the list of where it made its request.
	countAllocs = 23
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

		t.Run("reports true for the seven outcomes and false past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t,
				[]engine.Outcome{
					engine.Passed, engine.Counterexample, engine.Flaky,
					engine.Rejected, engine.CoverageUnmet, engine.Vacuous, engine.Skipped,
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
				name:   "returns Skipped at the simplest case of a body that skips every case",
				body:   func(c *engine.Case) { engine.Draw(c, small, drawn); c.Skip("unsupported") },
				want:   engine.Result{Outcome: engine.Skipped, Seed: referenceSeed, Reason: "unsupported"},
				calls:  1,
				digest: "2cbef72e43e8b4c7fe6a69e3589a182916e560f109a97cb8e8739de2aaf823d4",
			},
			{
				name: "counts the valid cases before the case that skips",
				body: func(c *engine.Case) {
					if engine.Draw(c, small, drawn) == 1000 {
						c.Skip("the largest value")
					}
				},
				want: engine.Result{
					Outcome: engine.Skipped, Cases: 3, Seed: referenceSeed, Reason: "the largest value",
				},
				calls:  5,
				digest: "ffa653582aea8eebece28b91dd7d7983f991ba246485d9d8fafbda069de96265",
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

		t.Run("returns the fault of a case that ends with a fault, before a shrink", func(t *testing.T) {
			t.Parallel()
			got, trace := recorded(func(c *engine.Case) {
				if engine.Draw(c, small, drawn) == 1000 {
					c.ReportFault(errEnvironment, true)
				}
			}, settled())
			assert.ErrorIs(t, got.Fault, errEnvironment, "the fault of the largest value")
			assert.Equal(t, summary(got), engine.Result{}, "a run that ends with a fault states nothing else")
			assert.Nil(t, got.Failing, "no failing case")
			assert.Length(t, trace, 5, "the cases up to the edge case of the largest value")
		})

		t.Run("returns Skipped for a case that skips where an earlier case made a request", func(t *testing.T) {
			t.Parallel()
			calls := 0
			got, trace := recorded(func(c *engine.Case) {
				calls++
				if engine.Draw(c, digit, "first") == 0 && calls > 1 {
					c.Skip("zero again")
				}
				engine.Draw(c, digit, "second")
			}, settled())
			want := engine.Result{Outcome: engine.Skipped, Cases: 3, Seed: referenceSeed, Reason: "zero again"}
			assert.Equal(t, summary(got), want, "the case that skips, without a leaf that diverges")
			assert.Equal(t, len(trace), 4, "the calls of the body")
			assert.Equal(t, digestOf(trace), "83eb67c61a519617a6e42972f7df65c84004d3deffcc06cdf5cf53bbaf8cfb0d",
				"the choices of every call, in order")
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
				adjust: func(s *engine.Settings) { s.Stored = []engine.Stored{{Choices: integers(950)}} },
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
				name: "returns a failure of a later stored case after an earlier one passes",
				body: atLeast900,
				adjust: func(s *engine.Settings) {
					s.Stored = []engine.Stored{{Choices: integers(100)}, {Choices: integers(950)}}
				},
				cases: 1,
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
			s.Stored = []engine.Stored{{Choices: integers(100)}, {Choices: integers(950)}, {Choices: integers(300)}}
			s.Shrink = 0
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
			s.Examples = []engine.Example{{Choices: integers(4)}}
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
			assert.Equal(t, []int{r.Replays, r.Reproduced}, []int{1, 1}, "one replay, which reproduces the failure")
		})

		t.Run("replays a failing case Replays times before its shrink, and counts the replays that reproduce it",
			func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Replays = 3
				var r engine.Result
				phases := phasesOf(recordedRun(t, s, func(s engine.Settings) {
					r = engine.Run(ended(choiceOf(1001)), s)
				}))
				replay := slices.Index(phases, "replay")
				assert.Equal(t, phases[replay:replay+4], []string{"replay", "replay", "replay", "shrink"},
					"three replays, and then the shrink")
				assert.Equal(t, []any{r.Outcome, r.Replays, r.Reproduced}, []any{engine.Counterexample, 3, 3},
					"each replay reproduces the failure")
			})

		t.Run("returns Counterexample for a failing case that one replay of three reproduces", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Replays = 3
			calls := 0
			r := engine.Run(func(c *engine.Case) {
				calls++
				if engine.Draw(c, tenThousand, drawn) > 1000 && calls%2 == 0 {
					c.Report(assert.Failure{Assertion: "even"}, false)
				}
			}, s)
			assert.Equal(t, []any{r.Outcome, r.Replays, r.Reproduced}, []any{engine.Counterexample, 3, 1},
				"the failing case is an even call, so its replays are an odd, an even and an odd call")
			assert.Equal(t, r.Failing.Identity.Assertion, "even", "the failure of the minimal case")
		})

		t.Run("returns Flaky with the difference of the first replay when no replay reproduces the failure",
			func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Replays = 3
				big := 0
				var r engine.Result
				phases := phasesOf(recordedRun(t, s, func(s engine.Settings) {
					r = engine.Run(ended(func(c *engine.Case) {
						if engine.Draw(c, tenThousand, drawn) <= 1000 {
							return
						}
						big++
						if big == 1 {
							c.Report(assert.Failure{Assertion: "once"}, false)
						}
						if big > 2 {
							c.Report(assert.Failure{Assertion: "later"}, false)
						}
					}), s)
				}))
				divergence := &engine.Divergence{
					What: engine.VerdictDifference, Index: 1, Recorded: engine.Identity{Assertion: "once"},
				}
				assert.Equal(t, []any{r.Outcome, r.Divergence}, []any{engine.Flaky, divergence},
					"the first replay passes where the case failed, and the later two fail otherwise")
				assert.Equal(t, []int{r.Replays, r.Reproduced}, []int{3, 0}, "no replay of three reproduces it")
				assert.False(t, slices.Contains(phases, "shrink"), "the run shrinks nothing")
			})

		t.Run("returns the fault of a later replay that ends with one", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Replays = 3
			replays := 0
			r := engine.Run(func(c *engine.Case) {
				if engine.Draw(c, tenThousand, drawn) <= 1000 {
					return
				}
				if replays++; replays == 3 {
					c.ReportFault(errEnvironment, true)
				}
				c.Report(assert.Failure{Assertion: "big"}, false)
			}, s)
			assert.ErrorIs(t, r.Fault, errEnvironment, "the second replay ends the run with its fault")
		})

		t.Run("replays nothing under a shrink budget of 0", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Replays, s.Shrink = 3, 0
			r := engine.Run(choiceOf(1001), s)
			assert.Equal(t, []any{r.Outcome, r.Replays, r.Reproduced}, []any{engine.Counterexample, 0, 0},
				"the failing case is reported as found")
		})

		t.Run("runs every run of the body inside Settings.Around", func(t *testing.T) {
			t.Parallel()
			var depth, runs, outside atomic.Int32
			s := settled()
			s.Around = func(run func(context.Context)) {
				depth.Add(1)
				defer depth.Add(-1)
				run(t.Context())
			}
			failing := choiceOf(1001)
			r := engine.Run(func(c *engine.Case) {
				runs.Add(1)
				if depth.Load() != 1 {
					outside.Add(1)
				}
				failing(c)
			}, s)
			assert.Equal(t, r.Outcome, engine.Counterexample, "the run finds the failing case")
			assert.True(t, r.Runs > 0, "the run shrinks and explains the case")
			assert.True(t, runs.Load() > int32(r.Runs), "the cases, the replay, the shrink and the explanation ran")
			assert.Equal(t, outside.Load(), int32(0), "no run of the body runs outside Around")
		})

		t.Run("replays each valid case whose count is a multiple of Confirm right after it", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Cases, s.Confirm = 10, 5
			got := recordedRun(t, s, func(s engine.Settings) {
				engine.Run(ended(func(c *engine.Case) { engine.Draw(c, small, drawn) }), s)
			})
			want := []callRecord{
				{Run: 1, Phase: "simplest"},
				{Run: 2, Phase: "random"},
				{Run: 4, Phase: "random"},
				{Run: 5, Phase: "edge"},
				{Run: 6, Phase: "random"},
				{Run: 7, Phase: "confirm"},
				{Run: 8, Phase: "edge"},
				{Run: 9, Phase: "random"},
				{Run: 11, Phase: "random"},
				{Run: 12, Phase: "random"},
				{Run: 13, Phase: "random"},
				{Run: 14, Phase: "confirm"},
			}
			assert.Equal(t, got, want, "the replays of the fifth and the tenth valid case")
		})

		t.Run("replays on four workers the cases that one worker replays", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Cases, s.Confirm = 30, 4
			body := ended(func(c *engine.Case) { engine.Draw(c, small, drawn) })
			one := recordedRun(t, s, func(s engine.Settings) { engine.Run(body, s) })
			s.Workers = fourWorkers
			four := recordedRun(t, s, func(s engine.Settings) { engine.Run(body, s) })
			assert.Equal(t, four, one, "the calls of the cases and of the replays of one worker")
		})

		t.Run("does not replay an example of values", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Confirm = 1
			s.Examples = []engine.Example{{Values: []any{7}}}
			var r engine.Result
			got := recordedRun(t, s, func(s engine.Settings) {
				r = engine.Run(ended(func(c *engine.Case) { c.Observe(uint64(engine.Draw(c, digit, drawn))) }), s)
			})
			assert.Equal(t, r.Outcome, engine.Passed, "no replay draws the target in place of the stated 7")
			assert.Equal(t, phasesOf(got)[:3], []string{"example", "simplest", "confirm"},
				"the example, then the simplest case and its replay")
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
			{
				name:    "returns Skipped for a replayed case that skips",
				body:    func(c *engine.Case) { engine.Draw(c, digit, drawn); c.Skip("unsupported") },
				choices: integers(3),
				want:    engine.Result{Outcome: engine.Skipped, Seed: referenceSeed, Reason: "unsupported"},
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

		t.Run("replays the case under a Confirm of 1", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Confirm = 1
			got := recordedRun(t, s, func(s engine.Settings) {
				engine.RunReplay(ended(atLeast5), s, integers(3), record.Token)
			})
			want := []callRecord{{Run: 1, Phase: "token"}, {Run: 2, Phase: "confirm"}}
			assert.Equal(t, got, want, "the case and its replay")
		})

		t.Run("returns Flaky without a token for a case whose replay differs", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Confirm = 1
			calls := uint64(0)
			got := engine.RunReplay(func(c *engine.Case) {
				engine.Draw(c, digit, drawn)
				calls++
				c.Observe(calls)
			}, s, integers(3), record.Token)
			divergence := &engine.Divergence{
				What: engine.FingerprintDifference, Recorded: uint64(1), Replayed: uint64(2),
			}
			want := engine.Result{Outcome: engine.Flaky, Cases: 1, Seed: referenceSeed, Divergence: divergence}
			assert.Equal(t, summary(got), want, "the replay observes 2 where the case observed 1")
			assert.Equal(t, got.Token, "", "a flaky run has no token")
		})
	})

	t.Run("RunStored", func(t *testing.T) {
		t.Parallel()

		atLeast5 := func(c *engine.Case) {
			if engine.Draw(c, digit, drawn) >= 5 {
				c.Report(assert.Failure{Assertion: "big"}, false)
			}
		}

		t.Run("returns a pass that counts the stored cases", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(3)}, {Choices: integers(4)}}
			got := engine.RunStored(atLeast5, s)
			want := engine.Result{Outcome: engine.Passed, Cases: 2, Seed: referenceSeed}
			assert.Equal(t, summary(got), want, "a pass of both")
			assert.Length(t, got.Stored, 2, "the runs of both stored cases")
		})

		t.Run("returns the first failing stored case as found, with its token", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(3)}, {Choices: integers(7)}, {Choices: integers(9)}}
			got := engine.RunStored(atLeast5, s)
			want := engine.Result{Outcome: engine.Counterexample, Cases: 1, Seed: referenceSeed}
			assert.Equal(t, summary(got), want, "a counterexample after one valid case")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{7}, "the stored value, not shrunk")
			assert.Equal(t, got.Token, "prop1:AAc", "the token of the stored case")
			assert.Length(t, got.Stored, 2, "the runs up to the failing case")
		})

		t.Run("returns Skipped at the first stored case that skips, with the runs up to it", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(3)}, {Choices: integers(7)}, {Choices: integers(9)}}
			got := engine.RunStored(func(c *engine.Case) {
				if engine.Draw(c, digit, drawn) == 7 {
					c.Skip("seven")
				}
			}, s)
			want := engine.Result{Outcome: engine.Skipped, Cases: 1, Seed: referenceSeed, Reason: "seven"}
			assert.Equal(t, summary(got), want, "a skip after one valid case")
			assert.Length(t, got.Stored, 2, "the runs up to the case that skips")
		})

		t.Run("returns the fault of the first stored case that ends with one", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(3)}, {Choices: integers(7)}, {Choices: integers(9)}}
			got := engine.RunStored(func(c *engine.Case) {
				if engine.Draw(c, digit, drawn) == 7 {
					c.ReportFault(errEnvironment, true)
				}
			}, s)
			assert.ErrorIs(t, got.Fault, errEnvironment, "the fault of the second stored case")
			assert.Length(t, got.Stored, 2, "the runs up to the case that ends with a fault")
		})

		t.Run("records the calls of each case under stored", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(3)}, {Choices: integers(4)}}
			got := recordedRun(t, s, func(s engine.Settings) { engine.RunStored(ended(atLeast5), s) })
			assert.Equal(t, got, []callRecord{{Run: 1, Phase: "stored"}, {Run: 2, Phase: "stored"}}, "both cases")
		})

		t.Run("replays each stored case under a Confirm of 1", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored, s.Confirm = []engine.Stored{{Choices: integers(3)}, {Choices: integers(4)}}, 1
			got := recordedRun(t, s, func(s engine.Settings) { engine.RunStored(ended(atLeast5), s) })
			want := []callRecord{
				{Run: 1, Phase: "stored"},
				{Run: 2, Phase: "confirm"},
				{Run: 3, Phase: "stored"},
				{Run: 4, Phase: "confirm"},
			}
			assert.Equal(t, got, want, "each stored case and its replay")
		})

		never := func(*engine.Case) bool { return false }
		always := func(*engine.Case) bool { return true }
		traceOf := func(values ...int) []engine.Entry {
			entries := make([]engine.Entry, len(values))
			for i, v := range values {
				entries[i] = engine.Entry{Label: drawn, Value: v}
			}
			return entries
		}

		t.Run(
			"returns the failure of the trace of a stored case whose choices decode to other values",
			func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Stored = []engine.Stored{{Choices: integers(3), Matches: never, Trace: traceOf(7)}}
				got := engine.RunStored(atLeast5, s)
				want := engine.Result{Outcome: engine.Counterexample, Seed: referenceSeed}
				assert.Equal(t, summary(got), want, "a counterexample of the trace, as found")
				assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{7}, "the value of the trace")
				assert.Equal(t, got.Token, "prop1:AAc", "the token of the case of the trace")
				assert.Equal(t, []any{got.Stored[0].Traced, got.Stored[0].Refusal}, []any{true, nil},
					"the case of the trace counts")
			},
		)

		t.Run("counts the trace of a stored case that takes every entry and passes", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(3), Matches: never, Trace: traceOf(4)}}
			got := engine.RunStored(atLeast5, s)
			assert.Equal(t, summary(got), engine.Result{Outcome: engine.Passed, Cases: 1, Seed: referenceSeed},
				"a pass of one valid case")
			assert.True(t, got.Stored[0].Traced, "the case of the trace counts")
			assert.Equal(t, drawValues(got.Stored[0].Case.Draws()), []any{4}, "the value of the trace")
		})

		t.Run("returns the fault of the trace of a stored case that ends with one", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(3), Matches: never, Trace: traceOf(7, 8)}}
			got := engine.RunStored(func(c *engine.Case) {
				if engine.Draw(c, digit, drawn) == 7 {
					c.ReportFault(errEnvironment, true)
				}
			}, s)
			assert.ErrorIs(t, got.Fault, errEnvironment, "the fault of the case of the trace")
			assert.True(t, got.Stored[0].Traced, "a fault counts before the entry left untaken")
		})

		choicesCount := []struct {
			name       string
			give       engine.Stored
			wantValue  int
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:      "counts the choices of a stored case that matches its entry",
				give:      engine.Stored{Choices: integers(3), Matches: always, Trace: traceOf(7)},
				wantValue: 3,
			},
			{
				name:      "counts the choices of a stored case whose entry has no counterexample",
				give:      engine.Stored{Choices: integers(3), Trace: traceOf(7)},
				wantValue: 3,
			},
			{
				name:      "counts the choices of a stored case without a trace",
				give:      engine.Stored{Choices: integers(3), Matches: never},
				wantValue: 3,
			},
			{
				name: "counts the choices of a stored case whose trace the body refuses",
				give: engine.Stored{
					Choices: integers(3), Matches: never, Trace: []engine.Entry{{Label: "n", Value: 7}},
				},
				wantValue:  3,
				wantPath:   fault.Path{fault.Index(0), fault.Field("label")},
				wantReason: `the draw labelled "value" takes the entry labelled "n"`,
			},
			{
				name:       "counts the choices of a stored case whose trace leaves an entry untaken",
				give:       engine.Stored{Choices: integers(3), Matches: never, Trace: traceOf(4, 2)},
				wantValue:  3,
				wantPath:   fault.Path{fault.Index(1)},
				wantReason: "the case leaves the entry untaken",
			},
		}
		for _, tt := range choicesCount {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Stored = []engine.Stored{tt.give}
				got := engine.RunStored(atLeast5, s)
				assert.Equal(t, summary(got), engine.Result{Outcome: engine.Passed, Cases: 1, Seed: referenceSeed},
					"a pass of one valid case")
				run := got.Stored[0]
				assert.False(t, run.Traced, "the case of the choices counts")
				assert.Equal(t, drawValues(run.Case.Draws()), []any{tt.wantValue}, "the value of the choices")
				if tt.wantReason == "" {
					assert.NoError(t, run.Refusal, "no trace ran")
					return
				}
				f := assert.ErrorAs[*fault.Error](t, run.Refusal, "the refusal of the trace")
				assert.Equal(t, []any{f.Path, f.Reason}, []any{tt.wantPath, tt.wantReason}, "the entry and why")
			})
		}

		t.Run("counts a failing case of the choices without its trace", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Stored = []engine.Stored{{Choices: integers(7), Matches: never, Trace: traceOf(3)}}
			got := engine.RunStored(atLeast5, s)
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{7}, "the value of the choices")
			assert.False(t, got.Stored[0].Traced, "the failing case of the choices counts")
		})

		recordings := []struct {
			name string
			give []engine.Entry
			want []callRecord
		}{
			{
				name: "records the calls of the case of the choices and of the trace that counts under stored",
				give: traceOf(4),
				want: []callRecord{{Run: 1, Phase: "stored"}, {Run: 2, Phase: "stored"}},
			},
			{
				name: "records the calls of the case of the choices alone when the trace does not count",
				give: traceOf(4, 2),
				want: []callRecord{{Run: 1, Phase: "stored"}},
			},
		}
		for _, tt := range recordings {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Stored = []engine.Stored{{Choices: integers(3), Matches: never, Trace: tt.give}}
				got := recordedRun(t, s, func(s engine.Settings) { engine.RunStored(ended(atLeast5), s) })
				assert.Equal(t, got, tt.want, "the calls of the cases that the run records")
			})
		}
	})

	t.Run("Count", func(t *testing.T) {
		t.Parallel()

		confirming := settled()
		confirming.Confirm = 1

		t.Run("returns a pass of one valid case without a replay under a Confirm of 2", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Confirm = 2
			calls := 0
			body := func(c *engine.Case) {
				calls++
				engine.Draw(c, digit, drawn)
			}
			got := engine.Count(body, s, engine.Bridge(body, []byte{7}, engine.Settings{}))
			assert.Equal(t, summary(got), engine.Result{Outcome: engine.Passed, Cases: 1, Seed: referenceSeed},
				"the case is the first valid case")
			assert.Equal(t, calls, 1, "the bridged case alone")
		})

		t.Run("returns a pass of one rejected case for a rejected case", func(t *testing.T) {
			t.Parallel()
			body := func(c *engine.Case) {
				engine.Draw(c, digit, drawn)
				c.Assume(false)
			}
			got := engine.Count(body, confirming, engine.Bridge(body, []byte{7}, engine.Settings{}))
			assert.Equal(t, summary(got), engine.Result{Outcome: engine.Passed, Rejected: 1, Seed: referenceSeed},
				"no valid case, so no replay")
		})

		t.Run("records the bridged case and its replay under a Confirm of 1", func(t *testing.T) {
			t.Parallel()
			body := ended(func(c *engine.Case) { engine.Draw(c, digit, drawn) })
			var r engine.Result
			got := recordedRun(t, confirming, func(s engine.Settings) {
				r = engine.Count(body, s, engine.Bridge(body, []byte{7}, s))
			})
			assert.Equal(t, got, []callRecord{{Run: 1, Phase: "fuzz"}, {Run: 2, Phase: "confirm"}},
				"the bridged case and its replay")
			assert.Equal(t, summary(r), engine.Result{Outcome: engine.Passed, Cases: 1, Seed: referenceSeed},
				"a replay that agrees")
		})

		t.Run("returns Flaky for a case whose replay differs", func(t *testing.T) {
			t.Parallel()
			calls := uint64(0)
			body := func(c *engine.Case) {
				engine.Draw(c, digit, drawn)
				calls++
				c.Observe(calls)
			}
			got := engine.Count(body, confirming, engine.Bridge(body, []byte{7}, engine.Settings{}))
			divergence := &engine.Divergence{
				What: engine.FingerprintDifference, Recorded: uint64(1), Replayed: uint64(2),
			}
			want := engine.Result{Outcome: engine.Flaky, Cases: 1, Seed: referenceSeed, Divergence: divergence}
			assert.Equal(t, summary(got), want, "the replay observes 2 where the case observed 1")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{7}, "the bridged case")
		})

		t.Run("returns the fault of a replay that ends with one", func(t *testing.T) {
			t.Parallel()
			calls := 0
			body := func(c *engine.Case) {
				engine.Draw(c, digit, drawn)
				calls++
				if calls == 2 {
					c.ReportFault(errEnvironment, true)
				}
			}
			got := engine.Count(body, confirming, engine.Bridge(body, []byte{7}, engine.Settings{}))
			assert.ErrorIs(t, got.Fault, errEnvironment, "the fault of the replay")
			assert.Equal(t, summary(got), engine.Result{}, "a run that ends with a fault states nothing else")
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

		t.Run("replays a failing case that the bridge decoded Replays times", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Replays = 2
			got := engine.Conclude(choiceOf(1001), s, engine.Bridge(choiceOf(1001), largestBytes, engine.Settings{}))
			assert.Equal(t, []any{got.Outcome, got.Replays, got.Reproduced}, []any{engine.Counterexample, 2, 2},
				"two replays, each of which reproduces the failure")
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

		t.Run("returns Flaky for a failing case whose replay skips", func(t *testing.T) {
			t.Parallel()
			calls := 0
			once := func(c *engine.Case) {
				calls++
				engine.Draw(c, digit, drawn)
				if calls > 1 {
					c.Skip("after")
				}
				c.Report(assert.Failure{Assertion: "once"}, false)
			}
			got := engine.Conclude(once, settled(), engine.Bridge(once, []byte{7}, engine.Settings{}))
			divergence := &engine.Divergence{
				What:     engine.VerdictDifference,
				Index:    1,
				Recorded: engine.Identity{Assertion: "once"},
			}
			want := engine.Result{Outcome: engine.Flaky, Seed: referenceSeed, Divergence: divergence}
			assert.Equal(t, summary(got), want, "the replay skips, which counts as a replay that does not fail")
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

// TestRunnerAllocs checks the allocation ceilings of a run, of a replay, of
// a run of the stored cases, of concluding a bridged case and of counting
// one, and that Valid allocates nothing.
func TestRunnerAllocs(t *testing.T) {
	small := engine.Integer(0, 1000)
	body := func(c *engine.Case) { engine.Draw(c, small, drawn) }
	s, seven := settled(), integers(7)
	stored := settled(seven...)
	big := fromThousand()
	failing := engine.Bridge(big, largestBytes, engine.Settings{})
	confirming := settled()
	confirming.Confirm = 1
	passed := engine.Bridge(body, largestBytes, engine.Settings{})
	assert.MaxAllocs(t, func() { engine.Run(body, s) }, runAllocs, "a run of 100 cases")
	assert.MaxAllocs(t, func() { engine.RunReplay(body, s, seven, record.Token) }, replayAllocs, "a replay of one case")
	assert.MaxAllocs(t, func() { engine.RunStored(body, stored) }, storedAllocs, "a run of one stored case")
	assert.MaxAllocs(t, func() { engine.Conclude(big, s, failing) }, concludeAllocs, "the conclusion of a case")
	assert.MaxAllocs(t, func() { engine.Count(body, confirming, passed) }, countAllocs, "the count of a case")
	assert.MaxAllocs(t, func() { _ = engine.Vacuous.Valid() }, 0, "Valid allocates nothing")
}

// BenchmarkRunner measures a run of 100 cases, a replay of one case, the
// conclusion and the count of a bridged case, and Valid.
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

	b.Run("RunStored", func(b *testing.B) {
		var got engine.Result
		s := settled(integers(7)...)
		c := bench.Start(b).MaxAllocs(storedAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.RunStored(body, s)
		}
		assert.Equal(b, got.Cases, 1, "the stored case passes")
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

	b.Run("Count", func(b *testing.B) {
		var got engine.Result
		s := settled()
		s.Confirm = 1
		passed := engine.Bridge(body, largestBytes, engine.Settings{})
		c := bench.Start(b).MaxAllocs(countAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Count(body, s, passed)
		}
		assert.Equal(b, got.Outcome, engine.Passed, "the replay agrees with the case")
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
