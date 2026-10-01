// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// invalidRelevance is the first value past the three relevances.
const invalidRelevance engine.Relevance = 3

// TestExplain checks what the explain phase finds for each draw of a
// counterexample, through whole runs pinned to what the definition's
// executable reference reports, and pins each relevance's spelling.
func TestExplain(t *testing.T) {
	t.Parallel()

	digit, percent := engine.Integer(0, 9), engine.Integer(0, 100)
	small, wide := engine.Integer(0, 1000), engine.Integer(0, 1_000_000_000)
	noisy := func(c *engine.Case) string {
		value := engine.Draw(c, wide, "n")
		engine.Draw(c, wide, "noise")
		return failsWhen(value > 1000, "above")
	}

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give engine.Relevance
			want bool
		}{
			{name: "reports true for Untested", give: engine.Untested, want: true},
			{name: "reports true for ValueMatters", give: engine.ValueMatters, want: true},
			{name: "reports false past ValueMatters", give: invalidRelevance, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a relevance")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give engine.Relevance
			want string
		}{
			{name: "returns untested for Untested", give: engine.Untested, want: "untested"},
			{name: "returns any-value-fails for AnyValueFails", give: engine.AnyValueFails, want: "any-value-fails"},
			{name: "returns value-matters for ValueMatters", give: engine.ValueMatters, want: "value-matters"},
			{
				name: "returns Relevance(3) for a value that is no relevance",
				give: invalidRelevance,
				want: "Relevance(3)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the relevance's spelling")
			})
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			p      property
			budget int
			want   reference
		}{
			{
				name: "marks a draw for which every filling fails the same way",
				p: func(c *engine.Case) string {
					engine.Draw(c, percent, "value")
					return "every"
				},
				budget: engine.DefaultShrink,
				want: reference{
					explanation: []engine.Explained{{Label: "value", Value: 0, Relevance: engine.AnyValueFails}},
					token:       "prop1:AAA",
					runs:        5,
					calls:       7,
					digest:      "03fcc5853d14ed8a31cf56557c7215d753f109c0b064931c0806ba0374fde3d6",
				},
			},
			{
				name: "marks a draw at its target for which a filling passes",
				p: func(c *engine.Case) string {
					return failsWhen(engine.Draw(c, digit, "x") == 0, "zero")
				},
				budget: engine.DefaultShrink,
				want: reference{
					explanation: []engine.Explained{{Label: "x", Value: 0, Relevance: engine.ValueMatters}},
					token:       "prop1:AAA",
					runs:        2,
					calls:       4,
					digest:      "575d7d360cd1a317d068590b2298dcc8f2d63e11c46b63bf22737b3ba19664c3",
				},
			},
			{
				name: "marks a draw for which a filling fails another way",
				p: func(c *engine.Case) string {
					if engine.Draw(c, small, "x") > 5 {
						return "big"
					}
					return "small"
				},
				budget: engine.DefaultShrink,
				want: reference{
					explanation: []engine.Explained{{Label: "x", Value: 0, Relevance: engine.ValueMatters}},
					token:       "prop1:AAA",
					runs:        2,
					calls:       4,
					digest:      "32e0b1b870a0668189034f1cbf404a660eb66d6ef19f6c9776e972490c425910",
				},
			},
			{
				name:   "reports the value one step towards the target that passes",
				p:      noisy,
				budget: engine.DefaultShrink,
				want: reference{
					explanation: []engine.Explained{
						{Label: "n", Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
						{Label: "noise", Value: 0, Relevance: engine.AnyValueFails},
					},
					token:  "prop1:AOkHAAA",
					runs:   60,
					calls:  63,
					digest: "d32832002c1213e7e5049a6e8f93228032bfc42606d348c44e1fc88ff3bc95c0",
				},
			},
			{
				name: "reports no nearest value when the step fails another way",
				p: func(c *engine.Case) string {
					value := engine.Draw(c, wide, "n")
					if value == 1000 {
						return "limit"
					}
					return failsWhen(value > 1000, "above")
				},
				budget: engine.DefaultShrink,
				want: reference{
					explanation: []engine.Explained{{Label: "n", Value: 1001, Relevance: engine.ValueMatters}},
					token:       "prop1:AOkH",
					runs:        48,
					calls:       51,
					digest:      "d6fd9ff847294f85ecb23f2754c06639927b23453393f28aeb66e72595dca994",
				},
			},
			{
				name: "leaves a draw that made no choice untested",
				p: func(c *engine.Case) string {
					engine.Draw(c, engine.Just(5), "five")
					return failsWhen(engine.Draw(c, small, "x") > 500, "big")
				},
				budget: engine.DefaultShrink,
				want: reference{
					explanation: []engine.Explained{
						{Label: "five", Value: 5, Relevance: engine.Untested},
						{Label: "x", Value: 501, Relevance: engine.ValueMatters, NearestPassing: 500},
					},
					token:  "prop1:APUD",
					runs:   23,
					calls:  26,
					digest: "585f9e776464aebe7171971ba9668612edae1bc5bb03ce8d99d827ef829f37b3",
				},
			},
			{
				name:   "leaves a draw untested when the budget runs out before its nearest value",
				p:      noisy,
				budget: 55,
				want: reference{
					explanation: []engine.Explained{
						{Label: "n", Value: 1001, Relevance: engine.Untested},
						{Label: "noise", Value: 0, Relevance: engine.Untested},
					},
					token:  "prop1:AOkHAAA",
					runs:   55,
					calls:  58,
					digest: "37c6eb7c90f651f8f675eb870465615f9d5b603e7d95f5cd91ac0129eaa6c30b",
				},
			},
			{
				name:   "leaves a draw untested when the budget runs out among its fillings",
				p:      noisy,
				budget: 54,
				want: reference{
					explanation: []engine.Explained{
						{Label: "n", Value: 1001, Relevance: engine.Untested},
						{Label: "noise", Value: 0, Relevance: engine.Untested},
					},
					token:  "prop1:AOkHAAA",
					runs:   54,
					calls:  57,
					digest: "87f3aefcd8b54442476207862ec3049c35939b784d66d7866b4a4d66fec82e30",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Shrink = tt.budget
				got, trace := traced(tt.p, s)
				matchesReference(t, got, trace, tt.want)
			})
		}

		sevens := digit.Filter(func(v int) bool { return v == 7 })
		triple := engine.UniqueList(engine.Integer(0, 2), sizes(t, 3, 3))
		fillings := []struct {
			name string
			p    property
			seed uint64
			want reference
		}{
			{
				name: "skips a filling whose decode rejects at no cost",
				p: func(c *engine.Case) string {
					engine.Draw(c, triple, "xs")
					return "always"
				},
				seed: 0,
				want: reference{
					explanation: []engine.Explained{
						{Label: "xs", Value: []int{0, 1, 2}, Relevance: engine.AnyValueFails},
					},
					token:  "prop1:AAEAAAABAAEAAQACAAA",
					runs:   20,
					calls:  23,
					digest: "6c280b38efc143ba281da721da72ba2c5c028ff8e9fdb87f3c03370cac207f85",
				},
			},
			{
				name: "leaves a draw untested when no filling decodes",
				p: func(c *engine.Case) string {
					engine.Draw(c, sevens, "n")
					return "seven"
				},
				seed: 2,
				want: reference{
					explanation: []engine.Explained{{Label: "n", Value: 7, Relevance: engine.Untested}},
					token:       "prop1:AAc",
					runs:        5,
					calls:       10,
					digest:      "699991e0eceae7a2f424e139518781d08f4b99ee1478f13bab19ea2287e80927",
				},
			},
			{
				name: "marks a draw whose one decoded filling fails the same way",
				p: func(c *engine.Case) string {
					engine.Draw(c, sevens, "n")
					return "seven"
				},
				seed: referenceSeed,
				want: reference{
					explanation: []engine.Explained{{Label: "n", Value: 7, Relevance: engine.AnyValueFails}},
					token:       "prop1:AAc",
					runs:        6,
					calls:       22,
					digest:      "7c032387f5e8a527179e33d0d3472ca8a8e19d6683ffa616ea41f3f6957fa0cb",
				},
			},
		}
		for _, tt := range fillings {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := settled()
				s.Seed = tt.seed
				got, trace := traced(tt.p, s)
				matchesReference(t, got, trace, tt.want)
			})
		}

		t.Run("explains nothing when explaining is off", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Explain = false
			got, trace := traced(noisy, s)
			assert.Empty(t, got.Explanation, "no explanation")
			assert.Equal(t, got.Runs, 53, "the runs of shrinking alone")
			assert.Equal(t, digestOf(trace), "184d753561be9e47391cfc5870e98193899e8b21c88e45b237bbd20e9dd1b951",
				"no explanation run")
		})
	})
}

// TestExplainZeroAlloc checks that no method of Relevance allocates.
func TestExplainZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.ValueMatters.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = engine.ValueMatters.String() }, 0, "String allocates nothing")
}

// BenchmarkExplain measures each method of Relevance under a ceiling of no
// allocation.
func BenchmarkExplain(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.ValueMatters.Valid()
		}
		assert.True(b, got, "ValueMatters is a relevance")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.ValueMatters.String()
		}
		assert.Equal(b, got, "value-matters", "the relevance's spelling")
	})
}
