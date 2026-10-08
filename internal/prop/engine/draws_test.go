// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"slices"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocations of a run whose case of the entries ends at a refusal of
// the next entry, measured.
const (
	// refusedRunAllocs are the allocations of a run whose case of twoSteps
	// refuses the first entry.
	refusedRunAllocs = 11
	// servedRunAllocs are the allocations of such a run whose case serves one
	// value first: those of a refused run, and the queue of the value.
	servedRunAllocs = 12
)

// refusal is the reason of the refusals that the specs of the trace make.
const refusal = `"get" is not among the actions that the step lists`

// The step entries of the specs of the trace.
var (
	// putStep is a step of the action put outside a concurrent section.
	putStep = &engine.MachineStep{Action: "put", Client: -1}
	// getStep is a step of the action get on client 1.
	getStep = &engine.MachineStep{Action: "get", Client: 1}
	// twoSteps are the entries of a case of the two steps, put first.
	twoSteps = []engine.Entry{{Step: putStep}, {Step: getStep}}
)

// refusedRun are the settings of a run whose case of twoSteps ends at a
// refusal in each body of the trace's ceilings.
var refusedRun = engine.Settings{
	Seed:       referenceSeed,
	Cases:      engine.DefaultCases,
	MaxChoices: engine.MaxChoices,
	Draws:      twoSteps,
}

// call is what one call of a body drew, and the choices it made.
type call struct {
	// values are the values the call drew, in order.
	values []any
	// choices are the choices the call made.
	choices []choice.Choice
}

