// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

// TestResult checks the result that a run hands to the function of
// OnResult: the fields of the detail of the run, for each outcome.
func TestResult(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("hands the outcome, the counts and the seed of a passing run", func(t *testing.T) {
			t.Parallel()
			got := resultOf(t, draws(prop.Integer(0, 9)), prop.Seed(math.MaxUint64))
			want := prop.Result{Outcome: prop.Passed, Cases: 10, Seed: math.MaxUint64}
			assert.Equal(t, got, want, "the result of a run over the ten digits, every other field empty")
		})

		t.Run("hands the fields of the detail of a counterexample", func(t *testing.T) {
			t.Parallel()
			want := prop.Result{
				Outcome: prop.Counterexample,
				Cases:   1,
				Seed:    7,
				Counterexample: []prop.Entry{
					prop.Drawn{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000},
				},
				Failure: &assert.Failure{Assertion: big},
				Choices: "prop1:AOkH",
				Others:  []prop.Other{},
			}
			got := resultOf(t, failsAtLeast(10000, 1001, big), prop.Seed(7))
			assert.Equal(t, got, want, "the result of the definition's vector")
		})

		t.Run("hands the values of the failure record of the run", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			var got prop.Result
			prop.ForAll(rec, contract, failsAtLeast(10000, 1001, big), prop.Seed(7),
				prop.OnResult(func(r prop.Result) { got = r }))
			assert.NotNil(t, got.Failure, "the result of a counterexample has a failure")
			assert.Equal(t, map[string]any{
				outcomeField:        got.Outcome,
				casesField:          got.Cases,
				rejectedField:       got.Rejected,
				seedField:           strconv.FormatUint(got.Seed, 10),
				counterexampleField: got.Counterexample,
				failureField:        *got.Failure,
				choicesField:        got.Choices,
				othersField:         got.Others,
				divergenceField:     nil,
				coverageField:       nil,
				reasonField:         nil,
			}, rec.Failures()[0].Detail, "the detail of the failure record")
		})

		t.Run("hands the counts of a run that rejects every case", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				c.Draw(prop.Integer(0, 1000), drawn)
				c.Assume(false)
			}
			want := prop.Result{Outcome: prop.Rejected, Rejected: 458, Seed: 7}
			assert.Equal(t, resultOf(t, body, prop.Seed(7)), want, "the counts of the definition's vector")
		})

		t.Run("hands the reason of a skipped run", func(t *testing.T) {
			t.Parallel()
			got := resultOf(t, func(c *prop.Case) {
				c.Draw(prop.Integer(0, 1000), drawn)
				c.Skipf("unsupported")
			}, prop.Seed(7))
			want := prop.Result{Outcome: prop.Skipped, Seed: 7, Reason: "unsupported"}
			assert.Equal(t, got, want, "the reason of the skip")
		})

		t.Run("hands the divergence of a flaky run", func(t *testing.T) {
			t.Parallel()
			got := resultOf(t, once(func(c *prop.Case) { fail(c, "once") }), prop.Seed(7))
			want := &prop.Divergence{What: prop.VerdictDifference, Recorded: "once"}
			assert.Equal(t, got.Divergence, want, "the verdict that the replay did not repeat")
		})

		t.Run("hands the requirement that a run missed", func(t *testing.T) {
			t.Parallel()
			got := resultOf(t, classifies(1000, even, isEven), prop.Seed(7), prop.Require(even, 0.9))
			want := &prop.Shortfall{Label: even, Share: 0.9, Counted: 45, Valid: 100, Verdict: prop.Refuted}
			assert.Equal(t, got.Coverage, want, "the shortfall of the definition's vector")
		})
	})
}

// resultOf runs body as the property contract on a recorder under opts, and
// returns the one result that the run hands to the function of OnResult.
func resultOf(t *testing.T, body func(*prop.Case), opts ...prop.Option) prop.Result {
	t.Helper()
	var got []prop.Result
	prop.ForAll(assert.NewRecorder(), contract, body,
		append(opts, prop.OnResult(func(r prop.Result) { got = append(got, r) }))...)
	assert.Length(t, got, 1, "one result")
	return got[0]
}
