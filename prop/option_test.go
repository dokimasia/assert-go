// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/prop"
)

// optionAllocs are the allocations of an option that its caller keeps,
// measured: the closure of its setting.
const optionAllocs = 1

// never is a label that no case counts.
const never = "never"

// TestOption checks each option of a run.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("Cases", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the number of valid cases a run aims for", func(t *testing.T) {
			t.Parallel()
			got := detailOf(draws(prop.Integer(0, 1000000000)), prop.Seed(7), prop.Cases(10), prop.Require(never, 1))
			assert.Equal(t, counts(got), []any{prop.CoverageUnmet, 10, 0}, "the first check after ten valid cases")
		})

		t.Run("aims for 100 valid cases by default", func(t *testing.T) {
			t.Parallel()
			got := detailOf(draws(prop.Integer(0, 1000000000)), prop.Seed(7), prop.Require(never, 1))
			assert.Equal(t, counts(got), []any{prop.CoverageUnmet, 100, 0}, "the first check after 100 valid cases")
		})

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.Cases(0) }, "a run of no case")
			assert.Equal(t, got, any("prop: Cases(0) is below 1"), "the panic names the option")
		})

		t.Run("returns the option of one case", func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { prop.Cases(1) }, "a run of one case")
		})
	})

	t.Run("Seed", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the seed that the record states", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(math.MaxUint64))
			assert.Equal(t, got[seedField], any("18446744073709551615"), "the largest seed in decimal")
		})
	})

	t.Run("Replay", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the case of a failing token and nothing else", func(t *testing.T) {
			t.Parallel()
			want := map[string]any{
				outcomeField:        prop.Counterexample,
				casesField:          0,
				rejectedField:       0,
				seedField:           "7",
				counterexampleField: []prop.Drawn{{Label: drawn, Value: 7}},
				failureField:        assert.Failure{Assertion: big},
				choicesField:        "prop1:AAc",
				othersField:         []prop.Other{},
				divergenceField:     nil,
				coverageField:       nil,
			}
			got := detailOf(failsAtLeast(9, 5, big), prop.Seed(7), prop.Replay("prop1:AAc"))
			assert.Equal(t, got, want, "the record of the definition's vector")
		})

		t.Run("passes the case of a passing token", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, detailOf(failsAtLeast(9, 5, big), prop.Seed(7), prop.Replay("prop1:AAM")), "no record")
		})

		t.Run("fails the run at once for a token that no encoder writes", func(t *testing.T) {
			t.Parallel()
			var calls int
			seat := &matchertest.Seat{}
			prop.ForAll(seat, contract, func(*prop.Case) { calls++ }, prop.Replay("prop2:AAc"))
			expectOnlyFault(t, seat.Faults(), fault.Error{
				Op:     forAllOp,
				Path:   fault.Path{fault.Field("Replay")},
				Kind:   token.ErrInvalid,
				Reason: `"prop2:AAc" does not start with prop1:`,
			})
			assert.Equal(t, calls, 0, "no case runs")
		})
	})

	t.Run("Require", func(t *testing.T) {
		t.Parallel()

		t.Run("adds each requirement in order", func(t *testing.T) {
			t.Parallel()
			got := detailOf(classifies(1000, even, isEven), prop.Seed(7),
				prop.Require(even, 0.3), prop.Require(never, 1))
			want := &prop.Shortfall{Label: never, Share: 1, Valid: 100, Verdict: prop.Refuted}
			assert.Equal(t, got[coverageField], any(want), "the second requirement, which no case meets")
		})

		tests := []struct {
			name string
			give float64
			want string
		}{
			{
				name: "panics for a share above 1",
				give: 1.5,
				want: `prop: Require("even", 1.5) states a share outside [0, 1]`,
			},
			{
				name: "panics for a share below 0",
				give: -0.5,
				want: `prop: Require("even", -0.5) states a share outside [0, 1]`,
			},
			{
				name: "panics for a share of NaN",
				give: math.NaN(),
				want: `prop: Require("even", NaN) states a share outside [0, 1]`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := assert.Panics(t, func() { prop.Require(even, tt.give) }, "a share outside [0, 1]")
				assert.Equal(t, got, any(tt.want), "the panic names the requirement")
			})
		}

		t.Run("returns a requirement of a share of 0 or 1", func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { prop.Require(even, 0) }, "a share of 0")
			assert.NotPanics(t, func() { prop.Require(even, 1) }, "a share of 1")
		})
	})

	t.Run("Shrink", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the first failing case as found for a budget of 0", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Shrink(0))
			assert.Equal(t, got[counterexampleField], any([]prop.Drawn{{Label: drawn, Value: 8522}}), "the found value")
			assert.Equal(t, got[choicesField], any("prop1:AMpC"), "the token of the definition's vector")
		})

		t.Run("panics for a negative budget", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.Shrink(-1) }, "a negative budget")
			assert.Equal(t, got, any("prop: Shrink(-1) is below 0"), "the panic names the option")
		})
	})

	t.Run("ShrinkTime", func(t *testing.T) {
		t.Parallel()

		t.Run("ends a shrink on the platform clock under a seat's controlled clock", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder().WithClock(assert.NewControlled(today))
			prop.ForAll(rec, contract, failsAtLeast(10000, 1001, big), prop.Seed(7), prop.ShrinkTime(time.Nanosecond))
			got := rec.Failures()[0].Detail[counterexampleField]
			assert.Equal(t, got, any([]prop.Drawn{{Label: drawn, Value: 8522}}), "the found value, unshrunk")
		})

		t.Run("sets no limit for a time of 0", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.ShrinkTime(0))
			assert.Equal(t, got[choicesField], any("prop1:AOkH"), "the minimal case")
		})

		t.Run("panics for a negative time", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.ShrinkTime(-time.Second) }, "a negative time")
			assert.Equal(t, got, any("prop: ShrinkTime(-1s) is below 0"), "the panic names the option")
		})
	})

	t.Run("MaxChoices", func(t *testing.T) {
		t.Parallel()

		t.Run("rejects a case that requests more choices", func(t *testing.T) {
			t.Parallel()
			long := func(c *prop.Case) {
				if len(c.Draw(prop.List(prop.Integer(0, 1000)), drawn)) >= 2 {
					fail(c, "long")
				}
			}
			assert.Nil(t, detailOf(long, prop.Seed(7), prop.MaxChoices(4)), "no list of two elements fits")
			assert.NotNil(t, detailOf(long, prop.Seed(7)), "a list of two elements fits the default cap")
		})

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.MaxChoices(0) }, "a cap of no choice")
			assert.Equal(t, got, any("prop: MaxChoices(0) is below 1"), "the panic names the option")
		})

		t.Run("returns the option of a cap of one choice", func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { prop.MaxChoices(1) }, "a cap of one choice")
		})
	})

	t.Run("Explain", func(t *testing.T) {
		t.Parallel()

		t.Run("explains the counterexample when enabled", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Explain(false), prop.Explain(true))
			want := []prop.Drawn{{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000}}
			assert.Equal(t, got[counterexampleField], any(want), "the later option")
		})
	})

	t.Run("Draws", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the case of the entries before any other case", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 9999, big), prop.Seed(7),
				prop.Draws(`[{"label": "value", "value": {"type": "int", "value": 9999}}]`))
			want := []prop.Drawn{{Label: drawn, Value: 9999, Relevance: prop.ValueMatters, NearestPassing: 9998}}
			assert.Equal(t, got[casesField], any(0), "no valid case ran before the failing one")
			assert.Equal(t, got[counterexampleField], any(want), "the entry's value, which shrinks no further")
		})

		t.Run("gives a draw past the last entry its target", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				if c.Draw(prop.Integer(0, 9), "first") == 4 {
					c.Draw(prop.Integer(5, 9), "second")
					fail(c, big)
				}
			}
			got := detailOf(body, prop.Seed(7), prop.Explain(false),
				prop.Draws(`[{"label": "first", "value": {"type": "int", "value": 4}}]`))
			want := []prop.Drawn{{Label: "first", Value: 4}, {Label: "second", Value: 5}}
			assert.Equal(t, got[counterexampleField], any(want), "the entry's value, then the target")
			assert.Equal(t, got[casesField], any(0), "the case of the entries fails first")
		})

		t.Run("runs back a value that Of derives from the record of its shape", func(t *testing.T) {
			t.Parallel()
			body := func(c *prop.Case) {
				if len(c.Draw(prop.Of[order](), drawn).Lines) == 2 {
					fail(c, big)
				}
			}
			entries := `[{"label": "value", "value": {"type": "record", "fields": [` +
				`["id", {"type": "int", "value": 7}],` +
				`["lines", {"type": "list", "items": [` +
				`{"type": "record", "fields": [["sku", {"type": "string", "value": "a"}], ["qty", {"type": "int", "value": 1}]]},` +
				`{"type": "record", "fields": [["sku", {"type": "string", "value": "b"}], ["qty", {"type": "int", "value": 2}]]}]}],` +
				`["note", {"type": "null"}]]}}]`
			got := detailOf(body, prop.Seed(7), prop.Shrink(0), prop.Draws(entries))
			want := order{ID: 7, Lines: []line{{SKU: "a", Qty: 1}, {SKU: "b", Qty: 2}}}
			assert.Equal(t, got[counterexampleField], any([]prop.Drawn{{Label: drawn, Value: want}}),
				"the order of the entry, found by the case of the entries")
		})

		tests := []struct {
			name string
			give string
			want fault.Error
		}{
			{
				name: "fails the test at a draw whose label differs from its entry's",
				give: `[{"label": "other", "value": {"type": "int", "value": 4}}]`,
				want: fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field("Draws"), fault.Index(0), fault.Field("label")},
					Reason: `the draw labelled "value" takes the entry labelled "other"`,
				},
			},
			{
				name: "fails the test at a draw whose generator does not produce its entry's value",
				give: `[{"label": "value", "value": {"type": "int", "value": 12}}]`,
				want: fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field("Draws"), fault.Index(0), fault.Field("value")},
					Kind:   engine.ErrCannotInvert,
					Reason: "12 is outside [0, 9]",
				},
			},
			{
				name: "fails the run at once for entries that are no array",
				give: `{`,
				want: fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field("Draws")},
					Reason: "the entries are no JSON array of objects",
				},
			},
			{
				name: "fails the run at once for an entry without a label",
				give: `[{"value": {"type": "int", "value": 4}}]`,
				want: fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field("Draws"), fault.Index(0)},
					Reason: "the entry states no label or no value",
				},
			},
			{
				name: "fails the run at once at an entry without a value",
				give: `[{"label": "value", "value": {"type": "int", "value": 4}}, {"label": "value"}]`,
				want: fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field("Draws"), fault.Index(1)},
					Reason: "the entry states no label or no value",
				},
			},
			{
				name: "fails the run at once at the value of an entry that is no typed literal",
				give: `[{"label": "value", "value": {"type": "int", "value": 4}},` +
					` {"label": "value", "value": {"type": "widget"}}]`,
				want: fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field("Draws"), fault.Index(1), fault.Field("value"), fault.Field("type")},
					Kind:   literal.ErrUnknownType,
					Reason: `the type "widget" is no type of the encoding`,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				seat := &matchertest.Seat{}
				prop.ForAll(seat, contract, failsAtLeast(9, 5, big), prop.Seed(7), prop.Draws(tt.give))
				expectOnlyFault(t, seat.Faults(), tt.want)
				assert.Empty(t, seat.Records(), "no record of prop-for-all")
			})
		}
	})

	t.Run("Workers", func(t *testing.T) {
		t.Parallel()

		t.Run("reports on four workers what a run on one reports", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Workers(4))
			assert.Equal(t, got, detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7)), "the record of one worker")
		})

		t.Run("enters every filter attempt into the tree on four workers", func(t *testing.T) {
			t.Parallel()
			truth := draws(prop.Boolean().Filter(func(b bool) bool { return b }))
			got := detailOf(truth, prop.Seed(7), prop.Workers(4), prop.Require(never, 1))
			assert.Equal(t, counts(got), []any{prop.CoverageUnmet, 3, 1}, "the counts of the vector's run")
			assert.Equal(t, got, detailOf(truth, prop.Seed(7), prop.Require(never, 1)), "the record of one worker")
		})

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.Workers(0) }, "no worker")
			assert.Equal(t, got, any("prop: Workers(0) is below 1"), "the panic names the option")
		})

		t.Run("returns the option of one worker", func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { prop.Workers(1) }, "one worker")
		})
	})

	t.Run("Option", func(t *testing.T) {
		t.Parallel()

		t.Run("changes nothing as the zero option", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Option{})
			assert.Equal(t, got[choicesField], any("prop1:AOkH"), "the minimal case of the defaults")
		})
	})
}