// TestDraws checks the case of a run's Draws entries, the examples a run
// tries after it, the refusal of an entry, and the trace that a machine
// follows in the case, with the draws and the entries of the definition's
// vectors.
func TestDraws(t *testing.T) {
	t.Parallel()

	count, name := engine.Integer(0, 9), engine.String(sizes(t, 0, 3))
	countAndName := func(c *engine.Case) []any {
		return []any{engine.Draw(c, count, "count"), engine.Draw(c, name, "name")}
	}

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the case of the entries first, each draw decoding its entry's value", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}, {Label: "name", Value: "ab"}}
			_, calls := drawsRun(countAndName, s)
			assert.Equal(t, calls[0].values, []any{4, "ab"}, "the values of the entries")
			assert.True(t, sameChoices(calls[0].choices, []choice.Choice{integers(4)[0], sequence(10, 11)}),
				"the choices that decode to them")
		})

		t.Run("takes the target of each draw past the last entry", func(t *testing.T) {
			t.Parallel()
			named := engine.String(sizes(t, 1, 3))
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}}
			_, calls := drawsRun(func(c *engine.Case) []any {
				return []any{engine.Draw(c, count, "count"), engine.Draw(c, named, "name")}
			}, s)
			assert.Equal(t, calls[0].values, []any{4, "0"}, "the entry's value, then the simplest name")
		})

		t.Run("takes the target of a choice outside a draw", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}}
			_, calls := drawsRun(func(c *engine.Case) []any {
				return []any{c.Integer(digitRange).Magnitude(), engine.Draw(c, count, "count")}
			}, s)
			assert.Equal(t, calls[0].values, []any{uint64(0), 4}, "the target, then the entry's value")
		})

		t.Run("leaves an entry past the last draw unread", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}, {Label: "name", Value: "ab"}}
			got, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, count, "count")} }, s)
			assert.Equal(t, calls[0].values, []any{4}, "the first entry's value")
			assert.Equal(t, got.Outcome, engine.Passed, "the run passes")
		})

		t.Run("runs the examples after the case of the entries and before the stored cases", func(t *testing.T) {
			t.Parallel()
			s := settled(integers(4)...)
			s.Draws = []engine.Entry{{Label: "count", Value: 1}}
			s.Examples = []engine.Example{{Choices: integers(2)}, {Choices: integers(3)}}
			_, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, count, "count")} }, s)
			firsts := []any{calls[0].values[0], calls[1].values[0], calls[2].values[0], calls[3].values[0]}
			assert.Equal(t, firsts, []any{1, 2, 3, 4}, "the entries, the two examples, then the stored case")
		})

		failsFromFive := func(c *engine.Case) {
			if engine.Draw(c, count, "count") >= 5 {
				c.Report(assert.Failure{Assertion: "big"}, false)
			}
		}

		t.Run("shrinks a failing case of the entries", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 9}}
			got := engine.Run(failsFromFive, s)
			assert.Equal(t, got.Outcome, engine.Counterexample, "a counterexample")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{5}, "the smallest count that fails")
		})

		t.Run("shrinks a failing example", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Examples = []engine.Example{{Choices: integers(9)}}
			got := engine.Run(failsFromFive, s)
			assert.Equal(t, got.Outcome, engine.Counterexample, "a counterexample")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{5}, "the smallest count that fails")
		})

		t.Run("refuses an entry whose label differs at the entry's label, before any other case", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "name", Value: "ab"}}
			got, calls := drawsRun(countAndName, s)
			assert.Equal(t, got, engine.Result{Refused: got.Refused}, "the refusal, and nothing else")
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, f.Path, fault.Path{fault.Index(0), fault.Field("label")}, "the first entry's label")
			assert.Equal(t, f.Reason, `the draw labelled "count" takes the entry labelled "name"`, "both labels")
			assert.ErrorIsNot(t, got.Refused, engine.ErrCannotInvert, "no inverse ran")
			assert.Length(t, calls, 1, "no other case runs")
		})

		refusals := []struct {
			name       string
			give       engine.Generator[int]
			wantReason string
		}{
			{
				name:       "refuses a value that the draw's generator does not produce at the entry's value",
				give:       count,
				wantReason: "12 is outside [0, 9]",
			},
			{
				name:       "refuses the entry of a draw whose generator has no inverse at the entry's value",
				give:       count.Map(func(v int) int { return v }),
				wantReason: "map has no inverse",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = []engine.Entry{{Label: "count", Value: 12}}
				got := engine.Run(func(c *engine.Case) { engine.Draw(c, tt.give, "count") }, s)
				assert.ErrorIs(t, got.Refused, engine.ErrCannotInvert, "no choices decode to the entry's value")
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(t, f.Path, fault.Path{fault.Index(0), fault.Field("value")}, "the first entry's value")
				assert.Equal(t, f.Reason, tt.wantReason, "why no choices decode to the value")
			})
		}

		t.Run("refuses a later entry at its index and at the part that no choice produces", func(t *testing.T) {
			t.Parallel()
			digits := engine.List(count, sizes(t, 0, 3))
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}, {Label: "digits", Value: []int{1, 2, 12}}}
			got := engine.Run(func(c *engine.Case) {
				engine.Draw(c, count, "count")
				engine.Draw(c, digits, "digits")
			}, s)
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			want := fault.Path{fault.Index(1), fault.Field("value"), fault.Index(2)}
			assert.Equal(t, f.Path, want, "the third element of the second entry's value")
			assert.Equal(t, f.Reason, "12 is outside [0, 9]", "why no choices decode to the element")
		})

		t.Run("refuses the entry of a draw that takes a step entry at the entry's label", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = twoSteps
			got, _ := drawsRun(countAndName, s)
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, f.Path, fault.Path{fault.Index(0), fault.Field("label")}, "the first entry's label")
			assert.Equal(t, f.Reason, `the draw labelled "count" takes the step entry of "put"`,
				"the draw and the step")
		})
	})

	t.Run("Trace", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the trace of the case of the entries alone", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = twoSteps
			var mu sync.Mutex
			var followed []bool
			engine.Run(func(c *engine.Case) {
				_, ok := c.Trace()
				mu.Lock()
				followed = append(followed, ok)
				mu.Unlock()
				c.Integer(digitRange)
			}, s)
			assert.Equal(t, []any{followed[0], slices.Index(followed[1:], true)}, []any{true, -1},
				"the case of the entries follows a trace, and no later case does")
		})
	})

	t.Run("Names", func(t *testing.T) {
		t.Parallel()

		t.Run("reports whether a step entry from the next entry on names the action", func(t *testing.T) {
			t.Parallel()
			var got []bool
			entries := []engine.Entry{{Step: putStep}, {Label: "count", Value: 4}, {Step: getStep}}
			followTrace(entries, func(_ *engine.Case, tr engine.Trace) {
				got = append(got, tr.Names("put"), tr.Names("get"), tr.Names("flush"))
				tr.Take()
				got = append(got, tr.Names("put"))
			})
			assert.Equal(t, got, []bool{true, true, false, false}, "put and get, then get alone once put is taken")
		})
	})

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the next entry when it is a step entry", func(t *testing.T) {
			t.Parallel()
			var got []any
			entries := []engine.Entry{{Label: "count", Value: 4}, {Step: getStep}}
			followTrace(entries, func(c *engine.Case, tr engine.Trace) {
				_, before := tr.Next()
				engine.Draw(c, count, "count")
				step, next := tr.Next()
				tr.Take()
				_, after := tr.Next()
				got = []any{before, step, next, after}
			})
			assert.Equal(t, got, []any{false, *getStep, true, false},
				"no step at the draw entry, the step entry after it, and none past the entries")
		})
	})

	t.Run("Take", func(t *testing.T) {
		t.Parallel()

		t.Run("serves its values to the next requests, and leaves the next entry to the next draw", func(t *testing.T) {
			t.Parallel()
			var got []any
			entries := []engine.Entry{{Step: putStep}, {Label: "count", Value: 4}}
			followTrace(entries, func(c *engine.Case, tr engine.Trace) {
				tr.Take(1, 2)
				first, second := c.Integer(digitRange).Magnitude(), c.Integer(digitRange).Magnitude()
				got = []any{first, second, engine.Draw(c, count, "count")}
			})
			assert.Equal(t, got, []any{uint64(1), uint64(2), 4}, "the step's values, then the draw's entry")
		})
	})

	t.Run("Prepare", func(t *testing.T) {
		t.Parallel()

		t.Run("serves its values to the next requests before their targets", func(t *testing.T) {
			t.Parallel()
			var got []uint64
			followTrace(twoSteps, func(c *engine.Case, tr engine.Trace) {
				tr.Prepare(3, 5)
				for range 3 {
					got = append(got, c.Integer(digitRange).Magnitude())
				}
			})
			assert.Equal(t, got, []uint64{3, 5, 0}, "the two values, then the target")
		})
	})

	t.Run("Refuse", func(t *testing.T) {
		t.Parallel()

		t.Run("refuses the next entry at its step, before any other case", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = twoSteps
			calls := 0
			got := engine.Run(func(c *engine.Case) {
				calls++
				if tr, ok := c.Trace(); ok {
					tr.Take(1, 0)
					tr.Refuse(refusal)
				}
			}, s)
			assert.Equal(t, got, engine.Result{Refused: got.Refused}, "the refusal, and nothing else")
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, []any{f.Path, f.Reason}, []any{fault.Path{fault.Index(1), fault.Field("step")}, refusal},
				"the second entry's step, and the reason")
			assert.Equal(t, calls, 1, "no other case runs")
		})
	})
}

