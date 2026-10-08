// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// TestEntry checks the entries of a counterexample: the draws with what the
// explain phase found, pinned to the definition's behaviour vectors, and the
// steps of a machine among them.
func TestEntry(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the nearest passing value of an integer whose value matters", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7))
			want := []prop.Entry{
				prop.Drawn{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000},
			}
			assert.Equal(t, got[counterexampleField], any(want), "the minimal value and the one below it")
		})

		t.Run("reports a draw where any value fails", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(100, 0, every), prop.Seed(7))
			want := []prop.Entry{prop.Drawn{Label: drawn, Value: 0, Relevance: prop.AnyValueFails}}
			assert.Equal(t, got[counterexampleField], any(want), "the target, where every filling fails")
		})

		t.Run("reports a draw whose rejected fillings are skipped as one where any value fails", func(t *testing.T) {
			t.Parallel()
			unique := prop.List(prop.Integer(0, 2), prop.Unique(), prop.MinSize(3), prop.MaxSize(3))
			body := func(c *prop.Case) {
				c.Draw(unique, drawn)
				fail(c, always)
			}
			got := detailOf(body, prop.Seed(0))
			want := []prop.Entry{prop.Drawn{Label: drawn, Value: []int{0, 1, 2}, Relevance: prop.AnyValueFails}}
			assert.Equal(t, got[counterexampleField], any(want), "the one list of three distinct digits")
		})

		t.Run("leaves a draw untested when no filling decodes", func(t *testing.T) {
			t.Parallel()
			seven := prop.Integer(0, 9).Filter(func(v int) bool { return v == 7 })
			body := func(c *prop.Case) {
				c.Draw(seven, drawn)
				fail(c, "seven")
			}
			got := detailOf(body, prop.Seed(2))
			want := []prop.Entry{prop.Drawn{Label: drawn, Value: 7}}
			assert.Equal(t, got[counterexampleField], any(want), "the kept value, untested")
		})

		t.Run("leaves the draws untested when shrinking is off", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Shrink(0))
			want := []prop.Entry{prop.Drawn{Label: drawn, Value: 8522}}
			assert.Equal(t, got[counterexampleField], any(want), "the first failing value, as found")
		})

		t.Run("leaves the draws untested when explaining is off", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Explain(false))
			want := []prop.Entry{prop.Drawn{Label: drawn, Value: 1001}}
			assert.Equal(t, got[counterexampleField], any(want), "the minimal value, unexplained")
		})

		t.Run("lists each step before the draws that the case recorded after it", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				machine := (*engine.Case)(c)
				machine.Step(engine.MachineStep{Action: "put", Client: -1})
				c.Draw(prop.Integer(0, 9), "v")
				machine.Step(engine.MachineStep{Action: "get", Client: 1})
				machine.Step(engine.MachineStep{Action: "deliver", Client: -1, Drain: true})
				fail(c, always)
			}
			got := detailOf(body, prop.Seed(7))
			want := []prop.Entry{
				prop.Step{Action: "put", Client: -1},
				prop.Drawn{Label: "v", Value: 0, Relevance: prop.AnyValueFails},
				prop.Step{Action: "get", Client: 1},
				prop.Step{Action: "deliver", Client: -1, Drain: true},
			}
			assert.Equal(t, got[counterexampleField], any(want), "the step, the draw after it, then the last steps")
		})

		t.Run("lists the steps of a case that draws nothing", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				machine := (*engine.Case)(c)
				machine.Step(engine.MachineStep{Action: "put", Client: -1})
				c.Rand().Uint64()
				machine.Step(engine.MachineStep{Action: "get", Client: -1})
				fail(c, always)
			}
			got := detailOf(body, prop.Seed(7))
			want := []prop.Entry{prop.Step{Action: "put", Client: -1}, prop.Step{Action: "get", Client: -1}}
			assert.Equal(t, got[counterexampleField], any(want), "both steps in order")
		})
	})
}