// TestOptionAllocs checks the allocation ceilings of the options.
func TestOptionAllocs(t *testing.T) {
	var kept prop.Option
	assert.MaxAllocs(t, func() { kept = prop.Cases(10) }, optionAllocs, "Cases allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Seed(7) }, optionAllocs, "Seed allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Replay("prop1:AAc") }, optionAllocs, "Replay allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Require(even, 0.5) }, optionAllocs, "Require allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Shrink(10) }, optionAllocs, "Shrink allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.ShrinkTime(time.Second) }, optionAllocs,
		"ShrinkTime allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.MaxChoices(10) }, optionAllocs, "MaxChoices allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Store("") }, optionAllocs, "Store allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Explain(false) }, optionAllocs, "Explain allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Workers(2) }, optionAllocs, "Workers allocates its setting")
	assert.MaxAllocs(t, func() { kept = prop.Draws("[]") }, optionAllocs, "Draws allocates its setting")
	assert.NotEqual(t, kept, prop.Option{}, "the kept option states a setting")
}

// BenchmarkOption measures each option that a caller keeps.
func BenchmarkOption(b *testing.B) {
	tests := []struct {
		name   string
		option func() prop.Option
	}{
		{name: "Cases", option: func() prop.Option { return prop.Cases(10) }},
		{name: "Seed", option: func() prop.Option { return prop.Seed(7) }},
		{name: "Replay", option: func() prop.Option { return prop.Replay("prop1:AAc") }},
		{name: "Require", option: func() prop.Option { return prop.Require(even, 0.5) }},
		{name: "Shrink", option: func() prop.Option { return prop.Shrink(10) }},
		{name: "ShrinkTime", option: func() prop.Option { return prop.ShrinkTime(time.Second) }},
		{name: "MaxChoices", option: func() prop.Option { return prop.MaxChoices(10) }},
		{name: "Store", option: func() prop.Option { return prop.Store("") }},
		{name: "Explain", option: func() prop.Option { return prop.Explain(false) }},
		{name: "Workers", option: func() prop.Option { return prop.Workers(2) }},
		{name: "Draws", option: func() prop.Option { return prop.Draws("[]") }},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var got prop.Option
			c := bench.Start(b).MaxAllocs(optionAllocs)
			defer c.End()
			for c.Loop() {
				got = tt.option()
			}
			assert.NotEqual(b, got, prop.Option{}, "the option states a setting")
		})
	}
}
