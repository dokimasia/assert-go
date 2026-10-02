// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

// The detail fields of the record of a case whose body panicked.
const (
	// panicField is the type of the panic's value.
	panicField = "panic"
	// stackField is the stack of the body's goroutine at the panic.
	stackField = "stack"
)

// TestDetail checks the detail fields of a failing run's record: which
// fields each outcome sets, and the failure of a case that panicked.
func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("sets every field but the counts and the seed to nil for a rejected run", func(t *testing.T) {
			t.Parallel()
			want := map[string]any{
				outcomeField:        prop.Rejected,
				casesField:          0,
				rejectedField:       458,
				seedField:           "7",
				counterexampleField: nil,
				failureField:        nil,
				choicesField:        nil,
				othersField:         nil,
				divergenceField:     nil,
				coverageField:       nil,
			}
			body := func(c *prop.Case) {
				c.Draw(prop.Integer(0, 1000), drawn)
				c.Assume(false)
			}
			assert.Equal(t, detailOf(body, prop.Seed(7)), want, "the record of the definition's vector")
		})

		t.Run("reports a panic as a record without an assertion at the panic's frame", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			body := func(*prop.Case) { panic(errors.New("boom" + here(&at))) }
			got, ok := detailOf(body, prop.Seed(7))[failureField].(assert.Failure)
			assert.True(t, ok, "a failure record")
			assert.Equal(t, got.Assertion, "", "no assertion")
			assert.Equal(t, got.Contract, "boom", "the panic's value as the contract")
			assert.Equal(t, got.Where, at, "the frame that panicked")
			assert.Equal(t, got.Detail[panicField], any("*errors.errorString"), "the value's type")
			assert.Contains(t, got.Detail[stackField], "prop_test.TestDetail", "the stack of the body's goroutine")
		})

		t.Run("reports the first record of a case that fails twice", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				c.Report(assert.Failure{Assertion: "first"}, false)
				fail(c, "second")
			}
			got := detailOf(body, prop.Seed(7))
			assert.Equal(t, got[failureField], any(assert.Failure{Assertion: "first"}), "the first record")
		})
	})
}
