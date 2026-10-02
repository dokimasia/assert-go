// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// forAllAllocs are the allocations of a passing run of 100 cases that draw
// one integer each, measured: the engine's 2,689 for the run's cases, and
// one for the adapter of the body to the engine's case.
const forAllAllocs = 2690

// The contract of the test properties, the label of their draws, and the
// identities of their failures, as the definition's behaviour vectors
// name them.
const (
	// contract is the contract of every test property.
	contract = "the property holds"
	// drawn is the label of a body's draw.
	drawn = "value"
	// big is the identity of a failure at a large value.
	big = "big"
	// every is the identity of a failure at every value.
	every = "every"
	// always is the identity of a failure of every case.
	always = "always"
)

// The assertion of a failing run's record and the names of its detail
// fields, which the definition pins.
const (
	forAllID            = "prop-for-all"
	outcomeField        = "outcome"
	casesField          = "cases"
	rejectedField       = "rejected"
	seedField           = "seed"
	counterexampleField = "counterexample"
	failureField        = "failure"
	choicesField        = "choices"
	othersField         = "others"
	divergenceField     = "divergence"
	coverageField       = "coverage"
)

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
				counterexampleField: []prop.Drawn{
					{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000},
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
			want := []prop.Drawn{{
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
	})
}

// TestForAllZeroAlloc checks the allocation ceiling of a passing run.
func TestForAllZeroAlloc(t *testing.T) {
	rec := assert.NewRecorder()
	body := draws(prop.Integer(0, 1000))
	assert.MaxAllocs(t, func() { prop.ForAll(rec, contract, body, prop.Seed(7)) }, forAllAllocs,
		"a run allocates for each case")
}

// BenchmarkForAll measures a passing run of 100 cases.
func BenchmarkForAll(b *testing.B) {
	b.Run("ForAll", func(b *testing.B) {
		rec := assert.NewRecorder()
		body := draws(prop.Integer(0, 1000))
		c := bench.Start(b).MaxAllocs(forAllAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, body, prop.Seed(7))
		}
		assert.False(b, rec.Failed(), "every run passes")
	})
}

// detailOf runs body as the property contract on a recorder under opts, and
// returns the detail of the one record that the run reported, or nil for a
// run that reported none.
func detailOf(body func(*prop.Case), opts ...prop.Option) map[string]any {
	rec := assert.NewRecorder()
	prop.ForAll(rec, contract, body, opts...)
	records := rec.Failures()
	if len(records) == 0 {
		return nil
	}
	return records[0].Detail
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

// counts returns the outcome and the counts of valid and rejected cases of
// a record's detail.
func counts(detail map[string]any) []any {
	return []any{detail[outcomeField], detail[casesField], detail[rejectedField]}
}

// draws returns the body that draws one value of g and never fails.
func draws[T any](g prop.Generator[T]) func(*prop.Case) {
	return func(c *prop.Case) { c.Draw(g, drawn) }
}

// failsAtLeast returns the body that draws an integer in [0, most], and
// fails with identity at a value of least or more.
func failsAtLeast(most, least int, identity string) func(*prop.Case) {
	g := prop.Integer(0, most)
	return func(c *prop.Case) {
		if c.Draw(g, drawn) >= least {
			fail(c, identity)
		}
	}
}

// fail ends the case with an aborting record of identity, without a
// location, as a body of the definition's vectors fails.
func fail(c *prop.Case, identity string) {
	c.Report(assert.Failure{Assertion: identity}, true)
}
