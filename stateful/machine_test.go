// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// weightCases is the number of random cases whose steps the test of a
// Weight of 0 compares with those of a Weight of 1.
const weightCases = 20

// TestMachine checks the contract of a machine and its actions that Steps
// enforces: the machines that state no machine, the default weight, and
// the spec that checks nothing.
func TestMachine(t *testing.T) {
	t.Parallel()

	t.Run("Steps", func(t *testing.T) {
		t.Parallel()

		run := func(*prop.Case, int, any) {}
		tests := []struct {
			name string
			give []stateful.Action[int]
			want string
		}{
			{
				name: "panics for an action with a negative weight",
				give: []stateful.Action[int]{{Name: put, Weight: -1, Run: run}},
				want: `stateful: Steps of a machine whose action "put" has weight -1`,
			},
			{
				name: "panics for an action without Run",
				give: []stateful.Action[int]{{Name: put, Run: run}, {Name: get}},
				want: `stateful: Steps of a machine whose action "get" has no Run`,
			},
			{
				name: "panics for two actions with one name",
				give: []stateful.Action[int]{{Name: put, Run: run}, {Name: get, Run: run}, {Name: put, Run: run}},
				want: `stateful: Steps of a machine whose actions name "put" twice`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e := replayed(func(c *prop.Case) { stateful.Steps(c, stateful.Machine[int]{Actions: tt.give}) })
				assert.Equal(t, []any{e.Status, e.Panic}, []any{engine.CaseFailed, any(tt.want)},
					"the panic fails the case")
			})
		}

		t.Run("takes a Weight of 0 as 1", func(t *testing.T) {
			t.Parallel()
			weighted := func(first int) func(*prop.Case) {
				return func(c *prop.Case) {
					stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{
						{Name: put, Weight: first, Run: run}, {Name: get, Weight: 2, Run: run},
					}})
				}
			}
			for index := range uint64(weightCases) {
				zero := engine.Generate(bodyOf(weighted(0)), seed, index, nil)
				one := engine.Generate(bodyOf(weighted(1)), seed, index, nil)
				assert.Equal(t, stepsOf(zero), stepsOf(one), "the steps of the case")
			}
		})

		t.Run("checks nothing for a spec without Initial and Next, and passes the zero state", func(t *testing.T) {
			t.Parallel()
			var states []int
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{
					Actions:   []stateful.Action[int]{incrementOf(new(int), 1)},
					Invariant: func(_ *prop.Case, state int) { states = append(states, state) },
				})
			}, 1, 1, 0, 0)
			assert.Equal(t, e.Status, engine.CasePassed, "no check of the increment that skips a count")
			assert.Equal(t, states, []int{0, 0, 0}, "the zero state after setup, the step and the settle")
		})

		t.Run("checks the history against a spec that states Next alone, which states no spec", func(t *testing.T) {
			t.Parallel()
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{
					Spec:    history.Spec[int]{Next: counter.Next},
					Actions: []stateful.Action[int]{incrementOf(new(int), 0)},
				})
			})
			refused := fault.In("history.Linearizable", fault.New("the spec states no Initial or no Next"))
			want := []assert.Failure{{Contract: matcher.RenderFault(refused), Where: e.Case.Failures()[0].Where}}
			assert.Equal(t, e.Case.Failures(), want, "the fault of the check, as a record without an assertion")
		})
	})
}
