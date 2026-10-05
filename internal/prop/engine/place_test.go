// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The places, the generators and the bounds of the place tests.
var (
	// drainPlace is the place of the fourth drain step, of the action flush.
	drainPlace = engine.Place{Part: engine.DrainPart, Position: 3, Positioned: true, Action: "flush", Acting: true}
	// settlePlace is the place of the settle check.
	settlePlace = engine.Place{Part: engine.SettlePart}
	// firstGenerator and replayGenerator are the generators of a draw in the
	// first call of a body and in its replay.
	firstGenerator, replayGenerator = engine.Integer(0, 1_000_000_000), engine.Integer(0, 1000)
	// firstRange is the bounds of a request in the first call of a body. Its
	// replay requests digitRange.
	firstRange = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1_000_000_000))
)

// whereCase is a case of TestPlace: a body that fails in every call and runs
// give in each, what its replay differs in, and where the replay made the
// request or observed the fingerprint.
type whereCase struct {
	name     string
	give     func(calls int, c *engine.Case)
	wantWhat engine.Difference
	want     engine.Where
}

// TestPlace checks where the replay of a failing case made the request or
// observed the fingerprint that differs: the innermost draw that ran, and
// the place of a machine's steps that the body set.
func TestPlace(t *testing.T) {
	t.Parallel()

	t.Run("SetPlace", func(t *testing.T) {
		t.Parallel()
		tests := []whereCase{
			{
				name:     "gives the request that the replay makes in a draw the place that it set",
				give:     func(calls int, c *engine.Case) { c.SetPlace(drainPlace); otherDraw(calls, c, "n") },
				wantWhat: engine.RequestDifference,
				want:     engine.Where{Label: "n", Drawing: true, Place: drainPlace, Placed: true},
			},
			{
				name:     "gives the fingerprint that the replay observes the place that it set",
				give:     func(calls int, c *engine.Case) { c.SetPlace(settlePlace); c.Observe(uint64(calls)) },
				wantWhat: engine.FingerprintDifference,
				want:     engine.Where{Place: settlePlace, Placed: true},
			},
			{
				name:     "gives a request outside every draw the place that it set",
				give:     func(calls int, c *engine.Case) { c.SetPlace(drainPlace); otherRequest(calls, c) },
				wantWhat: engine.RequestDifference,
				want:     engine.Where{Place: drainPlace, Placed: true},
			},
		}
		checkWheres(t, tests)
	})

	t.Run("ClearPlace", func(t *testing.T) {
		t.Parallel()
		tests := []whereCase{
			{
				name: "leaves the requests after it outside a machine's steps",
				give: func(calls int, c *engine.Case) {
					c.SetPlace(drainPlace)
					c.ClearPlace()
					otherDraw(calls, c, "n")
				},
				wantWhat: engine.RequestDifference,
				want:     engine.Where{Label: "n", Drawing: true},
			},
		}
		checkWheres(t, tests)
	})

	t.Run("Where", func(t *testing.T) {
		t.Parallel()
		tests := []whereCase{
			{
				name: "states the label of the innermost draw that made the request",
				give: func(calls int, c *engine.Case) {
					inner := engine.Composite(func(c *engine.Case) int { return otherDraw(calls, c, "inner") })
					engine.Draw(c, inner, "outer")
				},
				wantWhat: engine.RequestDifference,
				want:     engine.Where{Label: "inner", Drawing: true},
			},
			{
				name: "states the label of the outer draw once its inner draw has ended",
				give: func(calls int, c *engine.Case) {
					engine.Draw(c, engine.Composite(func(c *engine.Case) int {
						engine.Draw(c, firstGenerator, "inner")
						otherRequest(calls, c)
						return 0
					}), "outer")
				},
				wantWhat: engine.RequestDifference,
				want:     engine.Where{Label: "outer", Drawing: true},
			},
			{
				name: "states no draw for a request after every draw has ended",
				give: func(calls int, c *engine.Case) {
					engine.Draw(c, firstGenerator, "n")
					otherRequest(calls, c)
				},
				wantWhat: engine.RequestDifference,
				want:     engine.Where{},
			},
		}
		checkWheres(t, tests)
	})
}

// TestPlaceAllocs checks that SetPlace and ClearPlace allocate nothing.
func TestPlaceAllocs(t *testing.T) {
	c := leaked()
	assert.MaxAllocs(t, func() { c.SetPlace(drainPlace) }, 0, "SetPlace allocates nothing")
	assert.MaxAllocs(t, func() { c.ClearPlace() }, 0, "ClearPlace allocates nothing")
}

// BenchmarkPlace measures SetPlace and ClearPlace under a ceiling of no
// allocation.
func BenchmarkPlace(b *testing.B) {
	b.Run("SetPlace", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.SetPlace(drainPlace)
		}
		assert.Length(b, cs.Failures(), 1, "the case's one record")
	})

	b.Run("ClearPlace", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.ClearPlace()
		}
		assert.Length(b, cs.Failures(), 1, "the case's one record")
	})
}

// checkWheres runs each case of tests as a subtest of t: the body fails in
// every call, and the run is flaky because its replay differs.
func checkWheres(t *testing.T, tests []whereCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			got, _ := traced(func(c *engine.Case) string {
				calls++
				tt.give(calls, c)
				return "always"
			}, settled())
			assert.Equal(t, got.Outcome, engine.Flaky, "a flaky run")
			assert.Equal(t, got.Divergence.What, tt.wantWhat, "what the replay differs in")
			assert.Equal(t, got.Divergence.Where, tt.want, "where the replay differs")
		})
	}
}

// otherDraw draws under label from firstGenerator in the first call of a
// body, and from replayGenerator in every later call.
func otherDraw(calls int, c *engine.Case, label string) int {
	if calls == 1 {
		return engine.Draw(c, firstGenerator, label)
	}
	return engine.Draw(c, replayGenerator, label)
}

// otherRequest requests firstRange in the first call of a body, and
// digitRange in every later call.
func otherRequest(calls int, c *engine.Case) {
	if calls == 1 {
		c.Integer(firstRange)
		return
	}
	c.Integer(digitRange)
}
