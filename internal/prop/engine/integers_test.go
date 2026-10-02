// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestIntegers checks the passes that move integers towards their targets,
// alone, in pairs, in groups, with the spans they size and with the data
// before them, each through a run from a stored failing case that the pass
// shrinks, pinned to what the definition's executable reference reports.
func TestIntegers(t *testing.T) {
	t.Parallel()

	digit, hundred := engine.Integer(0, 9), engine.Integer(0, 100)
	small, wide := engine.Integer(0, 1000), engine.Integer(0, 1_000_000_000)
	signed, positive := engine.Integer(-100, 100), engine.Integer(1, 1000)
	signedList := engine.List(signed, unbounded(t, 0))
	unit := engine.Float(0.0, 10.0, choice.ExcludeNaN)
	percent, five := engine.List(hundred, unbounded(t, 0)), engine.Just(5)
	anyBytes, firstByte := engine.Bytes(unbounded(t, 0)), engine.Bytes(unbounded(t, 1))
	twoBytes := engine.Bytes(unbounded(t, 2))
	exact := make([]engine.Generator[[]int], 10)
	for n := range exact {
		exact[n] = engine.List(digit, sizes(t, n, n))
	}
	indexed := func(c *engine.Case) string {
		values, index := engine.Draw(c, percent, "xs"), engine.Draw(c, digit, "i")
		return failsWhen(index < len(values) && values[index] > 50, "indexed")
	}
	high := func(g engine.Generator[[]byte]) property {
		return func(c *engine.Case) string {
			value, index := engine.Draw(c, g, "b"), engine.Draw(c, digit, "i")
			return failsWhen(index < len(value) && value[index] > 200, "high")
		}
	}

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			p      property
			stored []choice.Choice
			want   reference
		}{
			{
				name:   "lowers 2000 to the boundary 1001 by bisection",
				p:      above(wide),
				stored: integers(2000),
				want: reference{
					explanation: []engine.Explained{
						{Label: "n", Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
					},
					token:  "prop1:AOkH",
					runs:   24,
					calls:  26,
					digest: "48b73811cfca0ae3e7854260c761829e0c292e9a555e749065732cdf7009ce46",
				},
			},
			{
				name: "moves a value below its target to the simpler value above it",
				p: func(c *engine.Case) string {
					values := engine.Draw(c, signedList, "xs")
					reversed := slices.Clone(values)
					slices.Reverse(reversed)
					return failsWhen(!slices.Equal(values, reversed), "reverse")
				},
				stored: integers(1, 0, 1, -1, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{0, 1}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEAAAABAAEAAA",
					runs:        27,
					calls:       29,
					digest:      "e476d7f21ca58386a40ec34c6ac82f46bffcfc5d4bec0098d08538475502cacc",
				},
			},
			{
				name: "crosses the target to a nearer failing value",
				p: func(c *engine.Case) string {
					value := engine.Draw(c, signed, "x")
					return failsWhen(max(value, -value) >= 2 && value != 2, "far")
				},
				stored: integers(3),
				want: reference{
					explanation: []engine.Explained{{Label: "x", Value: -2, Relevance: engine.AnyValueFails}},
					token:       "prop1:AQI",
					runs:        9,
					calls:       11,
					digest:      "bed2c433c787ec6a812a68c66e04cd1c39dd9222e133a7ba9a4f3b22e1653fdd",
				},
			},
			{
				name: "moves a float that no failure depends on to its target",
				p: func(c *engine.Case) string {
					engine.Draw(c, unit, "v")
					return failsWhen(engine.Draw(c, wide, "n") > 1000, "above")
				},
				stored: []choice.Choice{float(3.75), unsigned(2000)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "v", Value: 0.0, Relevance: engine.AnyValueFails},
						{Label: "n", Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
					},
					token:  "prop1:AgAAAAAAAAAAAOkH",
					runs:   35,
					calls:  37,
					digest: "b7d15fb2a89d162a8d2dbe7907454841616c88cb7b18cbe80ad2f69c6912ae9e",
				},
			},
			{
				name: "lowers two values one apart by one amount",
				p: func(c *engine.Case) string {
					x, y := engine.Draw(c, positive, "x"), engine.Draw(c, positive, "y")
					return failsWhen(x >= 10 && (x-y == 1 || y-x == 1), "one")
				},
				stored: integers(36, 37),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 10, Relevance: engine.ValueMatters, NearestPassing: 9},
						{Label: "y", Value: 9, Relevance: engine.ValueMatters, NearestPassing: 8},
					},
					token:  "prop1:AAoACQ",
					runs:   60,
					calls:  62,
					digest: "8b8f6dc890a623a804d6624b11d195c48588d46f6cf3297e5a0e180d3c40fe56",
				},
			},
			{
				name: "leaves two values on both sides of the target",
				p: func(c *engine.Case) string {
					x, y := engine.Draw(c, signed, "x"), engine.Draw(c, signed, "y")
					return failsWhen(y-x >= 7, "spread")
				},
				stored: integers(-3, 4),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: -3, Relevance: engine.ValueMatters, NearestPassing: -2},
						{Label: "y", Value: 4, Relevance: engine.ValueMatters, NearestPassing: 3},
					},
					token:  "prop1:AQMABA",
					runs:   19,
					calls:  21,
					digest: "1adb8c08f51a530fa2d12261b93b3ff97358cf1decffece82006315408269bb0",
				},
			},
			{
				name: "lowers a count with the items it sizes",
				p: func(c *engine.Case) string {
					count := engine.Draw(c, digit, "count")
					items := make([]int, count)
					for i := range items {
						items[i] = engine.Draw(c, digit, "item")
					}
					return failsWhen(count >= 2 && items[count-1] == 9, "nine")
				},
				stored: integers(3, 0, 0, 9),
				want: reference{
					explanation: []engine.Explained{
						{Label: "count", Value: 2, Relevance: engine.ValueMatters, NearestPassing: 1},
						{Label: "item", Value: 0, Relevance: engine.AnyValueFails},
						{Label: "item", Value: 9, Relevance: engine.ValueMatters, NearestPassing: 8},
					},
					token:  "prop1:AAIAAAAJ",
					runs:   41,
					calls:  43,
					digest: "631cbf2e2819e80f2634adcac8e9ac127b0eacc1af025699f74829bf458965a9",
				},
			},
			{
				name: "deletes nothing after a step that sizes nothing",
				p: func(c *engine.Case) string {
					x := engine.Draw(c, digit, "x")
					engine.Draw(c, digit, "y")
					return failsWhen(x == 9, "nine")
				},
				stored: integers(9, 5),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 9, Relevance: engine.ValueMatters, NearestPassing: 8},
						{Label: "y", Value: 0, Relevance: engine.AnyValueFails},
					},
					token:  "prop1:AAkAAA",
					runs:   18,
					calls:  20,
					digest: "9f7d59a9f3be642800fe3269e1e4dd768ba6fea4d32d5b45bc8a3b49e7f8f6a4",
				},
			},
			{
				name: "deletes an element inside the list that a count sizes",
				p: func(c *engine.Case) string {
					count := engine.Draw(c, digit, "count")
					return failsWhen(slices.Contains(engine.Draw(c, exact[count], "xs"), 9), "nine")
				},
				stored: integers(3, 1, 0, 1, 0, 1, 9, 0),
				want: reference{
					explanation: []engine.Explained{
						{Label: "count", Value: 1, Relevance: engine.ValueMatters, NearestPassing: 0},
						{Label: "xs", Value: []int{9}, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEAAQAJAAA",
					runs:   46,
					calls:  48,
					digest: "1414793e6dccdaae5b2b78ac8f244a58c01ed93da1a12e7caf0d89bd69cc40fc",
				},
			},
			{
				name: "moves value from one integer to the next with the same bounds",
				p: func(c *engine.Case) string {
					x, y := engine.Draw(c, small, "x"), engine.Draw(c, small, "y")
					return failsWhen(x+y > 500, "sum")
				},
				stored: integers(300, 300),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 0, Relevance: engine.AnyValueFails},
						{Label: "y", Value: 501, Relevance: engine.ValueMatters, NearestPassing: 500},
					},
					token:  "prop1:AAAA9QM",
					runs:   45,
					calls:  47,
					digest: "2d4c18435a318a457521e59a8b030cc6a3b54a563723ef7763b0e75e54e6c8af",
				},
			},
			{
				name: "moves part of a value when the whole of it passes",
				p: func(c *engine.Case) string {
					x, y := engine.Draw(c, small, "x"), engine.Draw(c, small, "y")
					return failsWhen(x+y > 500 && x >= 100, "partial")
				},
				stored: integers(300, 300),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 100, Relevance: engine.ValueMatters, NearestPassing: 99},
						{Label: "y", Value: 401, Relevance: engine.ValueMatters, NearestPassing: 400},
					},
					token:  "prop1:AGQAkQM",
					runs:   66,
					calls:  68,
					digest: "b13ea0a2f68919566f363c93a753bf0bc269a2c8db00c58c6390da038515df79",
				},
			},
			{
				name: "moves no value up to an integer without room below it",
				p: func(c *engine.Case) string {
					x, y := engine.Draw(c, signed, "x"), engine.Draw(c, signed, "y")
					return failsWhen(x <= -10 && y <= -95, "room")
				},
				stored: integers(-10, -95),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: -10, Relevance: engine.ValueMatters, NearestPassing: -9},
						{Label: "y", Value: -95, Relevance: engine.ValueMatters, NearestPassing: -94},
					},
					token:  "prop1:AQoBXw",
					runs:   27,
					calls:  29,
					digest: "5952f35b707c7c13cc7d63c63f7e1aaadba7d734fe98821ad9da0d0b8bb8bc58",
				},
			},
			{
				name: "steps a count alone once a later pass lowered what it is compared with",
				p: func(c *engine.Case) string {
					x, s := engine.Draw(c, hundred, "x"), engine.Draw(c, firstByte, "s")
					return failsWhen(s[0] >= 1 && x >= int(s[0]), "step")
				},
				stored: []choice.Choice{unsigned(50), sequence(50)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 1, Relevance: engine.AnyValueFails},
						{Label: "s", Value: []byte{1}, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEDAQE",
					runs:   33,
					calls:  35,
					digest: "daa1a5a2b5a4b2aa4bfde855fc378ad97a6051cd604d267862a88a0d8a6ee4f6",
				},
			},
			{
				name: "sets two equal floats to their target together",
				p: func(c *engine.Case) string {
					return failsWhen(engine.Draw(c, unit, "x") == engine.Draw(c, unit, "y"), "twins")
				},
				stored: []choice.Choice{float(7.5), float(7.5)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 0.0, Relevance: engine.ValueMatters},
						{Label: "y", Value: 0.0, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AgAAAAAAAAAAAgAAAAAAAAAA",
					runs:   5,
					calls:  7,
					digest: "3fbaffa3a6c178942dadc4efe2bfe7d7ed986adfc52415b340764d67985318fb",
				},
			},
			{
				name: "sets two equal floats with a choice between them to their target together",
				p: func(c *engine.Case) string {
					x, n, y := engine.Draw(c, unit, "x"), engine.Draw(c, digit, "n"), engine.Draw(c, unit, "y")
					return failsWhen(x == y && n > 5, "separated")
				},
				stored: []choice.Choice{float(7.5), unsigned(9), float(7.5)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 0.0, Relevance: engine.ValueMatters},
						{Label: "n", Value: 6, Relevance: engine.ValueMatters, NearestPassing: 5},
						{Label: "y", Value: 0.0, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AgAAAAAAAAAAAAYCAAAAAAAAAAA",
					runs:   29,
					calls:  31,
					digest: "03b78cbfc36fd34d29e23988f8b8cd6e06227bda3b1e2b06673bf27612dde907",
				},
			},
			{
				name: "lowers two equal values together",
				p: func(c *engine.Case) string {
					x, y := engine.Draw(c, small, "x"), engine.Draw(c, small, "y")
					return failsWhen(x == y && x > 0, "equal")
				},
				stored: integers(400, 400),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 1, Relevance: engine.ValueMatters, NearestPassing: 0},
						{Label: "y", Value: 1, Relevance: engine.ValueMatters, NearestPassing: 0},
					},
					token:  "prop1:AAEAAQ",
					runs:   59,
					calls:  61,
					digest: "9e27bd5891bdfd4d9d947c4bd475c97a51bf71224075d6cfe823d3633b0d2d83",
				},
			},
			{
				name: "lowers a value that occurs once apart from the equal pair",
				p: func(c *engine.Case) string {
					x, y := engine.Draw(c, small, "x"), engine.Draw(c, small, "y")
					engine.Draw(c, small, "z")
					return failsWhen(x == y && x > 0, "equal")
				},
				stored: integers(400, 400, 7),
				want: reference{
					explanation: []engine.Explained{
						{Label: "x", Value: 1, Relevance: engine.ValueMatters, NearestPassing: 0},
						{Label: "y", Value: 1, Relevance: engine.ValueMatters, NearestPassing: 0},
						{Label: "z", Value: 0, Relevance: engine.AnyValueFails},
					},
					token:  "prop1:AAEAAQAA",
					runs:   78,
					calls:  80,
					digest: "65ed9e5ad888c6ef94662827d87682d11b520ca2c4423553f142541af413bf56",
				},
			},
			{
				name:   "deletes an element before the index that it lowers",
				p:      indexed,
				stored: integers(1, 0, 1, 60, 0, 1),
				want: reference{
					explanation: []engine.Explained{
						{Label: "xs", Value: []int{51}, Relevance: engine.ValueMatters},
						{Label: "i", Value: 0, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEAMwAAAAA",
					runs:   64,
					calls:  66,
					digest: "ba1851f50ee9ad8c7b302d92f9ae46f8835fcb013e86e11da34ef094448ee98b",
				},
			},
			{
				name:   "deletes a byte before the index that it lowers",
				p:      high(anyBytes),
				stored: []choice.Choice{sequence(1, 250), unsigned(1)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "b", Value: []byte{201}, Relevance: engine.ValueMatters},
						{Label: "i", Value: 0, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AwHJAQAA",
					runs:   42,
					calls:  44,
					digest: "478873bf08b251281e5d19557cb37acd861a9e4211b20f7e3446775f578a3421",
				},
			},
			{
				name:   "keeps a byte string at its minimum length before the index",
				p:      high(twoBytes),
				stored: []choice.Choice{sequence(1, 250), unsigned(1)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "b", Value: []byte{0, 201}, Relevance: engine.ValueMatters},
						{Label: "i", Value: 1, Relevance: engine.ValueMatters, NearestPassing: 0},
					},
					token:  "prop1:AwIAyQEAAQ",
					runs:   26,
					calls:  28,
					digest: "861a87b5655ab8ca1e90754579d309aa6c9ea7f3712bfef704b789fdb7f6cc2b",
				},
			},
			{
				name: "deletes no empty span before the index that it lowers",
				p: func(c *engine.Case) string {
					engine.Draw(c, five, "j")
					return indexed(c)
				},
				stored: integers(1, 0, 1, 60, 0, 1),
				want: reference{
					explanation: []engine.Explained{
						{Label: "j", Value: 5, Relevance: engine.Untested},
						{Label: "xs", Value: []int{51}, Relevance: engine.ValueMatters},
						{Label: "i", Value: 0, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEAMwAAAAA",
					runs:   64,
					calls:  66,
					digest: "3edc5434de5b96a6569f0470e4e4cf7d00a8b81091f44c4a2dddd4b1b16d62ef",
				},
			},
			{
				name: "runs another round after a deletion that lowers an index",
				p: func(c *engine.Case) string {
					values, index := engine.Draw(c, percent, "xs"), engine.Draw(c, digit, "i")
					return failsWhen(index < len(values) && values[index] > 50+10*index, "offset")
				},
				stored: integers(1, 0, 1, 70, 0, 1),
				want: reference{
					explanation: []engine.Explained{
						{Label: "xs", Value: []int{51}, Relevance: engine.ValueMatters},
						{Label: "i", Value: 0, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEAMwAAAAA",
					runs:   70,
					calls:  72,
					digest: "54b31da56a7b5a618679d8213f9a1b93a91f01153d9a835edf2c21bc23773604",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, trace := traced(tt.p, settled(tt.stored...))
				matchesReference(t, got, trace, tt.want)
			})
		}
	})
}
