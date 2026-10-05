// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/token"
)

// The parameters of the machine requests that the tests make.
const (
	// remainingActions is the number of actions from the one that a swarm
	// choice decides to the last.
	remainingActions = 3
	// stepMaximum and stepMean are the maximum and the mean of a run of
	// steps.
	stepMaximum, stepMean = 10, 3
	// clientCount is the number of values of a uniform choice of a client.
	clientCount = 3
	// machineCases is the number of random cases whose draws the tests
	// compare with the random package's.
	machineCases = 40
	// repeatedRuns is the number of runs that a repeated case asks for.
	repeatedRuns = 3
)

// The run of the failing repeat that the case tree stops: under
// repeatedSeed, random cases 0 and 3, and no other of the first 80, draw
// repeatedValue from thousand. A run of repeatedCases valid cases ends
// before random case 80.
const (
	repeatedSeed  = 389
	repeatedValue = 10
	repeatedCases = 20
)

// thousand are the bounds [0, 999] of the one choice of a repeated case.
var thousand = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(999))

// The allocations of a whole replayed case whose body records a step or
// asks for runs, measured.
const (
	// stepCaseAllocs are the allocations of a case that records one step: the
	// case's own three, and the growth of its steps.
	stepCaseAllocs = 4
	// repeatCaseAllocs are the allocations of a case that asks for two runs
	// and passes both: the own three of the case and of its repeat.
	repeatCaseAllocs = 6
)

// stepWeights are the weights of the actions that a step's index chooses
// among in the tests.
var stepWeights = []uint64{1, 2, 3}

