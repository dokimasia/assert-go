// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestConfirm checks the replay of a failing case before shrinking: the
// first difference between the failing run and its replay makes the run
// flaky, pinned to what the definition's executable reference reports.
func TestConfirm(t *testing.T) {
	t.Parallel()

	small, wide := engine.Integer(0, 1000), engine.Integer(0, 1_000_000_000)
	smallBounds := choice.OfInteger(choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1000)))
	wideBounds := choice.OfInteger(choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1_000_000_000)))

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			body func(calls int, c *engine.Case) string
			want engine.Divergence
		}{
			{
				name: "returns a request difference for a replay that requests other bounds",
				body: func(calls int, c *engine.Case) string {
					if calls == 1 {
						engine.Draw(c, wide, "n")
					} else {
						engine.Draw(c, small, "n")
					}
					return "always"
				},
				want: engine.Divergence{
					What: engine.RequestDifference, Recorded: wideBounds, Replayed: smallBounds,
					Where: engine.Where{Label: "n", Drawing: true},
				},
			},
			{
				name: "returns a request difference for a replay that ends earlier",
				body: func(calls int, c *engine.Case) string {
					engine.Draw(c, wide, "n")
					if calls == 1 {
						engine.Draw(c, wide, "m")
					}
					return "always"
				},
				want: engine.Divergence{What: engine.RequestDifference, Index: 1, Recorded: wideBounds},
			},
			{
				name: "returns a fingerprint difference for a replay that observes another fingerprint",
				body: func(calls int, c *engine.Case) string {
					engine.Draw(c, wide, "n")
					c.Observe(uint64(calls))
					return "always"
				},
				want: engine.Divergence{What: engine.FingerprintDifference, Recorded: uint64(1), Replayed: uint64(2)},
			},
			{
				name: "returns a fingerprint difference for a replay that observes none",
				body: func(calls int, c *engine.Case) string {
					engine.Draw(c, wide, "n")
					if calls == 1 {
						c.Observe(9)
					}
					return "always"
				},
				want: engine.Divergence{What: engine.FingerprintDifference, Index: 0, Recorded: uint64(9)},
			},
			{
				name: "returns a verdict difference for a replay that fails another way",
				body: func(calls int, c *engine.Case) string {
					engine.Draw(c, wide, "n")
					if calls == 1 {
						return "first"
					}
					return "second"
				},
				want: engine.Divergence{
					What:     engine.VerdictDifference,
					Index:    1,
					Recorded: engine.Identity{Assertion: "first"},
					Replayed: engine.Identity{Assertion: "second"},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				calls := 0
				got, trace := traced(func(c *engine.Case) string {
					calls++
					return tt.body(calls, c)
				}, settled())
				assert.Equal(t, got.Outcome, engine.Flaky, "a flaky run")
				assert.Equal(t, *got.Divergence, tt.want, "the first difference")
				assert.Equal(t, got.Cases, 0, "the simplest case failed")
				assert.Length(t, trace, 2, "the simplest case and its replay")
				assert.Equal(t, got.Runs, 0, "nothing shrunk")
			})
		}

		t.Run("returns a verdict difference for a replay that passes", func(t *testing.T) {
			t.Parallel()
			failed := false
			got, trace := traced(func(c *engine.Case) string {
				if engine.Draw(c, wide, "value") > 1000 && !failed {
					failed = true
					return "once"
				}
				return ""
			}, settled())
			once := engine.Identity{Assertion: "once"}
			want := engine.Divergence{What: engine.VerdictDifference, Index: 1, Recorded: once}
			assert.Equal(t, got.Outcome, engine.Flaky, "a flaky run")
			assert.Equal(t, *got.Divergence, want, "the recorded failure against a pass")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{558560502},
				"random case 0 of seed 7, which the replay contradicted")
			assert.Equal(t, got.Cases, 1, "the simplest case passed")
			assert.Equal(t, digestOf(trace), "0b176f0efea58cb450463e6523c53e8678843e669718b88d234c7a6eb949201f",
				"the simplest case, random case 0 and its replay")
		})
	})
}
