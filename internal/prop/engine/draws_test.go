// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"slices"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocation ceilings of a run whose case of the entries ends at a
// refusal of the next entry.
const (
	// refusedRunAllocs is the ceiling of the allocations of a run whose
	// case of twoSteps refuses the first entry.
	refusedRunAllocs = 14
	// servedRunAllocs is the ceiling of the allocations of such a run whose
	// case serves one value first: those of a refused run, and the queue of
	// the value.
	servedRunAllocs = 15
)

// refusal is the reason of the refusals that the specs of the trace make.
const refusal = `"get" is not among the actions that the step lists`

// The step entries of the specs of the trace.
var (
	// putStep is a step of the action put outside a concurrent section.
	putStep = &engine.MachineStep{Action: "put", Client: -1}
	// getStep is a step of the action get on client 1.
	getStep = &engine.MachineStep{Action: "get", Client: 1}
	// twoSteps are the entries of a case of the two steps, put first.
	twoSteps = []engine.Entry{{Step: putStep}, {Step: getStep}}
)

// refusedRun are the settings of a run whose case of twoSteps ends at a
// refusal in each body of the trace's ceilings.
var refusedRun = engine.Settings{
	Seed:       referenceSeed,
	Cases:      engine.DefaultCases,
	MaxChoices: engine.MaxChoices,
	Draws:      twoSteps,
}

// call is what one call of a body drew, and the choices it made.
type call struct {
	// values are the values the call drew, in order.
	values []any
	// choices are the choices the call made.
	choices []choice.Choice
}