// TestMachine checks the requests that a machine's run of steps makes on a
// case, the steps that the case records, and the repetition of a case.
func TestMachine(t *testing.T) {
	t.Parallel()

	t.Run("Keep", func(t *testing.T) {
		t.Parallel()

		t.Run("draws as random.Keep draws with odds of 1 in 2", func(t *testing.T) {
			t.Parallel()
			for index := range uint64(machineCases) {
				for _, kept := range []bool{false, true} {
					var got bool
					engine.Generate(func(c *engine.Case) { got = c.Keep(kept, remainingActions) }, referenceSeed, index,
						nil)
					twin := random.ForCase(referenceSeed, index)
					assert.Equal(t, got, random.Keep(&twin, 1, 2, kept, remainingActions),
						"the swarm choice of the case")
				}
			}
		})

		t.Run("keeps the last action without consuming the stream while no earlier one is kept", func(t *testing.T) {
			t.Parallel()
			var kept bool
			var next uint64
			e := engine.Generate(func(c *engine.Case) {
				kept, next = c.Keep(false, 1), c.Integer(digitRange).Magnitude()
			}, referenceSeed, 0, nil)
			twin := random.ForCase(referenceSeed, 0)
			assert.True(t, kept, "the last action")
			assert.Equal(t, next, random.Integer(&twin, digitRange).Magnitude(), "the next choice's draw")
			assert.True(t, sameChoices(e.Case.Choices(), []choice.Choice{unsigned(1), unsigned(next)}),
				"the swarm choice is recorded as 1")
		})

		t.Run("disables an action in the simplest case and keeps it in an edge case", func(t *testing.T) {
			t.Parallel()
			got := requestedIn(func(c *engine.Case) any { return c.Keep(true, remainingActions) })
			assert.Equal(t, []any{got[0], got[2]}, []any{false, true}, "the target 0, then the edge 1")
		})
	})

	t.Run("Weighted", func(t *testing.T) {
		t.Parallel()

		t.Run("draws as random.Weighted draws", func(t *testing.T) {
			t.Parallel()
			for index := range uint64(machineCases) {
				var got int
				engine.Generate(func(c *engine.Case) { got = c.Weighted(stepWeights) }, referenceSeed, index, nil)
				twin := random.ForCase(referenceSeed, index)
				assert.Equal(t, got, random.Weighted(&twin, stepWeights), "the index of the case")
			}
		})

		t.Run("chooses the first index in the simplest case and in an edge case", func(t *testing.T) {
			t.Parallel()
			got := requestedIn(func(c *engine.Case) any { return c.Weighted(stepWeights) })
			assert.Equal(t, []any{got[0], got[2]}, []any{0, 0}, "the target and the edge 0")
		})

		t.Run("replays the index of the last action, and takes the first for an index past it", func(t *testing.T) {
			t.Parallel()
			var got []int
			body := func(c *engine.Case) { got = append(got, c.Weighted(stepWeights)) }
			engine.Replay(body, integers(2), nil)
			engine.Replay(body, integers(3), nil)
			assert.Equal(t, got, []int{2, 0}, "the bounds [0, 2] admit 2, and coerce 3 to their target")
		})
	})

	t.Run("Continue", func(t *testing.T) {
		t.Parallel()

		t.Run("continues by a coin of the mean in one more", func(t *testing.T) {
			t.Parallel()
			for index := range uint64(machineCases) {
				var got bool
				engine.Generate(func(c *engine.Case) { got = c.Continue(1, stepMaximum, stepMean) }, referenceSeed,
					index, nil)
				twin := random.ForCase(referenceSeed, index)
				assert.Equal(t, got, twin.Coin(stepMean, stepMean+1), "the flag of the case")
			}
		})

		t.Run("stops without consuming the stream at the maximum", func(t *testing.T) {
			t.Parallel()
			var more bool
			var next uint64
			engine.Generate(func(c *engine.Case) {
				more, next = c.Continue(stepMaximum, stepMaximum, stepMean), c.Integer(digitRange).Magnitude()
			}, referenceSeed, 0, nil)
			twin := random.ForCase(referenceSeed, 0)
			assert.False(t, more, "the run is at its maximum")
			assert.Equal(t, next, random.Integer(&twin, digitRange).Magnitude(), "the next choice's draw")
		})

		t.Run("stops in the simplest case, and gives an edge case one step", func(t *testing.T) {
			t.Parallel()
			got := requestedIn(func(c *engine.Case) any {
				return []bool{c.Continue(0, stepMaximum, stepMean), c.Continue(1, stepMaximum, stepMean)}
			})
			assert.Equal(t, []any{got[0], got[2]}, []any{[]bool{false, false}, []bool{true, false}},
				"the target stops, and the edge continues once")
		})
	})

	t.Run("Uniform", func(t *testing.T) {
		t.Parallel()

		t.Run("draws uniformly below n", func(t *testing.T) {
			t.Parallel()
			for index := range uint64(machineCases) {
				var got uint64
				engine.Generate(func(c *engine.Case) { got = c.Uniform(clientCount, 0) }, referenceSeed, index, nil)
				twin := random.ForCase(referenceSeed, index)
				assert.Equal(t, got, twin.Below(clientCount), "the choice of the case")
			}
		})

		t.Run("returns 0 below 1 without consuming the stream", func(t *testing.T) {
			t.Parallel()
			var got, next uint64
			engine.Generate(func(c *engine.Case) {
				got, next = c.Uniform(1, 0), c.Integer(digitRange).Magnitude()
			}, referenceSeed, 0, nil)
			twin := random.ForCase(referenceSeed, 0)
			assert.Equal(t, []uint64{got, next}, []uint64{0, random.Integer(&twin, digitRange).Magnitude()},
				"the one value, then the next choice's draw")
		})

		t.Run("takes its target in the simplest case and its edge in an edge case", func(t *testing.T) {
			t.Parallel()
			got := requestedIn(func(c *engine.Case) any { return c.Uniform(clientCount, clientCount-1) })
			assert.Equal(t, []any{got[0], got[2]}, []any{uint64(0), uint64(clientCount - 1)},
				"the target, then the edge")
		})
	})

	t.Run("SpanFrom", func(t *testing.T) {
		t.Parallel()

		t.Run("opens a span from an earlier choice to the last choice that f makes", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				start := c.Position()
				c.Integer(digitRange)
				c.SpanFrom(start, "put", func() { c.Integer(digitRange) })
				c.Integer(digitRange)
			}, nil, nil)
			spans := e.Case.Spans()
			assert.Length(t, spans, 1, "the step's span")
			assert.Equal(t, []any{spans[0].Label, spans[0].Start, spans[0].End}, []any{"put", 0, 2},
				"the span covers the choice before f and the one inside it")
		})
	})

	t.Run("Step", func(t *testing.T) {
		t.Parallel()

		t.Run("records each step after the draws so far", func(t *testing.T) {
			t.Parallel()
			digit := engine.Integer(0, 9)
			e := engine.Replay(func(c *engine.Case) {
				engine.Draw(c, digit, "capacity")
				c.Step(engine.MachineStep{Action: "put", Client: -1})
				engine.Draw(c, digit, "v")
				c.Step(engine.MachineStep{Action: "get", Client: 1})
				c.Step(engine.MachineStep{Action: "deliver", Client: -1, Drain: true})
			}, nil, nil)
			assert.Equal(t, e.Case.Steps(), []engine.RecordedStep{
				{Action: "put", Client: -1, Draws: 1},
				{Action: "get", Client: 1, Draws: 2},
				{Action: "deliver", Client: -1, Drain: true, Draws: 2},
			}, "each step with the draws before it")
		})
	})

	t.Run("Steps", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Steps()[0].Action = "changed"
			assert.Equal(t, c.Steps(), []engine.RecordedStep{{Action: "put", Client: -1, Draws: 1}},
				"the recorded step")
		})
	})

	t.Run("Repeat", func(t *testing.T) {
		t.Parallel()

		t.Run("runs a passing case again on its choices up to the runs that it asked for", func(t *testing.T) {
			t.Parallel()
			var runs [][]choice.Choice
			e := engine.Replay(func(c *engine.Case) {
				c.Repeat(repeatedRuns)
				c.Integer(digitRange)
				runs = append(runs, c.Choices())
			}, integers(4), nil)
			assert.Equal(t, e.Status, engine.CasePassed, "every run passes")
			assert.Length(t, runs, repeatedRuns, "the case and its repeats")
			assert.True(t, sameRecords(runs, [][]choice.Choice{integers(4), integers(4), integers(4)}),
				"every run replays the case's choices")
		})

		t.Run("keeps the largest count of runs that the case asked for", func(t *testing.T) {
			t.Parallel()
			runs := 0
			engine.Replay(func(c *engine.Case) {
				c.Repeat(repeatedRuns)
				c.Repeat(1)
				runs++
			}, nil, nil)
			assert.Equal(t, runs, repeatedRuns, "three runs")
		})

		t.Run("ends the case with the first run that fails, with its record", func(t *testing.T) {
			t.Parallel()
			runs := 0
			e := engine.Replay(func(c *engine.Case) {
				c.Repeat(repeatedRuns + 1)
				runs++
				if runs == 2 {
					c.Report(assert.Failure{Assertion: "lost"}, false)
				}
			}, nil, nil)
			assert.Equal(t, []any{e.Status, runs, e.Case.Failures()[0].Assertion},
				[]any{engine.CaseFailed, 2, "lost"}, "the second run's failure ends the case")
		})

		t.Run("reports on more workers what one worker reports for a case that fails on a repeat", func(t *testing.T) {
			t.Parallel()
			s := settled()
			one := engine.Run(failsOnRepeat(), s)
			s.Workers = fourWorkers
			assert.Equal(t, summary(engine.Run(failsOnRepeat(), s)), summary(one),
				"the counts and the outcome of one worker")
			assert.Equal(t, one.Outcome, engine.Counterexample, "a repeat of a case of 3 modulo 4 fails")
		})

		t.Run("keeps one worker's calls of a failing repeat that the tree stops on four workers", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Seed, s.Cases = repeatedSeed, repeatedCases
			var results []engine.Result
			var counts []int
			recordsOf := func(s engine.Settings) []callRecord {
				body, runs := repeatsTested()
				run := func(s engine.Settings) { results = append(results, summary(engine.Run(body, s))) }
				records := recordedRun(t, s, run)
				counts = append(counts, runs())
				return records
			}
			one := recordsOf(s)
			s.Workers = fourWorkers
			assert.Equal(t, recordsOf(s), one, "the calls of one worker")
			assert.Equal(t, results[1], results[0], "the counts and the outcome of one worker")
			assert.Equal(t, counts, []int{2, 4},
				"one worker stops random case 3 at its choice, and four run its failing repeat")
		})
	})
}

