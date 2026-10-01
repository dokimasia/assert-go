// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// fourWorkers is the number of workers of the runs that a test compares
// with a run on one worker.
const fourWorkers = 4

// runReport is what a run reports, without the cases it ran: how it ended
// with its counts, its minimal case's token, the runs that shrinking and
// explaining spent, the explanation, and the other failures.
type runReport struct {
	// summary is how the run ended, with its counts.
	summary engine.Result
	// token is the token of the minimal case.
	token string
	// runs are the runs that shrinking and explaining spent.
	runs int
	// explanation explains each draw of the minimal case.
	explanation []engine.Explained
	// others are the other failures.
	others []other
}

// concurrency counts the cases that run at once, and the most that did.
type concurrency struct {
	// mu guards the fields below.
	mu sync.Mutex
	// running is the number of cases running.
	running int
	// peak is the most cases that ran at once.
	peak int
}

// enter counts one more running case, and returns the function that counts
// it out.
func (g *concurrency) enter() func() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.running++
	g.peak = max(g.peak, g.running)
	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.running--
	}
}

// now returns the number of cases running.
func (g *concurrency) now() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.running
}

// most returns the most cases that ran at once.
func (g *concurrency) most() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.peak
}

// TestAhead checks a run on more than one worker: it reports what a run on
// one worker reports, it runs at most Workers cases at once, it starts a
// case at most Workers positions past the case the runner waits for, and it
// returns once every case it started has ended.
func TestAhead(t *testing.T) {
	t.Parallel()

	digit, small := engine.Integer(0, 9), engine.Integer(0, 1000)
	tenThousand, whole := engine.Integer(0, 10_000), engine.Integer[uint64](0, math.MaxUint64)
	even := func(v int) bool { return v%2 == 0 }
	trues := engine.Boolean(1, 2).Filter(func(b bool) bool { return b })
	high := engine.Integer(0, 3).Filter(func(v int) bool { return v >= 2 })
	classify := func(label string, holds func(int) bool) engine.Body {
		return func(c *engine.Case) {
			if holds(engine.Draw(c, small, drawn)) {
				c.Classify(label)
			}
		}
	}
	atLeast := func(g engine.Generator[int], from int) engine.Body {
		return func(c *engine.Case) {
			if engine.Draw(c, g, drawn) >= from {
				c.Report(assert.Failure{Assertion: "big"}, false)
			}
		}
	}

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			body   engine.Body
			adjust func(*engine.Settings)
		}{
			{
				name: "reports the pass of a body that never fails",
				body: func(c *engine.Case) { engine.Draw(c, small, drawn) },
			},
			{
				name: "reports the rejection of a body that rejects every case",
				body: func(c *engine.Case) { engine.Draw(c, small, drawn); c.Assume(false) },
			},
			{
				name: "reports the counts of a body that rejects four cases in five",
				body: func(c *engine.Case) { c.Assume(engine.Draw(c, small, drawn)%5 == 0) },
			},
			{
				name: "reports the rejection of each case larger than MaxChoices",
				body: func(c *engine.Case) {
					if len(engine.Draw(c, engine.List(small, unbounded(t, 0)), drawn)) >= 2 {
						c.Report(assert.Failure{Assertion: "long"}, false)
					}
				},
				adjust: func(s *engine.Settings) { s.MaxChoices = 4 },
			},
			{name: "reports a body that requests no input", body: func(*engine.Case) {}},
			{
				name: "reports a pass once every input of a domain is tested",
				body: func(c *engine.Case) { engine.Draw(c, digit, drawn) },
			},
			{
				name: "reports the cases of a domain whose repeats have prefix cases",
				body: func(c *engine.Case) { engine.Draw(c, engine.Integer(0, 30), drawn) },
			},
			{
				name:   "reports a met coverage requirement",
				body:   classify("even", even),
				adjust: require(engine.Requirement{Label: "even", Share: 0.3}),
			},
			{
				name:   "reports a requirement the last check leaves unmet",
				body:   classify("even", even),
				adjust: require(engine.Requirement{Label: "even", Share: 0.6}),
			},
			{
				name: "reports the minimal counterexample and its nearest passing value",
				body: atLeast(tenThousand, 1001),
			},
			{
				name: "reports a failure at the maximum that an edge case finds",
				body: func(c *engine.Case) {
					if engine.Draw(c, whole, drawn) == math.MaxUint64 {
						c.Report(assert.Failure{Assertion: "largest"}, false)
					}
				},
			},
			{
				name:   "reports a failure of a later stored case",
				body:   atLeast(small, 900),
				adjust: func(s *engine.Settings) { s.Stored = [][]choice.Choice{integers(100), integers(950)} },
			},
			{
				name:   "reports the first failing case when shrinking is off",
				body:   atLeast(tenThousand, 1001),
				adjust: func(s *engine.Settings) { s.Shrink = 0 },
			},
			{
				name:   "reports a failure of an edge case left after the random cases stop",
				body:   atLeast(tenThousand, 10_000),
				adjust: func(s *engine.Settings) { s.Cases, s.Shrink = 2, 0 },
			},
			{
				name: "reports the cases of a filter whose rejected attempts enter the tree",
				body: func(c *engine.Case) { engine.Draw(c, trues, drawn) },
			},
			{
				name: "reports the prefix case of a repeat that followed rejected attempts",
				body: func(c *engine.Case) {
					first, second := engine.Draw(c, high, drawn), engine.Draw(c, high, "second")
					if first == 3 && second == 2 {
						c.Report(assert.Failure{Assertion: "pair"}, false)
					}
				},
				adjust: func(s *engine.Settings) { s.Seed = 2 },
			},
			{
				name: "reports the counterexample of a filter that shrinks",
				body: atLeast(engine.Integer(0, 1000).Filter(even), 501),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				one, four := settled(), settled()
				if tt.adjust != nil {
					tt.adjust(&one)
					tt.adjust(&four)
				}
				four.Workers = fourWorkers
				assert.Equal(t, reportOf(engine.Run(tt.body, four)), reportOf(engine.Run(tt.body, one)),
					"what four workers report, against one")
			})
		}

		t.Run("reports on four workers the counterexample that the definition pins for one", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Workers = fourWorkers
			got := engine.Run(atLeast(tenThousand, 1001), s)
			assert.Equal(t, reportOf(got), runReport{
				summary: engine.Result{Outcome: engine.Counterexample, Cases: 1, Seed: referenceSeed},
				token:   "prop1:AOkH",
				runs:    27,
				explanation: []engine.Explained{
					{Label: drawn, Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
				},
				others: []other{},
			}, "the counterexample of one worker")
		})

		t.Run("reports on four workers the filter cases that the definition pins for one", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Workers = fourWorkers
			got := engine.Run(func(c *engine.Case) { engine.Draw(c, trues, drawn) }, s)
			want := engine.Result{Outcome: engine.Passed, Cases: 3, Rejected: 1, Seed: referenceSeed}
			assert.Equal(t, summary(got), want,
				"true after no, one and two rejected attempts, and three rejected attempts")
		})

		t.Run("runs at most Workers cases at once", func(t *testing.T) {
			t.Parallel()
			var gauge concurrency
			s := settled()
			s.Workers = fourWorkers
			engine.Run(func(c *engine.Case) {
				defer gauge.enter()()
				engine.Draw(c, small, drawn)
			}, s)
			assert.InRange(t, gauge.most(), 1, fourWorkers, "the most cases that ran at once")
		})

		t.Run("returns once every case it started has ended", func(t *testing.T) {
			t.Parallel()
			var gauge concurrency
			var longest atomic.Int64
			s := settled()
			s.Workers, s.Shrink, s.MaxChoices = fourWorkers, 0, 1<<18
			got := engine.Run(func(c *engine.Case) {
				defer gauge.enter()()
				if engine.Draw(c, digit, drawn) == 0 {
					c.Report(assert.Failure{Assertion: "zero"}, false)
					return
				}
				draws := int64(1)
				defer func() { raise(&longest, draws) }()
				for {
					engine.Draw(c, digit, drawn)
					draws++
				}
			}, s)
			assert.Equal(t, got.Outcome, engine.Counterexample, "the simplest case fails")
			assert.Equal(t, gauge.now(), 0, "no case runs after Run returns")
			assert.True(t, longest.Load() < 1<<18, "a case started ahead ends at its next draw, not at its cap")
		})

		t.Run("starts only the cases up to Workers positions past the case the runner waits for", func(t *testing.T) {
			t.Parallel()
			signed := engine.Integer(-1_000_000_000, 1_000_000_000)
			random := make([]int, 4)
			for index := range random {
				engine.Generate(func(c *engine.Case) { random[index] = engine.Draw(c, signed, drawn) },
					referenceSeed, uint64(index), nil)
			}
			synctest.Test(t, func(t *testing.T) {
				var mu sync.Mutex
				var started []int
				release, ended := make(chan struct{}), make(chan engine.Result, 1)
				s := settled()
				s.Workers = fourWorkers
				go func() {
					ended <- engine.Run(func(c *engine.Case) {
						v := engine.Draw(c, signed, drawn)
						mu.Lock()
						started = append(started, v)
						mu.Unlock()
						if v == random[1] {
							<-release
						}
					}, s)
				}()
				synctest.Wait()
				mu.Lock()
				got := slices.Sorted(slices.Values(started))
				mu.Unlock()
				close(release)
				<-ended
				want := []int{0, random[0], -1_000_000_000, random[1], 1_000_000_000, random[2], 1, random[3]}
				assert.Equal(t, got, slices.Sorted(slices.Values(want)),
					"the simplest case and positions 0 to 6 while the runner waits for random case 1 at position 2")
			})
		})
	})
}

// reportOf returns what r reports, without the cases it ran.
func reportOf(r engine.Result) runReport {
	return runReport{summary: summary(r), token: r.Token, runs: r.Runs, explanation: r.Explanation, others: others(r)}
}

// raise sets v to n when n is larger.
func raise(v *atomic.Int64, n int64) {
	for {
		old := v.Load()
		if n <= old || v.CompareAndSwap(old, n) {
			return
		}
	}
}
