// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// edgeValues is one call's value of each generator that the edge test
// draws, in draw order.
type edgeValues struct {
	// signed is an integer in [-5, 5].
	signed int
	// above is an integer in [3, 9], whose target has no value below it.
	above int
	// below is an integer in [-9, -3], whose target has no value above it.
	below int
	// unit is a float in [0.25, 0.75].
	unit float64
	// one is a float in [1, 1].
	one float64
	// word is a string over "ab" of at most three characters.
	word string
	// letter is a string over "a" of at most three characters.
	letter string
	// pair is a byte string of two bytes.
	pair []byte
	// maybe is an optional integer in [1, 3].
	maybe *int
	// pick is one of "x" and "y".
	pick string
	// digits is a list of digits.
	digits []int
	// empty is a list of digits with a maximum length of 0.
	empty []int
}

// TestEdge checks the value that each edge case gives every kind of
// choice, through a run of four cases of seed 7 whose seven calls are the
// simplest case, two random cases and the four edge cases, pinned to what
// the definition's executable reference reports.
func TestEdge(t *testing.T) {
	t.Parallel()

	signed, above, below := engine.Integer(-5, 5), engine.Integer(3, 9), engine.Integer(-9, -3)
	unit, one := engine.Float(0.25, 0.75, choice.ExcludeNaN), engine.Float(1.0, 1.0, choice.ExcludeNaN)
	word, letter := engine.StringOver("ab", sizes(t, 0, 3)), engine.StringOver("a", sizes(t, 0, 3))
	pair, maybe := engine.Bytes(sizes(t, 2, 2)), engine.Optional(engine.Integer(1, 3))
	pick := engine.SampledFrom("x", "y")
	digit := engine.Integer(0, 9)
	digits, empty := engine.List(digit, unbounded(t, 0)), engine.List(digit, sizes(t, 0, 0))

	s := settled()
	s.Cases = 4
	var calls []edgeValues
	got, trace := recorded(func(c *engine.Case) {
		calls = append(calls, edgeValues{
			signed: engine.Draw(c, signed, "signed"),
			above:  engine.Draw(c, above, "above"),
			below:  engine.Draw(c, below, "below"),
			unit:   engine.Draw(c, unit, "unit"),
			one:    engine.Draw(c, one, "one"),
			word:   engine.Draw(c, word, "word"),
			letter: engine.Draw(c, letter, "letter"),
			pair:   engine.Draw(c, pair, "pair"),
			maybe:  engine.Draw(c, maybe, "maybe"),
			pick:   engine.Draw(c, pick, "pick"),
			digits: engine.Draw(c, digits, "digits"),
			empty:  engine.Draw(c, empty, "empty"),
		})
	}, s)

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the four edge cases after the random cases stop", func(t *testing.T) {
			t.Parallel()
			want := engine.Result{Outcome: engine.Passed, Cases: 7, Seed: referenceSeed}
			assert.Equal(t, summary(got), want, "every case passed")
			assert.Length(t, calls, 7, "the simplest case, two random cases and four edge cases")
			assert.Equal(t, digestOf(trace), "7341c6fd8b92db105f53d08980f9781958e2021b5d24e8edeaaf04eaecd123af",
				"the choices of every call")
		})

		tests := []struct {
			name string
			call int
			want edgeValues
		}{
			{
				name: "gives every value choice its lower bound in the first edge case",
				call: 2,
				want: edgeValues{
					signed: -5, above: 3, below: -9, unit: 0.25, one: 1, word: "a", letter: "a",
					pair: []byte{0, 0}, maybe: new(1), pick: "x", digits: []int{0}, empty: []int{},
				},
			},
			{
				name: "gives every value choice its upper bound in the second edge case",
				call: 4,
				want: edgeValues{
					signed: 5, above: 9, below: -3, unit: 0.75, one: 1, word: "b", letter: "a",
					pair: []byte{0xff, 0xff}, maybe: new(3), pick: "x", digits: []int{9}, empty: []int{},
				},
			},
			{
				name: "gives every value choice the value above its target in the third edge case",
				call: 5,
				want: edgeValues{
					signed: 1, above: 4, below: -3, unit: 0.5000000000000001, one: 1, word: "b", letter: "a",
					pair: []byte{1, 1}, maybe: new(2), pick: "x", digits: []int{1}, empty: []int{},
				},
			},
			{
				name: "gives every value choice the value below its target in the fourth edge case",
				call: 6,
				want: edgeValues{
					signed: -1, above: 3, below: -4, unit: 0.49999999999999994, one: 1, word: "a", letter: "a",
					pair: []byte{0, 0}, maybe: new(1), pick: "x", digits: []int{0}, empty: []int{},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, calls[tt.call], tt.want, "the value of each generator")
			})
		}
	})
}