// repeatsTested returns a body that asks for two runs of its case and makes
// a call before and after its one choice from thousand, with the count of
// the runs of repeatedValue. The fourth run of repeatedValue fails. On four
// workers, random case 3 of repeatedSeed starts after the runner has taken
// random case 0, so its first run is the third of repeatedValue and its
// repeat the failing fourth. The tree then stops it at its choice, as one
// worker does, where no run of it counts.
func repeatsTested() (engine.Body, func() int) {
	var mu sync.Mutex
	runs := make(map[uint64]int)
	body := func(c *engine.Case) {
		c.Repeat(2)
		assert.True(c, true, "the call before the choice")
		v := c.Integer(thousand).Magnitude()
		mu.Lock()
		runs[v]++
		n := runs[v]
		mu.Unlock()
		assert.True(c, true, "the call after the choice")
		if v == repeatedValue && n == 4 {
			c.Report(assert.Failure{Assertion: "late"}, false)
		}
	}
	return body, func() int {
		mu.Lock()
		defer mu.Unlock()
		return runs[repeatedValue]
	}
}

// failsOnRepeat returns a body that asks for two runs of its case, and fails
// the second run of a case whose one choice is 3 modulo 4. Every run of a
// case passes its first run, so each case runs twice, and the parity of the
// runs of its choices tells the two runs apart. The choice spans the
// unsigned range, so no two cases of one choice that can fail run at once.
func failsOnRepeat() engine.Body {
	var mu sync.Mutex
	runs := make(map[string]int)
	return func(c *engine.Case) {
		c.Repeat(2)
		v := c.Rand().Uint64()
		mu.Lock()
		tok := token.Encode(c.Choices())
		runs[tok]++
		second := runs[tok]%2 == 0
		mu.Unlock()
		if v%4 == 3 && second {
			c.Report(assert.Failure{Assertion: "late"}, false)
		}
	}
}

