// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// call is what one call of a body drew, and the choices it made.
type call struct {
	// values are the values the call drew, in order.
	values []any
	// choices are the choices the call made.
	choices []choice.Choice
}

// TestDraws checks the case of a run's Draws entries, the examples a run
// tries after it, and the refusal of an entry, with the draws and the
// entries of the definition's vectors.
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
			s.Examples = [][]choice.Choice{integers(2), integers(3)}
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
			s.Examples = [][]choice.Choice{integers(9)}
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
	})
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