// TestDraws checks the case of a run's Draws entries, the examples a run
// tries after it, the refusal of an entry, and the trace that a machine
// follows in the case, with the draws and the entries of the definition's
// vectors.
func TestDraws(t *testing.T) {
	t.Parallel()

	count, name := engine.Integer(0, 9), engine.String(sizes(t, 0, 3))
	countAndName := func(c *engine.Case) []any {
		return []any{engine.Draw(c, count, "count"), engine.Draw(c, name, "name")}
	}

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the case of the entries first, each draw decoding its entry's value", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}, {Label: "name", Value: "ab"}}
			_, calls := drawsRun(countAndName, s)
			assert.Equal(t, calls[0].values, []any{4, "ab"}, "the values of the entries")
			assert.True(t, sameChoices(calls[0].choices, []choice.Choice{integers(4)[0], sequence(10, 11)}),
				"the choices that decode to them")
		})

		t.Run("takes the target of each draw past the last entry", func(t *testing.T) {
			t.Parallel()
			named := engine.String(sizes(t, 1, 3))
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}}
			_, calls := drawsRun(func(c *engine.Case) []any {
				return []any{engine.Draw(c, count, "count"), engine.Draw(c, named, "name")}
			}, s)
			assert.Equal(t, calls[0].values, []any{4, "0"}, "the entry's value, then the simplest name")
		})

		t.Run("takes the target of a choice outside a draw", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}}
			_, calls := drawsRun(func(c *engine.Case) []any {
				return []any{c.Integer(digitRange).Magnitude(), engine.Draw(c, count, "count")}
			}, s)
			assert.Equal(t, calls[0].values, []any{uint64(0), 4}, "the target, then the entry's value")
		})

		t.Run("leaves an entry past the last draw unread", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}, {Label: "name", Value: "ab"}}
			got, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, count, "count")} }, s)
			assert.Equal(t, calls[0].values, []any{4}, "the first entry's value")
			assert.Equal(t, got.Outcome, engine.Passed, "the run passes")
		})

		t.Run("runs the examples after the case of the entries and before the stored cases", func(t *testing.T) {
			t.Parallel()
			s := settled(integers(4)...)
			s.Draws = []engine.Entry{{Label: "count", Value: 1}}
			s.Examples = []engine.Example{{Choices: integers(2)}, {Choices: integers(3)}}
			_, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, count, "count")} }, s)
			firsts := []any{calls[0].values[0], calls[1].values[0], calls[2].values[0], calls[3].values[0]}
			assert.Equal(t, firsts, []any{1, 2, 3, 4}, "the entries, the two examples, then the stored case")
		})

		failsFromFive := func(c *engine.Case) {
			if engine.Draw(c, count, "count") >= 5 {
				c.Report(assert.Failure{Assertion: "big"}, false)
			}
		}

		t.Run("shrinks a failing case of the entries", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 9}}
			got := engine.Run(failsFromFive, s)
			assert.Equal(t, got.Outcome, engine.Counterexample, "a counterexample")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{5}, "the smallest count that fails")
		})

		t.Run("shrinks a failing example", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Examples = []engine.Example{{Choices: integers(9)}}
			got := engine.Run(failsFromFive, s)
			assert.Equal(t, got.Outcome, engine.Counterexample, "a counterexample")
			assert.Equal(t, drawValues(got.Failing.Case.Draws()), []any{5}, "the smallest count that fails")
		})

		t.Run("refuses an entry whose label differs at the entry's label, before any other case", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "name", Value: "ab"}}
			got, calls := drawsRun(countAndName, s)
			assert.Equal(t, got, engine.Result{Refused: got.Refused}, "the refusal, and nothing else")
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, f.Path, fault.Path{fault.Index(0), fault.Field("label")}, "the first entry's label")
			assert.Equal(t, f.Reason, `the draw labelled "count" takes the entry labelled "name"`, "both labels")
			assert.ErrorIsNot(t, got.Refused, engine.ErrCannotInvert, "no inverse ran")
			assert.Length(t, calls, 1, "no other case runs")
		})

		refusals := []struct {
			name       string
			give       engine.Generator[int]
			wantReason string
		}{
			{
				name:       "refuses a value that the draw's generator does not produce at the entry's value",
				give:       count,
				wantReason: "12 is outside [0, 9]",
			},
			{
				name:       "refuses the entry of a draw whose generator has no inverse at the entry's value",
				give:       count.Map(func(v int) int { return v }),
				wantReason: "map has no inverse",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = []engine.Entry{{Label: "count", Value: 12}}
				got := engine.Run(func(c *engine.Case) { engine.Draw(c, tt.give, "count") }, s)
				assert.ErrorIs(t, got.Refused, engine.ErrCannotInvert, "no choices decode to the entry's value")
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(t, f.Path, fault.Path{fault.Index(0), fault.Field("value")}, "the first entry's value")
				assert.Equal(t, f.Reason, tt.wantReason, "why no choices decode to the value")
			})
		}

		t.Run("refuses a later entry at its index and at the part that no choice produces", func(t *testing.T) {
			t.Parallel()
			digits := engine.List(count, sizes(t, 0, 3))
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}, {Label: "digits", Value: []int{1, 2, 12}}}
			got := engine.Run(func(c *engine.Case) {
				engine.Draw(c, count, "count")
				engine.Draw(c, digits, "digits")
			}, s)
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			want := fault.Path{fault.Index(1), fault.Field("value"), fault.Index(2)}
			assert.Equal(t, f.Path, want, "the third element of the second entry's value")
			assert.Equal(t, f.Reason, "12 is outside [0, 9]", "why no choices decode to the element")
		})

		pair := engine.Composite(func(c *engine.Case) []int {
			return []int{engine.Draw(c, count, "a"), engine.Draw(c, count, "b")}
		})
		drawsPair := func(c *engine.Case) []any { return []any{engine.Draw(c, pair, "pair")} }

		t.Run("replays a composite after the entries of its draws", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "a", Value: 7}, {Label: "b", Value: 3}, {Label: "pair", Unstated: true}}
			got, calls := drawsRun(drawsPair, s)
			assert.Equal(t, calls[0].values, []any{[]int{7, 3}}, "the values of the entries of a and b")
			assert.NoError(t, got.Refused, "the pair takes its entry by its label")
		})

		t.Run("replays a composite inside a composite before the draws after it", func(t *testing.T) {
			t.Parallel()
			outer := engine.Composite(func(c *engine.Case) []any {
				return []any{engine.Draw(c, pair, "pair"), engine.Draw(c, count, "c")}
			})
			s := settled()
			s.Draws = []engine.Entry{
				{Label: "a", Value: 1},
				{Label: "b", Value: 2},
				{Label: "pair", Value: []int{1, 2}},
				{Label: "c", Value: 5},
				{Label: "outer", Unstated: true},
			}
			_, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, outer, "outer")} }, s)
			assert.Equal(t, calls[0].values, []any{[]any{[]int{1, 2}, 5}}, "the inner pair, then c")
		})

		t.Run("takes the targets of the draws of a composite past the last entry", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "a", Value: 7}}
			_, calls := drawsRun(drawsPair, s)
			assert.Equal(t, calls[0].values, []any{[]int{7, 0}}, "the entry of a, then the target of b")
		})

		composites := []struct {
			name       string
			give       engine.Entry
			wantReason string
		}{
			{
				name:       "refuses an entry of another label at the end of a composite at the entry's label",
				give:       engine.Entry{Label: "other", Value: []int{7, 3}},
				wantReason: `the draw labelled "pair" takes the entry labelled "other"`,
			},
			{
				name:       "refuses a step entry at the end of a composite at the entry's label",
				give:       engine.Entry{Step: putStep},
				wantReason: `the draw labelled "pair" takes the step entry of "put"`,
			},
		}
		for _, tt := range composites {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = []engine.Entry{{Label: "a", Value: 7}, {Label: "b", Value: 3}, tt.give}
				got, calls := drawsRun(drawsPair, s)
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(
					t,
					[]any{f.Path, f.Reason},
					[]any{fault.Path{fault.Index(2), fault.Field("label")}, tt.wantReason},
					"the third entry's label, and the reason",
				)
				assert.Length(t, calls, 1, "no other case runs")
			})
		}

		digitOrPair := engine.OneOf(engine.Erase(count), engine.Erase(pair))
		drawsDigitOrPair := func(c *engine.Case) []any { return []any{engine.Draw(c, digitOrPair, "pair")} }

		t.Run("replays a one-of through a composite after the entries of the composite's draws", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{
				{Label: "a", Value: 7},
				{Label: "b", Value: 3},
				{Label: "pair", Value: []int{7, 3}},
			}
			got, calls := drawsRun(drawsDigitOrPair, s)
			assert.NoError(t, got.Refused, "the one-of takes its entry once the pair decodes")
			assert.Equal(t, calls[0].values, []any{[]int{7, 3}}, "the values of the entries of a and b")
			assert.True(t, sameChoices(calls[0].choices, integers(1, 7, 3)), "the index of the pair, then a and b")
		})

		t.Run("takes the entry of an alternative without a composite when the draw starts", func(t *testing.T) {
			t.Parallel()
			pairFirst := engine.OneOf(engine.Erase(pair), engine.Erase(count))
			s := settled()
			s.Draws = []engine.Entry{{Label: "pair", Value: 7}, {Label: "x", Value: 5}}
			_, calls := drawsRun(func(c *engine.Case) []any {
				return []any{engine.Draw(c, pairFirst, "pair"), engine.Draw(c, count, "x")}
			}, s)
			assert.Equal(t, calls[0].values, []any{7, 5}, "the integer of the one-of, then x")
			assert.True(t, sameChoices(calls[0].choices, integers(1, 7, 5)), "the index of the integer")
		})

		t.Run("serves the flags of a list of composites around the draws of each pair", func(t *testing.T) {
			t.Parallel()
			pairs := engine.List(pair, sizes(t, 0, 3))
			s := settled()
			s.Draws = []engine.Entry{
				{Label: "a", Value: 1},
				{Label: "b", Value: 2},
				{Label: "a", Value: 7},
				{Label: "b", Value: 3},
				{Label: "pairs", Value: [][]int{{1, 2}, {7, 3}}},
			}
			got, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, pairs, "pairs")} }, s)
			assert.NoError(t, got.Refused, "the list takes its entry last")
			assert.Equal(t, calls[0].values, []any{[][]int{{1, 2}, {7, 3}}}, "both pairs")
			assert.True(t, sameChoices(calls[0].choices, integers(1, 1, 2, 1, 7, 3, 0)),
				"a flag before each pair, and the stop")
		})

		optionals := []struct {
			name      string
			giveDraws []engine.Entry
			want      []any
		}{
			{
				name: "replays a present optional composite after the entries of its draws",
				giveDraws: []engine.Entry{
					{Label: "a", Value: 7},
					{Label: "b", Value: 3},
					{Label: "maybe", Value: []int{7, 3}},
				},
				want: []any{[]int{7, 3}},
			},
			{
				name:      "takes the entry of an absent optional composite when the draw starts",
				giveDraws: []engine.Entry{{Label: "maybe", Value: nil}},
				want:      []any{[]int(nil)},
			},
		}
		for _, tt := range optionals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				maybe := engine.Optional(pair)
				s := settled()
				s.Draws = tt.giveDraws
				got, calls := drawsRun(func(c *engine.Case) []any {
					if v := engine.Draw(c, maybe, "maybe"); v != nil {
						return []any{*v}
					}
					return []any{[]int(nil)}
				}, s)
				assert.NoError(t, got.Refused, "the optional takes its entry")
				assert.Equal(t, calls[0].values, tt.want, "the pair, or no pair")
			})
		}

		t.Run("does not test the recorded value of a filtered composite", func(t *testing.T) {
			t.Parallel()
			whole := pair.Filter(func(p []int) bool { return len(p) == 2 })
			s := settled()
			s.Draws = []engine.Entry{{Label: "a", Value: 7}, {Label: "b", Value: 3}, {Label: "pair", Value: []int{7}}}
			got, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, whole, "pair")} }, s)
			assert.NoError(t, got.Refused, "the filter takes its entry once the pair decodes")
			assert.Equal(t, calls[0].values, []any{[]int{7, 3}}, "the pair that the filter keeps")
		})

		t.Run("does not compare the recorded values of the composites of a unique list", func(t *testing.T) {
			t.Parallel()
			unique := engine.UniqueList(pair, sizes(t, 0, 3))
			s := settled()
			s.Draws = []engine.Entry{
				{Label: "a", Value: 1},
				{Label: "b", Value: 2},
				{Label: "a", Value: 3},
				{Label: "b", Value: 4},
				{Label: "pairs", Value: [][]int{{0}, {0}}},
			}
			got, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, unique, "pairs")} }, s)
			assert.NoError(t, got.Refused, "the unique list takes its entry last")
			assert.Equal(t, calls[0].values, []any{[][]int{{1, 2}, {3, 4}}}, "the two distinct pairs")
		})

		throughRefusals := []struct {
			name       string
			giveDraws  []engine.Entry
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "refuses the entry that the composite leaves before the entry of the one-of at its label",
				giveDraws: []engine.Entry{
					{Label: "a", Value: 7},
					{Label: "b", Value: 3},
					{Label: "b", Value: 4},
					{Label: "pair", Value: []int{7, 3}},
				},
				wantPath:   fault.Path{fault.Index(2), fault.Field("label")},
				wantReason: `the draw labelled "pair" takes the entry labelled "b"`,
			},
			{
				name: "refuses the next entry when the entry of the one-of follows a step entry",
				giveDraws: []engine.Entry{
					{Label: "a", Value: 7},
					{Label: "b", Value: 3},
					{Step: putStep},
					{Label: "pair", Value: []int{7, 3}},
				},
				wantPath:   fault.Path{fault.Index(0), fault.Field("label")},
				wantReason: `the draw labelled "pair" takes the entry labelled "a"`,
			},
			{
				name:       "refuses the next entry when a later entry of the one-of has a value without a composite",
				giveDraws:  []engine.Entry{{Label: "a", Value: 7}, {Label: "pair", Value: 5}},
				wantPath:   fault.Path{fault.Index(0), fault.Field("label")},
				wantReason: `the draw labelled "pair" takes the entry labelled "a"`,
			},
			{
				name: "refuses an entry of the one-of without a value at the entry's value",
				giveDraws: []engine.Entry{
					{Label: "a", Value: 7},
					{Label: "b", Value: 3},
					{Label: "pair", Unstated: true},
				},
				wantPath:   fault.Path{fault.Index(2), fault.Field("value")},
				wantReason: "the entry records no value",
			},
		}
		for _, tt := range throughRefusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = tt.giveDraws
				got, calls := drawsRun(drawsDigitOrPair, s)
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(t, []any{f.Path, f.Reason}, []any{tt.wantPath, tt.wantReason}, "the entry, and the reason")
				assert.Length(t, calls, 1, "no other case runs")
			})
		}

		t.Run("refuses the entry of a one-of that a draw inside takes at the entry's label", func(t *testing.T) {
			t.Parallel()
			sameLabel := engine.Composite(func(c *engine.Case) []int { return []int{engine.Draw(c, count, "pair")} })
			single := engine.OneOf(sameLabel)
			s := settled()
			s.Draws = []engine.Entry{{Label: "pair", Value: 5}, {Label: "pair", Value: []int{5}}}
			got, _ := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, single, "pair")} }, s)
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, []any{f.Path, f.Reason}, []any{
				fault.Path{fault.Index(0), fault.Field("label")},
				`a draw inside the draw labelled "pair" takes the entry of the draw`,
			}, "the first entry's label, and the reason")
		})

		keyPair := engine.Composite(func(c *engine.Case) int { return engine.Draw(c, count, "k") })
		sum := func(p []int) int { return p[0] + p[1] }
		split := func(s int) ([]int, error) { return []int{s}, nil }
		same := func(v int) int { return v }
		digitBack := func(v int) (int, error) { return v, nil }
		erasedList := func(self engine.Generator[any]) engine.Generator[any] {
			return engine.Erase(engine.List(self, sizes(t, 0, 2)))
		}
		draws := func(g engine.Generator[any]) func(*engine.Case) []any {
			return func(c *engine.Case) []any { return []any{engine.Draw(c, g, "g")} }
		}
		throughs := []struct {
			name     string
			giveDraw func(*engine.Case) []any
		}{
			{
				name:     "takes a later entry for a one-of with a composite inside",
				giveDraw: draws(engine.OneOf(engine.Erase(pair))),
			},
			{
				name:     "takes a later entry for an optional with a composite inside",
				giveDraw: draws(engine.Erase(engine.Optional(pair))),
			},
			{
				name:     "takes a later entry for a list with a composite inside",
				giveDraw: draws(engine.Erase(engine.List(pair, sizes(t, 0, 2)))),
			},
			{
				name:     "takes a later entry for a unique list with a composite inside",
				giveDraw: draws(engine.Erase(engine.UniqueList(pair, sizes(t, 0, 2)))),
			},
			{
				name:     "takes a later entry for a dict with a composite key",
				giveDraw: draws(engine.Erase(engine.Dict(keyPair, count, sizes(t, 0, 2)))),
			},
			{
				name:     "takes a later entry for a dict with a composite value",
				giveDraw: draws(engine.Erase(engine.Dict(count, pair, sizes(t, 0, 2)))),
			},
			{
				name:     "takes a later entry for a filter of a composite",
				giveDraw: draws(engine.Erase(pair.Filter(func([]int) bool { return true }))),
			},
			{
				name:     "takes a later entry for a map back of a one-of with a composite inside",
				giveDraw: draws(engine.Erase(engine.OneOf(pair).MapBack(sum, split))),
			},
			{
				name:     "takes a later entry for a recursive with a composite base",
				giveDraw: draws(engine.Recursive(engine.Erase(pair), erasedList, engine.DefaultMaxLeaves)),
			},
			{
				name: "takes a later entry for a recursive with a composite inside its extension",
				giveDraw: draws(
					engine.Recursive(engine.Erase(count), func(self engine.Generator[any]) engine.Generator[any] {
						return engine.OneOf(engine.Erase(pair), erasedList(self))
					}, engine.DefaultMaxLeaves),
				),
			},
		}
		for _, tt := range throughs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = []engine.Entry{{Label: "a", Value: 7}, {Label: "g", Unstated: true}}
				got, _ := drawsRun(tt.giveDraw, s)
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(t, []any{f.Path, f.Reason}, []any{
					fault.Path{fault.Index(1), fault.Field("value")},
					"the entry records no value",
				}, "the second entry's value, and the reason")
			})
		}

		nexts := []struct {
			name     string
			giveDraw func(*engine.Case) []any
		}{
			{
				name:     "takes the next entry for a one-of without a composite",
				giveDraw: draws(engine.OneOf(engine.Erase(count))),
			},
			{
				name:     "takes the next entry for an optional without a composite",
				giveDraw: draws(engine.Erase(engine.Optional(count))),
			},
			{
				name:     "takes the next entry for a list without a composite",
				giveDraw: draws(engine.Erase(engine.List(count, sizes(t, 0, 2)))),
			},
			{
				name:     "takes the next entry for a unique list without a composite",
				giveDraw: draws(engine.Erase(engine.UniqueList(count, sizes(t, 0, 2)))),
			},
			{
				name:     "takes the next entry for a dict without a composite",
				giveDraw: draws(engine.Erase(engine.Dict(count, count, sizes(t, 0, 2)))),
			},
			{
				name:     "takes the next entry for a filter without a composite",
				giveDraw: draws(engine.Erase(count.Filter(func(int) bool { return true }))),
			},
			{
				name:     "takes the next entry for a map back without a composite",
				giveDraw: draws(engine.Erase(count.MapBack(same, digitBack))),
			},
			{
				name:     "takes the next entry for an erased generator without a composite",
				giveDraw: draws(engine.Erase(count)),
			},
			{
				name:     "takes the next entry for a recursive without a composite",
				giveDraw: draws(engine.Recursive(engine.Erase(count), erasedList, engine.DefaultMaxLeaves)),
			},
		}
		for _, tt := range nexts {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = []engine.Entry{{Label: "a", Value: 7}, {Label: "g", Unstated: true}}
				got, _ := drawsRun(tt.giveDraw, s)
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(t, []any{f.Path, f.Reason}, []any{
					fault.Path{fault.Index(0), fault.Field("label")},
					`the draw labelled "g" takes the entry labelled "a"`,
				}, "the first entry's label, and the reason")
			})
		}

		t.Run("does not map back the value of a one-of through a composite", func(t *testing.T) {
			t.Parallel()
			total := engine.OneOf(pair).MapBack(sum, split)
			s := settled()
			s.Draws = []engine.Entry{{Label: "a", Value: 7}, {Label: "b", Value: 3}, {Label: "sum", Value: 10}}
			got, calls := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, total, "sum")} }, s)
			assert.NoError(t, got.Refused, "the map back takes its entry once the pair decodes")
			assert.Equal(t, calls[0].values, []any{10}, "the sum of the pair")
		})

		t.Run("takes the entry of a draw through a composite that draws nothing", func(t *testing.T) {
			t.Parallel()
			nothing := engine.OneOf(engine.Composite(func(*engine.Case) int { return 1 }))
			s := settled()
			s.Draws = []engine.Entry{{Label: "pair", Value: 1}, {Label: "x", Value: 5}}
			got, calls := drawsRun(func(c *engine.Case) []any {
				return []any{engine.Draw(c, nothing, "pair"), engine.Draw(c, count, "x")}
			}, s)
			assert.NoError(t, got.Refused, "each draw takes its entry")
			assert.Equal(t, calls[0].values, []any{1, 5}, "the composite, then x")
		})

		t.Run("refuses the next entry when no later entry has the label at the next entry", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "x", Value: 5}, {Label: "a", Value: 7}}
			got, _ := drawsRun(func(c *engine.Case) []any {
				return []any{engine.Draw(c, count, "x"), engine.Draw(c, digitOrPair, "pair")}
			}, s)
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, []any{f.Path, f.Reason}, []any{
				fault.Path{fault.Index(1), fault.Field("label")},
				`the draw labelled "pair" takes the entry labelled "a"`,
			}, "the second entry's label, and the reason")
		})

		t.Run("serves the targets of each draw past the last entry and of a choice after them", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Value: 4}}
			_, calls := drawsRun(func(c *engine.Case) []any {
				return []any{
					engine.Draw(c, count, "count"),
					engine.Draw(c, count, "x"),
					engine.Draw(c, count, "y"),
					c.Integer(digitRange).Magnitude(),
				}
			}, s)
			assert.Equal(t, calls[0].values, []any{4, 0, 0, uint64(0)}, "the entry's value, then the targets")
		})

		emptyLabels := []struct {
			name     string
			giveDraw func(*engine.Case) []any
			giveSeen []engine.Entry
			wantPath fault.Path
		}{
			{
				name:     "refuses a step entry for a draw labelled with the empty string",
				giveDraw: func(c *engine.Case) []any { return []any{engine.Draw(c, count, "")} },
				giveSeen: []engine.Entry{{Step: putStep}},
				wantPath: fault.Path{fault.Index(0), fault.Field("label")},
			},
			{
				name:     "refuses a step entry at the end of a composite labelled with the empty string",
				giveDraw: func(c *engine.Case) []any { return []any{engine.Draw(c, pair, "")} },
				giveSeen: []engine.Entry{{Label: "a", Value: 7}, {Label: "b", Value: 3}, {Step: putStep}},
				wantPath: fault.Path{fault.Index(2), fault.Field("label")},
			},
		}
		for _, tt := range emptyLabels {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = tt.giveSeen
				got, _ := drawsRun(tt.giveDraw, s)
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(
					t,
					[]any{f.Path, f.Reason},
					[]any{tt.wantPath, `the draw labelled "" takes the step entry of "put"`},
					"the step entry's label, and the reason",
				)
			})
		}

		t.Run(
			"refuses an entry whose choices the replay of the inverse does not decode at its value",
			func(t *testing.T) {
				t.Parallel()
				liar := engine.NewInvertible("liar", func(c *engine.Case) int {
					return int(c.Integer(digitRange).Magnitude())
				}, func(engine.Inversion, any) ([]engine.Step, int, error) {
					return []engine.Step{{Bounds: choice.OfInteger(digitRange), Value: integers(3)[0]}}, 5, nil
				})
				s := settled()
				s.Draws = []engine.Entry{{Label: "x", Value: 5}}
				got, _ := drawsRun(func(c *engine.Case) []any { return []any{engine.Draw(c, liar, "x")} }, s)
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(t, []any{f.Path, f.Reason}, []any{
					fault.Path{fault.Index(0), fault.Field("value")},
					"the choices of 5 decode to 3",
				}, "the entry's value, and the reason")
			},
		)

		dicts := []struct {
			name      string
			giveDraw  func(*engine.Case) []any
			giveDraws []engine.Entry
			wantPath  fault.Path
		}{
			{
				name:     "refuses the entry of a dict with composite values at the entry's value",
				giveDraw: draws(engine.Erase(engine.Dict(count, pair, sizes(t, 0, 2)))),
				giveDraws: []engine.Entry{
					{Label: "a", Value: 1},
					{Label: "b", Value: 2},
					{Label: "g", Value: map[int][]int{5: {1, 2}}},
				},
				wantPath: fault.Path{fault.Index(2), fault.Field("value"), fault.Key(5)},
			},
			{
				name:      "refuses the entry of a dict with composite keys at the entry's value",
				giveDraw:  draws(engine.Erase(engine.Dict(keyPair, count, sizes(t, 0, 2)))),
				giveDraws: []engine.Entry{{Label: "k", Value: 5}, {Label: "g", Value: map[int]int{5: 1}}},
				wantPath:  fault.Path{fault.Index(1), fault.Field("value"), fault.Key(5)},
			},
		}
		for _, tt := range dicts {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Draws = tt.giveDraws
				got, _ := drawsRun(tt.giveDraw, s)
				assert.ErrorIs(t, got.Refused, engine.ErrNoInverse, "the order of the entries of a map is unknown")
				f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
				assert.Equal(t, []any{f.Path, f.Reason}, []any{
					tt.wantPath,
					"a map does not state the order in which its composites drew",
				}, "the entry's value at its key, and the reason")
			})
		}

		t.Run("refuses a draw entry without a value at the entry's value", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = []engine.Entry{{Label: "count", Unstated: true}}
			got, _ := drawsRun(countAndName, s)
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, []any{f.Path, f.Reason}, []any{
				fault.Path{fault.Index(0), fault.Field("value")},
				"the entry records no value",
			}, "the first entry's value, and the reason")
		})

		t.Run("refuses the entry of a draw that takes a step entry at the entry's label", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = twoSteps
			got, _ := drawsRun(countAndName, s)
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, f.Path, fault.Path{fault.Index(0), fault.Field("label")}, "the first entry's label")
			assert.Equal(t, f.Reason, `the draw labelled "count" takes the step entry of "put"`,
				"the draw and the step")
		})
	})

	t.Run("Trace", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the trace of the case of the entries alone", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = twoSteps
			var mu sync.Mutex
			var followed []bool
			engine.Run(func(c *engine.Case) {
				_, ok := c.Trace()
				mu.Lock()
				followed = append(followed, ok)
				mu.Unlock()
				c.Integer(digitRange)
			}, s)
			assert.Equal(t, []any{followed[0], slices.Index(followed[1:], true)}, []any{true, -1},
				"the case of the entries follows a trace, and no later case does")
		})

		t.Run("returns the trace of a stored case whose choices decode to other values", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Cases = 1
			s.Stored = []engine.Stored{{
				Choices: integers(3),
				Matches: func(*engine.Case) bool { return false },
				Trace:   []engine.Entry{{Label: "count", Value: 4}},
			}}
			var mu sync.Mutex
			var followed []bool
			engine.Run(func(c *engine.Case) {
				_, ok := c.Trace()
				mu.Lock()
				followed = append(followed, ok)
				mu.Unlock()
				engine.Draw(c, count, "count")
			}, s)
			assert.Equal(t, followed[:2], []bool{false, true}, "the case of the choices, then the case of the trace")
		})
	})

	t.Run("Names", func(t *testing.T) {
		t.Parallel()

		t.Run("reports whether a step entry from the next entry on names the action", func(t *testing.T) {
			t.Parallel()
			var got []bool
			entries := []engine.Entry{{Step: putStep}, {Label: "count", Value: 4}, {Step: getStep}}
			followTrace(entries, func(_ *engine.Case, tr engine.Trace) {
				got = append(got, tr.Names("put"), tr.Names("get"), tr.Names("flush"))
				tr.Take()
				got = append(got, tr.Names("put"))
			})
			assert.Equal(t, got, []bool{true, true, false, false}, "put and get, then get alone once put is taken")
		})
	})

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the next entry when it is a step entry", func(t *testing.T) {
			t.Parallel()
			var got []any
			entries := []engine.Entry{{Label: "count", Value: 4}, {Step: getStep}}
			followTrace(entries, func(c *engine.Case, tr engine.Trace) {
				_, before := tr.Next()
				engine.Draw(c, count, "count")
				step, next := tr.Next()
				tr.Take()
				_, after := tr.Next()
				got = []any{before, step, next, after}
			})
			assert.Equal(t, got, []any{false, *getStep, true, false},
				"no step at the draw entry, the step entry after it, and none past the entries")
		})
	})

	t.Run("Take", func(t *testing.T) {
		t.Parallel()

		t.Run("serves its values to the next requests, and leaves the next entry to the next draw", func(t *testing.T) {
			t.Parallel()
			var got []any
			entries := []engine.Entry{{Step: putStep}, {Label: "count", Value: 4}}
			followTrace(entries, func(c *engine.Case, tr engine.Trace) {
				tr.Take(1, 2)
				first, second := c.Integer(digitRange).Magnitude(), c.Integer(digitRange).Magnitude()
				got = []any{first, second, engine.Draw(c, count, "count")}
			})
			assert.Equal(t, got, []any{uint64(1), uint64(2), 4}, "the step's values, then the draw's entry")
		})
	})

	t.Run("Prepare", func(t *testing.T) {
		t.Parallel()

		t.Run("serves its values to the next requests before their targets", func(t *testing.T) {
			t.Parallel()
			var got []uint64
			followTrace(twoSteps, func(c *engine.Case, tr engine.Trace) {
				tr.Prepare(3, 5)
				for range 3 {
					got = append(got, c.Integer(digitRange).Magnitude())
				}
			})
			assert.Equal(t, got, []uint64{3, 5, 0}, "the two values, then the target")
		})
	})

	t.Run("Refuse", func(t *testing.T) {
		t.Parallel()

		t.Run("refuses the next entry at its step, before any other case", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Draws = twoSteps
			calls := 0
			got := engine.Run(func(c *engine.Case) {
				calls++
				if tr, ok := c.Trace(); ok {
					tr.Take(1, 0)
					tr.Refuse(refusal)
				}
			}, s)
			assert.Equal(t, got, engine.Result{Refused: got.Refused}, "the refusal, and nothing else")
			f := assert.ErrorAs[*fault.Error](t, got.Refused, "a fault")
			assert.Equal(t, []any{f.Path, f.Reason}, []any{fault.Path{fault.Index(1), fault.Field("step")}, refusal},
				"the second entry's step, and the reason")
			assert.Equal(t, calls, 1, "no other case runs")
		})
	})
}