// requestedIn runs a body that makes request and then a choice over the
// unsigned range, as a run of four valid cases of the reference seed, and
// returns what request returned in each call: the simplest case first, and
// the first edge case third. A call keeps what request returned before its
// second choice, which keeps the domain from running out, so every case of
// the run keeps one.
func requestedIn(request func(c *engine.Case) any) []any {
	s := settled()
	s.Cases = 4
	var got []any
	var mu sync.Mutex
	engine.Run(func(c *engine.Case) {
		v := request(c)
		mu.Lock()
		got = append(got, v)
		mu.Unlock()
		c.Rand().Uint64()
	}, s)
	return got
}

// machineAllocs are the cases of the allocation ceilings of the machine
// requests and records, each a whole replayed case whose body makes one.
var machineAllocs = []alloctest.Case{
	{
		Name:   "Keep",
		Call:   func(assert.TB) { engine.Replay(func(c *engine.Case) { c.Keep(true, remainingActions) }, nil, nil) },
		Allocs: valueCaseAllocs,
	},
	{
		Name:   "Weighted",
		Call:   func(assert.TB) { engine.Replay(func(c *engine.Case) { c.Weighted(stepWeights) }, nil, nil) },
		Allocs: valueCaseAllocs,
	},
	{
		Name: "Continue",
		Call: func(assert.TB) {
			engine.Replay(func(c *engine.Case) { c.Continue(0, stepMaximum, stepMean) }, nil, nil)
		},
		Allocs: valueCaseAllocs,
	},
	{
		Name:   "Uniform",
		Call:   func(assert.TB) { engine.Replay(func(c *engine.Case) { c.Uniform(clientCount, 0) }, nil, nil) },
		Allocs: valueCaseAllocs,
	},
	{
		Name: "SpanFrom",
		Call: func(assert.TB) {
			engine.Replay(func(c *engine.Case) {
				c.SpanFrom(c.Position(), "put", func() { c.Integer(digitRange) })
			}, nil, nil)
		},
		Allocs: spanCaseAllocs,
	},
	{
		Name: "Step",
		Call: func(assert.TB) {
			engine.Replay(func(c *engine.Case) { c.Step(engine.MachineStep{Action: "put", Client: -1}) }, nil, nil)
		},
		Allocs: stepCaseAllocs,
	},
	{
		Name:   "Repeat",
		Call:   func(assert.TB) { engine.Replay(func(c *engine.Case) { c.Repeat(2) }, nil, nil) },
		Allocs: repeatCaseAllocs,
	},
}

// TestMachineAllocs checks the allocation ceilings of the machine requests
// and records.
func TestMachineAllocs(t *testing.T) {
	alloctest.Check(t, machineAllocs)
	c := leaked()
	assert.MaxAllocs(t, func() { _ = c.Steps() }, copyAllocs, "Steps allocates its copy")
}

// BenchmarkMachine measures each machine request and record under its
// ceiling, on a whole replayed case, and Steps on a case whose body has
// ended.
func BenchmarkMachine(b *testing.B) {
	for _, c := range machineAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
	b.Run("Steps", func(b *testing.B) {
		var got []engine.RecordedStep
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Steps()
		}
		assert.Length(b, got, 1, "the case's one step")
	})
}
