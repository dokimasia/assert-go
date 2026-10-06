// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestExample checks the examples of a run: an example of values, whose
// draws take the values without a choice, and how a run reports it.
func TestExample(t *testing.T) {
	t.Parallel()

	digit := engine.Integer(0, 9)
	bit := engine.Integer(0, 1)

	t.Run("Example", func(t *testing.T) {
		t.Parallel()

		t.Run("gives each draw the next value, without a choice or a span", func(t *testing.T) {
			t.Parallel()
			var first *engine.Case
			s := settled()
			s.Examples = []engine.Example{{Values: []any{42, 7}}}
			engine.Run(func(c *engine.Case) {
				if first == nil {
					first = c
				}
				engine.Draw(c, digit, "a")
				engine.Draw(c, digit, "b")
			}, s)
			assert.Equal(t, drawValues(first.Draws()), []any{42, 7}, "the values, 42 outside the digit's domain too")
			assert.Empty(t, first.Choices(), "no choice")
			assert.Empty(t, first.Spans(), "no span")
		})

		t.Run("decodes a draw past the last value from the targets", func(t *testing.T) {
			t.Parallel()
			var first *engine.Case
			s := settled()
			s.Examples = []engine.Example{{Values: []any{5}}}
			engine.Run(func(c *engine.Case) {
				if first == nil {
					first = c
				}
				engine.Draw(c, digit, "a")
				engine.Draw(c, digit, "b")
			}, s)
			assert.Equal(t, drawValues(first.Draws()), []any{5, 0}, "the value, then the target")
			assert.True(t, sameChoices(first.Choices(), integers(0)), "the choice of the second draw")
		})

		t.Run("reports a failing example of values as found, without a token or a shrink", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Examples = []engine.Example{{Values: []any{42}}}
			r := engine.Run(func(c *engine.Case) {
				if engine.Draw(c, digit, drawn) >= 5 {
					c.Report(reported, false)
				}
			}, s)
			assert.Equal(t, []any{r.Outcome, r.Cases, r.Token, r.Runs, len(r.Others), len(r.Explanation)},
				[]any{engine.Counterexample, 0, "", 0, 0, 0}, "the case as found")
			assert.Equal(t, drawValues(r.Failing.Case.Draws()), []any{42}, "the stated value, not shrunk")
			assert.True(t, r.Failing.Case.Valued(), "an example of values")
		})

		t.Run("counts a passing example of values as a valid case", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Examples = []engine.Example{{Values: []any{1}}}
			r := engine.Run(func(c *engine.Case) { engine.Draw(c, bit, drawn) }, s)
			assert.Equal(t, []any{r.Outcome, r.Cases}, []any{engine.Passed, 3},
				"the example and the two bits, which the example does not exhaust")
		})

		t.Run("replays an example of choices", func(t *testing.T) {
			t.Parallel()
			var got []any
			s := settled()
			s.Examples = []engine.Example{{Choices: integers(6)}}
			engine.Run(func(c *engine.Case) { got = append(got, engine.Draw(c, digit, drawn)) }, s)
			assert.Equal(t, got[0], any(6), "the value of the choices")
		})
	})

	t.Run("Valued", func(t *testing.T) {
		t.Parallel()

		t.Run("reports whether the case states its values", func(t *testing.T) {
			t.Parallel()
			var valued []bool
			s := settled()
			s.Cases = 1
			s.Examples = []engine.Example{{Values: []any{3}}, {Choices: integers(4)}}
			engine.Run(func(c *engine.Case) {
				valued = append(valued, c.Valued())
				engine.Draw(c, digit, drawn)
			}, s)
			assert.Equal(t, valued[:3], []bool{true, false, false}, "the example of values alone")
		})
	})
}