// traceAllocs are the cases of the allocation ceilings of the trace's
// methods that serve values or refuse, each a whole run whose case of
// twoSteps ends at a refusal.
var traceAllocs = []alloctest.Case{
	{
		Name: "Prepare",
		Call: func(assert.TB) {
			engine.Run(func(c *engine.Case) {
				tr, _ := c.Trace()
				tr.Prepare(1)
				tr.Refuse(refusal)
			}, refusedRun)
		},
		Allocs: servedRunAllocs,
	},
	{
		Name: "Take",
		Call: func(assert.TB) {
			engine.Run(func(c *engine.Case) {
				tr, _ := c.Trace()
				tr.Take(1)
				tr.Refuse(refusal)
			}, refusedRun)
		},
		Allocs: servedRunAllocs,
	},
	{
		Name: "Refuse",
		Call: func(assert.TB) {
			engine.Run(func(c *engine.Case) {
				tr, _ := c.Trace()
				tr.Refuse(refusal)
			}, refusedRun)
		},
		Allocs: refusedRunAllocs,
	},
}

// TestDrawsAllocs checks the allocation ceilings of the trace's methods.
func TestDrawsAllocs(t *testing.T) {
	alloctest.Check(t, traceAllocs)
	c, tr := leakedTrace()
	assert.MaxAllocs(t, func() { _, _ = c.Trace() }, 0, "Trace allocates nothing")
	assert.MaxAllocs(t, func() { _ = tr.Names("get") }, 0, "Names allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = tr.Next() }, 0, "Next allocates nothing")
}

// BenchmarkDraws measures each method of the trace under its ceiling: the
// methods that serve values or refuse on a whole run that the case of the
// entries ends, and the others on the trace of a case whose body has ended.
func BenchmarkDraws(b *testing.B) {
	for _, c := range traceAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
	b.Run("Trace", func(b *testing.B) {
		var got bool
		cs, _ := leakedTrace()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			_, got = cs.Trace()
		}
		assert.True(b, got, "the case of the entries follows them")
	})
	b.Run("Names", func(b *testing.B) {
		var got bool
		_, tr := leakedTrace()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = tr.Names("get")
		}
		assert.True(b, got, "the second entry names get")
	})
	b.Run("Next", func(b *testing.B) {
		var got engine.MachineStep
		_, tr := leakedTrace()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = tr.Next()
		}
		assert.Equal(b, got, *putStep, "the first entry's step")
	})
}

// followTrace runs the case of entries, whose body calls follow with the
// case and its trace.
func followTrace(entries []engine.Entry, follow func(c *engine.Case, tr engine.Trace)) {
	s := settled()
	s.Cases = 1
	s.Draws = entries
	engine.Run(func(c *engine.Case) {
		if tr, ok := c.Trace(); ok {
			follow(c, tr)
		}
	}, s)
}

// leakedTrace returns the case of twoSteps, whose body took no entry, and
// its trace, for a caller to call after the run.
func leakedTrace() (*engine.Case, engine.Trace) {
	var c *engine.Case
	var tr engine.Trace
	followTrace(twoSteps, func(followed *engine.Case, trace engine.Trace) { c, tr = followed, trace })
	return c, tr
}

// drawsRun runs a body that returns what draw draws under s, and returns
// the result and what each call drew and chose, in call order.
func drawsRun(draw func(*engine.Case) []any, s engine.Settings) (engine.Result, []call) {
	var calls []call
	result := engine.Run(func(c *engine.Case) {
		var values []any
		defer func() { calls = append(calls, call{values: values, choices: c.Choices()}) }()
		values = draw(c)
	}, s)
	return result, calls
}
