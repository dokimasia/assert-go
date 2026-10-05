// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// TestCheck checks the check of the history after a step: the record of a
// check that fails, the one partition of every call, and the call records
// that the checks leave.
func TestCheck(t *testing.T) {
	t.Parallel()

	t.Run("Steps", func(t *testing.T) {
		t.Parallel()

		t.Run("fails the case with the record of linearizable at Steps for a rejected step", func(t *testing.T) {
			t.Parallel()
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Model: counter, Actions: []stateful.Action[int]{
					incrementOf(new(int), 1),
				}})
			}, 1, 1, 0, 0)
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
			f := e.Case.Failures()[0]
			assert.Equal(t, []any{f.Assertion, f.Contract, filepath.Base(f.Where.File)},
				[]any{"linearizable", "the history of the machine's steps is linearizable", "check_test.go"},
				"the record of the check, at the call of Steps")
		})

		t.Run("checks every call in one partition, whatever keys the calls declare", func(t *testing.T) {
			t.Parallel()
			count := 0
			keyed := func(key string) stateful.Action[int] {
				return stateful.Action[int]{Name: key, Run: func(c *prop.Case, client int, _ any) {
					call := c.History().Invoke(client, "increment", nil, key)
					count++
					call.OK(count)
				}}
			}
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Model: counter, Actions: []stateful.Action[int]{
					keyed("a"), keyed("b"),
				}})
			}, 1, 1, 1, 0, 1, 1, 0)
			assert.Equal(t, e.Status, engine.CasePassed, "the increments of a and b count one counter")
		})

		t.Run("writes no call record of a passing check in a recorded run", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, "the counter counts", func(c *prop.Case) {
				count := 0
				stateful.Steps(c, stateful.Machine[int]{
					Model: counter, Actions: []stateful.Action[int]{incrementOf(&count, 0)},
				}, stateful.Max(3))
			}, prop.Seed(seed), prop.Cases(5))
			assert.False(t, rec.Failed(), "the run passes")
			assert.Length(t, rec.Records(), 1, "the call record of the property alone")
		})
	})
}
