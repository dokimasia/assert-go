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
// alone, in pairs, in groups and with the spans they size, each through a
// run from a stored failing case that the pass shrinks, pinned to what the
// definition's executable reference reports.
func TestIntegers(t *testing.T) {
	t.Parallel()

	digit, hundred := engine.Integer(0, 9), engine.Integer(0, 100)
	small, wide := engine.Integer(0, 1000), engine.Integer(0, 1_000_000_000)
	signed, positive := engine.Integer(-100, 100), engine.Integer(1, 1000)
	signedList := engine.List(signed, unbounded(t, 0))
	unit := engine.Float(0.0, 10.0, choice.ExcludeNaN)
	firstByte := engine.Bytes(unbounded(t, 1))
	exact := make([]engine.Generator[[]int], 10)
	for n := range exact {
		exact[n] = engine.List(digit, sizes(t, n, n))
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
					runs:        25,
					calls:       27,
					digest:      "fd6afff264082c65c27e455a58edfc1c6ecfe28ec1c903666bbb59980f2c78f4",
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
					runs:   34,
					calls:  36,
					digest: "8ff7880f473da01f14b0df2f0d62b45db693224ac7b2ac3f7e1e22da1ae2db2d",
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
					runs:   59,
					calls:  61,
					digest: "e0fbda3c85ea0c955324748047738a316478723c0d602e8b520f7a6e0813371a",
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
					runs:   18,
					calls:  20,
					digest: "5a5bee4418faf95bb8472958b7087c5a7262d577f3035d1dfbdc70c49df9447f",
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
					runs:   39,
					calls:  41,
					digest: "9e5e12381e2b177a5d4bcb97e67bf454b2e04ce81c5052cd3d8c18cc47e282d3",
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
					runs:   45,
					calls:  47,
					digest: "0b80b7c070ed37107c305eae9984105e247668be31b511dc34fc4c198032133f",
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
					runs:   44,
					calls:  46,
					digest: "12b07d826d41f89a1e478e3269fad4b5eac8a2d47e70cb721c4eeb0560ed5b77",
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
					runs:   65,
					calls:  67,
					digest: "5685d78145826a880d24d26f07cdfcc6c9782c0ad2289127e6c0f32cb558f833",
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
					runs:   26,
					calls:  28,
					digest: "e116b830ad59a210320f441d763aaa32cf9b415c90998ef09e8007b171ab8b12",
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
					runs:   28,
					calls:  30,
					digest: "286e1be7a9c0ad9e3524f032f2fac9ef02c2689436d866c7ceb616eda12e22c0",
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
					runs:   58,
					calls:  60,
					digest: "95e30156e6084d09f406e583fb3495cebe716f616a8d90f4d0a090b84ecd3701",
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
					runs:   77,
					calls:  79,
					digest: "1530c8527b1b858543434745494914ba9639fbd30c3062d8a19f456db0f30507",
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