// traceAllocs are the cases of the allocation ceilings of the trace's
// methods that serve values or refuse, each a whole run whose case of
// twoSteps ends at a refusal.
var traceAllocs = []alloctest.Case{
	{
		Name: "Prepare",
		Call: func(assert.TB) {
			engine.Run(func(c *engine.Case) {
				tr, _ := c.Trace()
				tr.Prepare(1)
				tr.Refuse(refusal)
			}, refusedRun)
		},
		Allocs: servedRunAllocs,
	},
	{
		Name: "Take",
		Call: func(assert.TB) {
			engine.Run(func(c *engine.Case) {
				tr, _ := c.Trace()
				tr.Take(1)
				tr.Refuse(refusal)
			}, refusedRun)
		},
		Allocs: servedRunAllocs,
	},
	{
		Name: "Refuse",
		Call: func(assert.TB) {
			engine.Run(func(c *engine.Case) {
				tr, _ := c.Trace()
				tr.Refuse(refusal)
			}, refusedRun)
		},
		Allocs: refusedRunAllocs,
	},
}

// TestDrawsAllocs checks the allocation ceilings of the trace's methods.
func TestDrawsAllocs(t *testing.T) {
	alloctest.Check(t, traceAllocs)
	c, tr := leakedTrace()
	assert.MaxAllocs(t, func() { _, _ = c.Trace() }, 0, "Trace allocates nothing")
	assert.MaxAllocs(t, func() { _ = tr.Names("get") }, 0, "Names allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = tr.Next() }, 0, "Next allocates nothing")
}

// BenchmarkDraws measures each method of the trace under its ceiling: the
// methods that serve values or refuse on a whole run that the case of the
// entries ends, and the others on the trace of a case whose body has ended.
func BenchmarkDraws(b *testing.B) {
	for _, c := range traceAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
	b.Run("Trace", func(b *testing.B) {
		var got bool
		cs, _ := leakedTrace()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			_, got = cs.Trace()
		}
		assert.True(b, got, "the case of the entries follows them")
	})
	b.Run("Names", func(b *testing.B) {
		var got bool
		_, tr := leakedTrace()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = tr.Names("get")
		}
		assert.True(b, got, "the second entry names get")
	})
	b.Run("Next", func(b *testing.B) {
		var got engine.MachineStep
		_, tr := leakedTrace()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = tr.Next()
		}
		assert.Equal(b, got, *putStep, "the first entry's step")
	})
}

// followTrace runs the case of entries, whose body calls follow with the
// case and its trace.
func followTrace(entries []engine.Entry, follow func(c *engine.Case, tr engine.Trace)) {
	s := settled()
	s.Cases = 1
	s.Draws = entries
	engine.Run(func(c *engine.Case) {
		if tr, ok := c.Trace(); ok {
			follow(c, tr)
		}
	}, s)
}

// leakedTrace returns the case of twoSteps, whose body took no entry, and
// its trace, for a caller to call after the run.
func leakedTrace() (*engine.Case, engine.Trace) {
	var c *engine.Case
	var tr engine.Trace
	followTrace(twoSteps, func(followed *engine.Case, trace engine.Trace) { c, tr = followed, trace })
	return c, tr
}

// drawsRun runs a body that returns what draw draws under s, and returns
// the result and what each call drew and chose, in call order.
func drawsRun(draw func(*engine.Case) []any, s engine.Settings) (engine.Result, []call) {
	var calls []call
	result := engine.Run(func(c *engine.Case) {
		var values []any
		defer func() { calls = append(calls, call{values: values, choices: c.Choices()}) }()
		values = draw(c)
	}, s)
	return result, calls
}
