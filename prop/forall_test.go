// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/prop"
)

// forAllAllocs is the ceiling of the allocations of a passing run of 100
// cases that draw one integer each: the engine's allocations for the run's
// cases, and two for the closures that adapt the body to the engine's case
// and give each case a context.
const forAllAllocs = 620

// forAllID is the assertion of a failing run's record, which the
// definition pins.
const forAllID = "prop-for-all"

// TestForAll checks the outcome of each kind of run, pinned to the
// definition's behaviour vectors, and the record that a failing run
// reports.
func TestForAll(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("reports no record for a body that never fails", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, draws(prop.Integer(0, 1000)), prop.Seed(7))
			assert.False(t, rec.Failed(), "the run passes")
		})

		t.Run("reports the minimal counterexample with every detail field", func(t *testing.T) {
			t.Parallel()
			want := map[string]any{
				outcomeField:  prop.Counterexample,
				casesField:    1,
				rejectedField: 0,
				seedField:     "7",
				counterexampleField: []prop.Entry{
					prop.Drawn{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000},
				},
				failureField:    assert.Failure{Assertion: big},
				choicesField:    "prop1:AOkH",
				othersField:     []prop.Other{},
				divergenceField: nil,
				coverageField:   nil,
			}
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7))
			assert.Equal(t, got, want, "the record of the definition's vector")
		})

		t.Run("reports one aborting record of prop-for-all at the line of its call", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			var at assert.Where
			prop.ForAll(rec, contract, failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Store(here(&at)))
			records := rec.Failures()
			assert.Length(t, records, 1, "one record")
			assert.Equal(t, records[0].Assertion, forAllID, "the assertion")
			assert.Equal(t, records[0].Contract, contract, "the contract")
			assert.Equal(t, records[0].Where, at, "the call of ForAll")
			assert.Empty(t, rec.Messages(), "no recording failure")
		})

		t.Run("fails a run that rejects every case", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				c.Draw(prop.Integer(0, 1000), drawn)
				c.Assume(false)
			}
			got := detailOf(body, prop.Seed(7))
			assert.Equal(t, counts(got), []any{prop.Rejected, 0, 458}, "the outcome and the counts of the vector")
			assert.Nil(t, got[counterexampleField], "no counterexample")
		})

		t.Run("passes a run that rejects four cases in five", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) { c.Assume(c.Draw(prop.Integer(0, 1000), drawn)%5 == 0) }
			assert.Nil(t, detailOf(body, prop.Seed(7)), "no record")
		})

		t.Run("fails a body that requests no input", func(t *testing.T) {
			t.Parallel()
			got := detailOf(func(*prop.Case) {}, prop.Seed(7))
			assert.Equal(t, counts(got), []any{prop.Vacuous, 1, 0}, "one case, which requested nothing")
		})

		t.Run("passes once every input of a small domain is tested", func(t *testing.T) {
			t.Parallel()
			small := prop.OneOf(prop.Boolean().Map(func(b bool) any { return b }), prop.SampledFrom[any]("a", "b", "c"))
			rec := assert.NewRecorder()
			assert.Equal(t, completed(rec, small, prop.Seed(7)), 5, "the five inputs, each once")
			assert.False(t, rec.Failed(), "the run passes")
		})

		t.Run("tests each input of a domain once", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			assert.Equal(t, completed(rec, prop.Integer(0, 30), prop.Seed(7)), 31, "the 31 integers, each once")
			assert.False(t, rec.Failed(), "the run passes")
		})

		t.Run("finds a failure at the maximum in an edge case", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				if c.Draw(prop.Integer[uint64](0, math.MaxUint64), drawn) == math.MaxUint64 {
					fail(c, "largest")
				}
			}
			got := detailOf(body, prop.Seed(7))
			want := []prop.Entry{prop.Drawn{
				Label:          drawn,
				Value:          uint64(math.MaxUint64),
				Relevance:      prop.ValueMatters,
				NearestPassing: uint64(math.MaxUint64 - 1),
			}}
			assert.Equal(t, got[casesField], any(3), "the simplest case and random cases 0 and 1")
			assert.Equal(t, got[counterexampleField], any(want), "the maximum and the value below it")
			assert.Equal(t, got[choicesField], any("prop1:AP___________wE"), "the token of the vector")
		})

		t.Run("finds a failure in the prefix case of random case 0", func(t *testing.T) {
			t.Parallel()
			pair := prop.List(prop.Integer(0, 1000000000), prop.MinSize(2), prop.MaxSize(2))
			body := func(c *prop.Case) {
				if slices.Equal(c.Draw(pair, drawn), []int{558560502, 0}) {
					fail(c, "prefix")
				}
			}
			got := detailOf(body, prop.Seed(7), prop.Shrink(0))
			assert.Equal(t, got[casesField], any(2), "the simplest case and random case 0")
			assert.Equal(t, got[choicesField], any("prop1:AAEA9umrigIAAQAAAAA"), "the token of the vector")
		})

		t.Run("records a passing run with its detail, and the calls of each case under it", func(t *testing.T) {
			t.Parallel()
			digit := prop.Integer(0, 9)
			calls := callsOf(t, func(c *prop.Case) {
				assert.True(c, c.Draw(digit, drawn) < 10, "a digit")
			}, prop.Seed(7))
			run := calls[0]
			assert.Equal(t, []any{run["seq"], run["assertion"], run["contract"], run["verdict"], run["aborting"]},
				[]any{1.0, forAllID, contract, "pass", true}, "the property's call first")
			assert.Equal(t, run["detail"], any(map[string]any{
				outcomeField: "passed", casesField: 10.0, rejectedField: 0.0, seedField: "7",
				counterexampleField: nil, failureField: nil, choicesField: nil, othersField: nil,
				divergenceField: nil, coverageField: nil,
			}), "the detail of the run, every field that a pass does not use null")
			assert.Length(t, calls, 11, "the property and one call for each digit")
			for i, call := range calls[1:] {
				assert.Equal(
					t,
					[]any{call["seq"], call["parent"], call["assertion"]},
					[]any{float64(i + 2), 1.0, "true"},
					"a call of a case under the property",
				)
			}
			assert.Equal(t, calls[1]["phase"], any("simplest"), "the simplest case first")
		})

		t.Run("records a failing run with the detail of its counterexample", func(t *testing.T) {
			t.Parallel()
			detail := recordedDetail(t, failsAtLeast(10000, 1001, big), prop.Seed(7))
			assert.Equal(t, detail[outcomeField], any("counterexample"), "the outcome")
			assert.Equal(t, detail[counterexampleField], any([]any{map[string]any{
				"label": drawn, "value": map[string]any{"type": "int", "value": 1001.0},
				"any-value-fails": false, "nearest-passing": map[string]any{"type": "int", "value": 1000.0},
			}}), "the draw as a typed literal, with what the explain phase found")
			assert.Equal(
				t,
				detail[failureField],
				any(map[string]any{"assertion": big, "contract": "", "detail": map[string]any{}}),
				"the failure record of the minimal case",
			)
			assert.Equal(t, detail[choicesField], any("prop1:AOkH"), "the token")
			assert.Equal(t, detail[othersField], any([]any{}), "no other failure")
		})

		t.Run("ends the call with a fault for a token that no encoder writes", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, draws(prop.Integer(0, 9)), prop.Replay("nonsense"))
			calls := decodedCalls(t, rec.Records())
			assert.Length(t, calls, 1, "the property's call alone")
			assert.Equal(t, calls[0]["verdict"], any("error"), "a call that ended without a verdict")
			refused := &fault.Error{
				Op:     forAllOp,
				Path:   fault.Path{fault.Field("Replay")},
				Kind:   token.ErrInvalid,
				Reason: `"nonsense" does not start with prop1:`,
			}
			assert.Equal(t, calls[0]["error"], any(matcher.RenderFault(refused)), "the writer's text of the fault")
			assert.Empty(t, rec.Failures(), "no failure record")
		})

		t.Run("records on four workers the calls that one worker records", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				assert.True(c, c.Draw(prop.Integer(0, 30), drawn) < 25, "below 25")
			}
			assert.Equal(t, callsOf(t, body, prop.Seed(7), prop.Workers(4)), callsOf(t, body, prop.Seed(7)),
				"the calls of the cases of one worker, in its order")
		})
	})
}

// TestForAllAllocs checks the allocation ceiling of a passing run.
func TestForAllAllocs(t *testing.T) {
	rec := &matchertest.Seat{}
	body := draws(prop.Integer(0, 1000))
	assert.MaxAllocs(t, func() { prop.ForAll(rec, contract, body, prop.Seed(7)) }, forAllAllocs,
		"a run allocates for each case")
}

// BenchmarkForAll measures a passing run of 100 cases.
func BenchmarkForAll(b *testing.B) {
	b.Run("ForAll", func(b *testing.B) {
		rec := &matchertest.Seat{}
		body := draws(prop.Integer(0, 1000))
		c := bench.Start(b).MaxAllocs(forAllAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, body, prop.Seed(7))
		}
		assert.False(b, rec.Failed(), "every run passes")
	})
}

// completed runs the property contract on rec under opts with a body that
// draws one value of g and passes, and returns the number of cases whose
// body ran past the draw: the valid cases, since a case that repeats a
// tested input ends at its draw.
func completed[T any](rec *assert.Recorder, g prop.Generator[T], opts ...prop.Option) int {
	var calls int
	prop.ForAll(rec, contract, func(c *prop.Case) {
		c.Draw(g, drawn)
		calls++
	}, opts...)
	return calls
}
