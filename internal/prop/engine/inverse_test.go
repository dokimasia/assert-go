// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocations of running a generator backwards, measured.
const (
	// invertAllocs are the allocations of Invert of a digit: the step and
	// the choice, and the replay that checks them, with its body, its case,
	// its goroutine, its recorder and its record.
	invertAllocs = 16
	// inverseAllocs are the allocations of Inverse of a digit: its one step.
	inverseAllocs = 1
	// decodeCaseAllocs are the allocations of a whole replayed case that
	// decodes a digit: those of a case that makes a choice in a span, and
	// the replayed record.
	decodeCaseAllocs = 8
	// collectionStepsAllocs are the allocations of CollectionSteps of one
	// item: the growth of its steps.
	collectionStepsAllocs = 3
	// listItemsAllocs are the allocations of ListItems of two integers: the
	// list, and a copy of each element that reflect boxes, with the
	// interface of the slice it reads.
	listItemsAllocs = 4
	// sameValueAllocs are the allocations of SameValue of two lists of two
	// integers: the interfaces of the two lists, and for the canonical key of
	// each the growth of its bytes, the path that it enters and its text.
	sameValueAllocs = 12
)

// point is a struct whose typed literal is a map of its fields.
type point struct {
	// X is the first coordinate.
	X int
	// Y is the second coordinate.
	Y int
}

// TestInverse checks the choices that each generator runs backwards to,
// pinned from the definition's vectors where the definition states them,
// the order that picks one sequence of choices, and each value that a
// generator does not produce.
func TestInverse(t *testing.T) {
	t.Parallel()

	digit, double := engine.Integer(0, 9), func(v int) int { return 2 * v }
	wide := engine.Integer(5, 15)
	digits := engine.Integer[int64](0, 9)
	upToThree := sizes(t, 0, 3)
	tree := engine.Recursive(engine.Erase(engine.List(digit, unbounded(t, 0))),
		func(self engine.Generator[any]) engine.Generator[any] {
			return engine.Erase(engine.List(self, upToThree))
		},
		engine.DefaultMaxLeaves)
	seven, twelve := 7, 12
	var noDigit *int

	t.Run("Invert", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give func() ([]choice.Choice, error)
			want []choice.Choice
		}{
			{
				name: "returns an integer as its one choice",
				give: func() ([]choice.Choice, error) { return engine.Invert(digit, 7) },
				want: integers(7),
			},
			{
				name: "returns a negative integer as its one choice",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Integer(-5, 5), -3) },
				want: integers(-3),
			},
			{
				name: "returns an unsigned integer as its one choice",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Integer[uint8](0, 255), 200) },
				want: integers(200),
			},
			{
				name: "returns a duration's nanoseconds as its one choice",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Duration(0, time.Second), 1000) },
				want: integers(1000),
			},
			{
				name: "returns a float as its one choice",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Float(-2.0, 2.0, choice.ExcludeNaN), 1.5)
				},
				want: []choice.Choice{float(1.5)},
			},
			{
				name: "returns any NaN as the NaN of a float that admits it",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(
						engine.Float(0.0, 1.0, choice.AdmitNaN),
						math.Float64frombits(0x7ff8000000000001),
					)
				},
				want: []choice.Choice{float(math.NaN())},
			},
			{
				name: "returns a float32 that a literal states as a float64",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Float[float32](0, 2, choice.ExcludeNaN)), any(0.5))
				},
				want: []choice.Choice{float(0.5)},
			},
			{
				name: "returns 1 for true",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Boolean(1, 2), true) },
				want: integers(1),
			},
			{
				name: "returns no choice for the value of a just",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Just(4), 4) },
				want: integers(),
			},
			{
				name: "returns no choice for the literal's value of a just of a struct",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Just(point{X: 1, Y: 2})), any(map[any]any{"Y": 2, "X": 1}))
				},
				want: integers(),
			},
			{
				name: "returns the first index of an equal value",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.SampledFrom("a", "b", "a"), "a") },
				want: integers(0),
			},
			{
				name: "returns the index of a value that a literal states as another type",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.SampledFrom[int64](1, 2, 3)), any(2))
				},
				want: integers(1),
			},
			{
				name: "returns the first alternative that produces the value",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.OneOf(digit, wide), 7) },
				want: integers(0, 7),
			},
			{
				name: "returns a later alternative for a value only it produces",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.OneOf(digit, wide), 12) },
				want: integers(1, 12),
			},
			{
				name: "returns absent at the outer optional for nil",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Optional(engine.Optional(digit))), nil)
				},
				want: integers(0),
			},
			{
				name: "returns absent at the inner optional for a pointer to nil",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Optional(engine.Optional(digit)), &noDigit)
				},
				want: integers(1, 0),
			},
			{
				name: "returns present and the value for a pointer",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Optional(digit), &seven) },
				want: integers(1, 7),
			},
			{
				name: "returns present and the value for the value itself",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Erase(engine.Optional(digits)), any(7)) },
				want: integers(1, 7),
			},
			{
				name: "returns absent for a nil slice",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Optional(engine.List(digit, upToThree))), any([]int(nil)))
				},
				want: integers(0),
			},
			{
				name: "returns a list's flags and elements in order",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.List(digit, upToThree), []int{7, 3}) },
				want: integers(1, 7, 1, 3, 0),
			},
			{
				name: "returns the list of an array of another integer type",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.List(digits, upToThree)), any([2]int{7, 3}))
				},
				want: integers(1, 7, 1, 3, 0),
			},
			{
				name: "returns the distinct elements of a unique list in order",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.UniqueList(digit, upToThree), []int{7, 3})
				},
				want: integers(1, 7, 1, 3, 0),
			},
			{
				name: "returns a dict's entries in the shortlex order of their choices",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Dict(digit, digit, unbounded(t, 0)), map[int]int{4: 0, 1: 9})
				},
				want: integers(1, 1, 9, 1, 4, 0, 0),
			},
			{
				name: "returns six entries in the shortlex order of their choices",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Dict(digit, digit, unbounded(t, 0)),
						map[int]int{5: 0, 3: 1, 9: 0, 1: 2, 7: 0, 2: 0})
				},
				want: integers(1, 1, 2, 1, 2, 0, 1, 3, 1, 1, 5, 0, 1, 7, 0, 1, 9, 0, 0),
			},
			{
				name: "returns the entry of fewer choices first",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Dict(digit, engine.Optional(digit), unbounded(t, 0)),
						map[int]*int{1: &seven, 9: nil})
				},
				want: integers(1, 9, 0, 1, 1, 1, 7, 0),
			},
			{
				name: "returns the indices of a string's characters in the default alphabet",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.String(sizes(t, 0, 4)), "a0") },
				want: []choice.Choice{sequence(10, 0)},
			},
			{
				name: "returns the indices of a string's characters in its alphabet",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.StringOver("xyz", sizes(t, 1, 1)), "z")
				},
				want: []choice.Choice{sequence(2)},
			},
			{
				name: "returns the bytes in order",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Bytes(unbounded(t, 0)), []byte{0x00, 0xff})
				},
				want: []choice.Choice{sequence(0, 255)},
			},
			{
				name: "returns the smallest index at each swap",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Permutation(1, 1, 2), []int{1, 2, 1})
				},
				want: integers(0, 2),
			},
			{
				name: "returns each position's own index for the stated order",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Permutation(1, 2, 3), []int{1, 2, 3})
				},
				want: integers(0, 1),
			},
			{
				name: "returns no choice for the permutation of no value",
				give: func() ([]choice.Choice, error) { return engine.Invert(engine.Permutation[int](), []int{}) },
				want: integers(),
			},
			{
				name: "returns the base before the extension",
				give: func() ([]choice.Choice, error) { return engine.Invert(tree, any([]any{})) },
				want: integers(0, 0),
			},
			{
				name: "returns the extension for a value the base does not produce",
				give: func() ([]choice.Choice, error) { return engine.Invert(tree, any([]any{[]any{}})) },
				want: integers(1, 1, 0, 0, 0),
			},
			{
				name: "returns the source's choice of a value the filter keeps",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(digit.Filter(func(v int) bool { return v%2 == 0 }), 4)
				},
				want: integers(4),
			},
			{
				name: "returns the source's choice of the value that back returns",
				give: func() ([]choice.Choice, error) { return engine.Invert(digit.MapBack(double, halve), 8) },
				want: integers(4),
			},
			{
				name: "returns the source's choice of a source's value that MapBack does not take",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(digit.MapBack(strconv.Itoa, strconv.Atoi)), any(4))
				},
				want: integers(4),
			},
			{
				name: "returns the steps of a generator built with NewInvertible",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.NewInvertible("pair", decodePair, invertPair), [2]uint64{3, 4})
				},
				want: integers(3, 4),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give()
				assert.NoError(t, err, "the generator produces the value")
				assert.True(t, sameChoices(got, tt.want), "the choices")
			})
		}

		even := digit.Filter(func(v int) bool { return v%2 == 0 })
		budget := engine.Recursive(engine.Erase(digit),
			func(self engine.Generator[any]) engine.Generator[any] {
				return engine.Erase(engine.List(self, upToThree))
			},
			1)
		short := engine.NewInvertible("pair", decodePair, func(v any) ([]engine.Step, [2]uint64, error) {
			steps, pair, err := invertPair(v)
			return steps[:1], pair, err
		})
		refusals := []struct {
			name       string
			give       func() ([]choice.Choice, error)
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns an error for an integer outside the bounds",
				give:       func() ([]choice.Choice, error) { return engine.Invert(digit, 12) },
				wantReason: "12 is outside [0, 9]",
			},
			{
				name:       "returns an error for an integer's value of another kind",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.Erase(digit), any("7")) },
				wantReason: "7 is no integer",
			},
			{
				name: "returns an error for NaN where the float admits none",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Float(-2.0, 2.0, choice.ExcludeNaN), math.NaN())
				},
				wantReason: "NaN is outside [-2, 2] of width 64",
			},
			{
				name: "returns an error for a float's value of another kind",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Float(0.0, 1.0, choice.ExcludeNaN)), any(1))
				},
				wantReason: "1 is no float",
			},
			{
				name: "returns an error for a boolean's value of another kind",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Boolean(1, 2)), any(1))
				},
				wantReason: "1 is no bool",
			},
			{
				name:       "returns an error for another value than the just's",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.Just(4), 5) },
				wantReason: "5 is not 4",
			},
			{
				name:       "returns an error for a value that none of the sampled values equals",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.SampledFrom("a", "b"), "c") },
				wantReason: "c is none of the 2 values",
			},
			{
				name: "returns an error for a value of another type without a literal",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.SampledFrom(1, 2)), any(complex(1, 2)))
				},
				wantReason: "(1+2i) is none of the 2 values",
			},
			{
				name: "returns an error for a value whose literal decodes to no Go value",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.SampledFrom(1)), any(map[[1]int]int{{1}: 1}))
				},
				wantReason: "map[[1]:1] is none of the 1 values",
			},
			{
				name: "returns an error for a value whose literal decodes to no Go value, against nil",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.SampledFrom[any](nil), any(map[[1]int]int{{1}: 1}))
				},
				wantReason: "map[[1]:1] is none of the 1 values",
			},
			{
				name:       "returns an error for a value that no alternative produces",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.OneOf(digit, wide), 20) },
				wantReason: "none of the 2 alternatives produces 20",
			},
			{
				name:       "returns an error for a present value that the optional's generator does not produce",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.Optional(digit), &twelve) },
				wantReason: "12 is outside [0, 9]",
			},
			{
				name: "returns an error for a list's value of another kind",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.List(digit, upToThree)), any("x"))
				},
				wantReason: "x is no list",
			},
			{
				name: "returns an error at the element that the list's generator does not produce",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.List(digit, upToThree), []int{1, 12})
				},
				wantPath:   fault.Path{fault.Index(1)},
				wantReason: "12 is outside [0, 9]",
			},
			{
				name: "returns an error for a list longer than its sizes",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.List(digit, upToThree), []int{1, 2, 3, 4})
				},
				wantReason: "4 elements are outside the sizes of the collection",
			},
			{
				name: "returns an error at the element of a unique list that repeats an earlier one",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.UniqueList(digit, upToThree), []int{1, 2, 1})
				},
				wantPath:   fault.Path{fault.Index(2)},
				wantReason: "the element repeats element 0",
			},
			{
				name: "returns an error for a dict's value of another kind",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Dict(digit, digit, upToThree)), any("x"))
				},
				wantReason: "x is no map",
			},
			{
				name: "returns an error at a key that the dict's keys do not produce",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Dict(digit, digit, upToThree), map[int]int{12: 0})
				},
				wantPath:   fault.Path{fault.Key(12)},
				wantReason: "the generator of keys produces no such key",
			},
			{
				name: "returns an error at the key of a value that the dict's values do not produce",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Dict(digit, digit, upToThree), map[int]int{1: 12})
				},
				wantPath:   fault.Path{fault.Key(1)},
				wantReason: "12 is outside [0, 9]",
			},
			{
				name: "returns an error for two keys that decode to one key",
				give: func() ([]choice.Choice, error) {
					keys := engine.Erase(digits)
					return engine.Invert(engine.Dict(keys, keys, upToThree), map[any]any{1: 0, int64(1): 0})
				},
				wantReason: "two keys decode to the key 1",
			},
			{
				name: "returns an error for a dict larger than its sizes",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Dict(digit, digit, sizes(t, 0, 1)), map[int]int{1: 0, 2: 0})
				},
				wantReason: "2 elements are outside the sizes of the collection",
			},
			{
				name: "returns an error for a string's value of another kind",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.String(upToThree)), any(7))
				},
				wantReason: "7 is no string of UTF-8",
			},
			{
				name:       "returns an error for a string that is not UTF-8",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.String(upToThree), "\xff") },
				wantReason: "\xff is no string of UTF-8",
			},
			{
				name: "returns an error for a character outside the alphabet",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.StringOver("xyz", upToThree), "xa")
				},
				wantReason: `"xa" has the character 'a' outside the alphabet`,
			},
			{
				name:       "returns an error for a string longer than its sizes",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.String(sizes(t, 0, 1)), "ab") },
				wantReason: "2 elements are outside the sizes of the sequence",
			},
			{
				name: "returns an error for a byte string's value of another kind",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(engine.Bytes(upToThree)), any("ab"))
				},
				wantReason: "ab is no []byte",
			},
			{
				name: "returns an error for bytes longer than their sizes",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Bytes(sizes(t, 0, 1)), []byte{1, 2})
				},
				wantReason: "2 elements are outside the sizes of the sequence",
			},
			{
				name:       "returns an error for a permutation of another length",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.Permutation(1, 2), []int{1}) },
				wantReason: "[1] is no list of 2 values",
			},
			{
				name: "returns an error at an element of a permutation that is none of the values",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Permutation(1, 2), []int{3, 1})
				},
				wantPath:   fault.Path{fault.Index(0)},
				wantReason: "the element is none of the values left to order",
			},
			{
				name: "returns an error at a last element of a permutation that is not the value left",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Permutation(1, 2), []int{1, 3})
				},
				wantPath:   fault.Path{fault.Index(1)},
				wantReason: "the element is not the value left to order",
			},
			{
				name:       "returns an error at the one element of a permutation that is not the value",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.Permutation(1), []int{2}) },
				wantPath:   fault.Path{fault.Index(0)},
				wantReason: "the element is not the value left to order",
			},
			{
				name:       "returns an error for a value that the filter rejects",
				give:       func() ([]choice.Choice, error) { return engine.Invert(even, 3) },
				wantReason: "the filter rejects 3",
			},
			{
				name:       "returns an error for a value that the filter's source does not produce",
				give:       func() ([]choice.Choice, error) { return engine.Invert(even, 12) },
				wantReason: "12 is outside [0, 9]",
			},
			{
				name:       "returns an error for a value that neither the base nor the extension produces",
				give:       func() ([]choice.Choice, error) { return engine.Invert(tree, any("x")) },
				wantReason: "neither the base nor the extension produces x",
			},
			{
				name:       "returns an error for a recursive value of more leaves than its bound",
				give:       func() ([]choice.Choice, error) { return engine.Invert(budget, any([]any{1, []any{2}})) },
				wantReason: "the choices of [1 [2]] decode to [1 1]",
			},
			{
				name:       "returns an error for a value that back refuses",
				give:       func() ([]choice.Choice, error) { return engine.Invert(digit.MapBack(double, halve), 7) },
				wantReason: "the inverse of map refuses the value",
			},
			{
				name:       "returns an error for a value whose source value the source does not produce",
				give:       func() ([]choice.Choice, error) { return engine.Invert(digit.MapBack(double, halve), 24) },
				wantReason: "12 is outside [0, 9]",
			},
			{
				name:       "returns an error for a mapped generator",
				give:       func() ([]choice.Choice, error) { return engine.Invert(digit.Map(double), 8) },
				wantReason: "map has no inverse",
			},
			{
				name: "returns an error for an erased generator without an inverse",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.Erase(digit.Map(double)), any(8))
				},
				wantReason: "map has no inverse",
			},
			{
				name:       "returns an error for a bound generator",
				give:       func() ([]choice.Choice, error) { return engine.Invert(digit.Bind(engine.Just[int]), 8) },
				wantReason: "bind has no inverse",
			},
			{
				name: "returns an error for a composite",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(
						engine.Composite(func(c *engine.Case) int { return engine.Draw(c, digit, drawn) }),
						8,
					)
				},
				wantReason: "composite has no inverse",
			},
			{
				name: "returns an error for a generator built with NewGenerator",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.NewGenerator("pair", decodePair), [2]uint64{3, 4})
				},
				wantReason: "pair has no inverse",
			},
			{
				name:       "returns an error for steps that the decode runs past",
				give:       func() ([]choice.Choice, error) { return engine.Invert(short, [2]uint64{3, 4}) },
				wantReason: "the choices of [3 4] decode to no value",
			},
			{
				name: "returns an error for steps whose inverse states another value",
				give: func() ([]choice.Choice, error) {
					swapped := engine.NewInvertible("pair", decodePair, func(v any) ([]engine.Step, [2]uint64, error) {
						steps, pair, err := invertPair(v)
						return steps, [2]uint64{pair[1], pair[0]}, err
					})
					return engine.Invert(swapped, [2]uint64{3, 4})
				},
				wantReason: "the choices of [3 4] decode to [3 4]",
			},
			{
				name: "returns an error for steps that decode to another value",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(engine.NewInvertible("pair", decodePair, invertPair), [2]uint64{12, 4})
				},
				wantReason: "the choices of [12 4] decode to [0 4]",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give()
				assert.ErrorIs(t, err, engine.ErrCannotInvert, "no choices decode to the value")
				f := assert.ErrorAs[*fault.Error](t, err, "a fault")
				assert.Equal(t, f.Path, tt.wantPath, "the part that no choice produces")
				assert.Equal(t, f.Reason, tt.wantReason, "why no choice produces it")
				assert.Nil(t, got, "no choices")
			})
		}

		t.Run("returns an error caused by the error that back returns", func(t *testing.T) {
			t.Parallel()
			_, err := engine.Invert(digit.MapBack(double, halve), 7)
			assert.ErrorIs(t, err, errOdd, "the error of back")
		})

		t.Run("returns an error at a key caused by the fault of the dict's keys", func(t *testing.T) {
			t.Parallel()
			_, err := engine.Invert(engine.Dict(digit, digit, upToThree), map[int]int{12: 0})
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			cause := assert.ErrorAs[*fault.Error](t, f.Err, "the fault of the keys")
			assert.Equal(t, cause.Reason, "12 is outside [0, 9]", "why the keys produce no such key")
		})

		mapped := digit.Map(func(v int) int { return v })
		unknown := []struct {
			name       string
			give       func() ([]choice.Choice, error)
			wantReason string
		}{
			{
				name:       "returns no inverse for a generator that applies a function",
				give:       func() ([]choice.Choice, error) { return engine.Invert(mapped, 4) },
				wantReason: "map has no inverse",
			},
			{
				name:       "returns no inverse for a list of a generator without one",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.List(mapped, upToThree), []int{4}) },
				wantReason: "map has no inverse",
			},
			{
				name:       "returns no inverse for an optional of a generator without one",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.Optional(mapped), &seven) },
				wantReason: "map has no inverse",
			},
			{
				name: "returns no inverse for a filter of a generator without one",
				give: func() ([]choice.Choice, error) {
					return engine.Invert(mapped.Filter(func(int) bool { return true }), 4)
				},
				wantReason: "map has no inverse",
			},
			{
				name:       "returns no inverse for a one-of whose alternative without one might produce the value",
				give:       func() ([]choice.Choice, error) { return engine.Invert(engine.OneOf(digit, mapped), 12) },
				wantReason: "none of the 2 alternatives produces 12, and one of them has no inverse",
			},
			{
				name: "returns no inverse for a recursive generator whose base has none",
				give: func() ([]choice.Choice, error) {
					base := engine.Erase(mapped)
					shallow := engine.Recursive(base, func(self engine.Generator[any]) engine.Generator[any] {
						return engine.Erase(engine.List(self, upToThree))
					}, engine.DefaultMaxLeaves)
					return engine.Invert(shallow, any("x"))
				},
				wantReason: "neither the base nor the extension produces x, and one of them has no inverse",
			},
		}
		for _, tt := range unknown {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give()
				assert.ErrorIs(t, err, engine.ErrNoInverse, "no inverse on the way")
				assert.ErrorIs(t, err, engine.ErrCannotInvert, "and no choices decode to the value")
				assert.Equal(t, assert.ErrorAs[*fault.Error](t, err, "a fault").Reason, tt.wantReason,
					"which generator has no inverse")
				assert.Nil(t, got, "no choices")
			})
		}

		t.Run("runs a one-of back through an alternative after one without an inverse", func(t *testing.T) {
			t.Parallel()
			got, err := engine.Invert(engine.OneOf(mapped, digit), 3)
			assert.NoError(t, err, "the digit produces 3")
			assert.True(t, sameChoices(got, integers(1, 3)), "the index of the digit, then 3")
		})

		t.Run("returns a value outside every alternative with an inverse as outside the domain", func(t *testing.T) {
			t.Parallel()
			_, err := engine.Invert(engine.OneOf(digit, wide), 20)
			assert.ErrorIs(t, err, engine.ErrCannotInvert, "no alternative produces 20")
			assert.ErrorIsNot(t, err, engine.ErrNoInverse, "every alternative has an inverse")
		})
	})
}

// TestInverseParts checks the parts that a generator built outside this
// package runs backwards with: its steps, a collection's, their order, and
// the values it reads.
func TestInverseParts(t *testing.T) {
	t.Parallel()

	digit := engine.Integer(0, 9)

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the value of the choices, without a draw", func(t *testing.T) {
			t.Parallel()
			var got int
			e := engine.Replay(func(c *engine.Case) { got = digit.Decode(c) }, integers(7), nil)
			assert.Equal(t, got, 7, "the value")
			assert.Empty(t, e.Case.Draws(), "no draw")
		})
	})

	t.Run("Inverse", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the steps and the value, without a replay", func(t *testing.T) {
			t.Parallel()
			steps, got, err := digit.Inverse(any(7))
			assert.NoError(t, err, "the digit runs back")
			assert.Equal(t, [2]any{got, len(steps)}, [2]any{7, 1}, "the value and its one step")
			assert.True(t, steps[0].Value.Equal(integers(7)[0]), "the step's choice")
		})

		t.Run("returns an error for a generator without an inverse", func(t *testing.T) {
			t.Parallel()
			_, _, err := digit.Map(func(v int) int { return v }).Inverse(any(7))
			assert.ErrorIs(t, err, engine.ErrCannotInvert, "no choices decode to the value")
			assert.Equal(t, assert.ErrorAs[*fault.Error](t, err, "a fault").Reason, "map has no inverse",
				"a map has no inverse")
		})
	})

	t.Run("IntegerStep", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the step of an admitted integer", func(t *testing.T) {
			t.Parallel()
			step, err := engine.IntegerStep(digitRange, choice.UintOf(3))
			assert.NoError(t, err, "the bounds admit 3")
			assert.Equal(t, step.Bounds, choice.OfInteger(digitRange), "the bounds")
			assert.True(t, step.Value.Equal(integers(3)[0]), "the choice")
		})

		t.Run("returns an error that names an integer outside the bounds", func(t *testing.T) {
			t.Parallel()
			_, err := engine.IntegerStep(digitRange, choice.UintOf(12))
			assert.ErrorIs(t, err, engine.ErrCannotInvert, "no choice of the bounds is the integer")
			assert.Equal(t, assert.ErrorAs[*fault.Error](t, err, "a fault").Reason, "12 is outside [0, 9]",
				"the integer and the bounds")
		})
	})

	t.Run("CollectionSteps", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a flag and the steps of each item, then a stop", func(t *testing.T) {
			t.Parallel()
			items := [][]engine.Step{{stepOf(7)}, {stepOf(3)}}
			steps, err := engine.CollectionSteps(sizes(t, 0, 3), items)
			assert.NoError(t, err, "two items fit")
			assert.True(t, sameChoices(choicesOf(steps), integers(1, 7, 1, 3, 0)), "the steps")
		})

		t.Run("returns an error for more items than the sizes admit", func(t *testing.T) {
			t.Parallel()
			_, err := engine.CollectionSteps(sizes(t, 0, 1), [][]engine.Step{{stepOf(7)}, {stepOf(3)}})
			assert.ErrorIs(t, err, engine.ErrCannotInvert, "no collection of the sizes has two items")
			assert.Equal(t, assert.ErrorAs[*fault.Error](t, err, "a fault").Reason,
				"2 elements are outside the sizes of the collection", "two items are too many")
		})
	})

	t.Run("CompareSteps", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			a, b []engine.Step
			want int
		}{
			{
				name: "returns -1 for fewer steps",
				a:    []engine.Step{stepOf(9)},
				b:    []engine.Step{stepOf(0), stepOf(0)},
				want: -1,
			},
			{
				name: "returns +1 for a larger first step",
				a:    []engine.Step{stepOf(4)},
				b:    []engine.Step{stepOf(3)},
				want: 1,
			},
			{name: "returns 0 for the same steps", a: []engine.Step{stepOf(4)}, b: []engine.Step{stepOf(4)}, want: 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, engine.CompareSteps(tt.a, tt.b), tt.want, "the shortlex order")
			})
		}
	})

	t.Run("Absent", func(t *testing.T) {
		t.Parallel()

		var noInt *int
		tests := []struct {
			name string
			give any
			want bool
		}{
			{name: "reports true for nil", give: nil, want: true},
			{name: "reports true for a nil pointer", give: noInt, want: true},
			{name: "reports true for a nil slice", give: []int(nil), want: true},
			{name: "reports true for a nil map", give: map[string]int(nil), want: true},
			{name: "reports false for an empty slice", give: []int{}, want: false},
			{name: "reports false for zero", give: 0, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, engine.Absent(tt.give), tt.want, "whether the value is absent")
			})
		}
	})

	t.Run("ListItems", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give any
			want []any
			ok   bool
		}{
			{name: "returns the elements of a slice", give: []int{1, 2}, want: []any{1, 2}, ok: true},
			{name: "returns the elements of an array", give: [2]string{"a", "b"}, want: []any{"a", "b"}, ok: true},
			{name: "reports false for a string", give: "ab"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := engine.ListItems(tt.give)
				assert.Equal(t, [2]any{got, ok}, [2]any{tt.want, tt.ok}, "the elements")
			})
		}
	})

	t.Run("IntegerValue", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give any
			want choice.Int
			ok   bool
		}{
			{name: "returns a signed integer", give: int8(-3), want: choice.IntOf(-3), ok: true},
			{
				name: "returns an unsigned integer",
				give: uint64(math.MaxUint64),
				want: choice.UintOf(math.MaxUint64),
				ok:   true,
			},
			{name: "reports false for a float", give: 1.0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := engine.IntegerValue(tt.give)
				assert.Equal(t, [2]any{got, ok}, [2]any{tt.want, tt.ok}, "the integer")
			})
		}
	})

	t.Run("SameValue", func(t *testing.T) {
		t.Parallel()

		selfList := []any{nil}
		selfList[0] = selfList
		otherSelfList := []any{nil}
		otherSelfList[0] = otherSelfList
		tests := []struct {
			name string
			give any
			want any
			same bool
		}{
			{name: "reports true for two equal integers", give: 7, want: 7, same: true},
			{name: "keeps -0 apart from +0", give: math.Copysign(0, -1), want: 0.0},
			{
				name: "reports two NaNs with other payloads as one value",
				give: math.NaN(),
				want: math.Float64frombits(0x7FF8000000000001),
				same: true,
			},
			{
				name: "compares a value of another type by its typed literal",
				give: int8(3),
				want: 3,
				same: true,
			},
			{name: "reports false for a value of another type without a typed literal", give: make(chan int), want: 1},
			{
				name: "reports true for two lists that contain themselves",
				give: selfList,
				want: otherSelfList,
				same: true,
			},
			{
				name: "keeps a list that contains itself apart from a list that contains an empty list",
				give: selfList,
				want: []any{[]any{}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, engine.SameValue(tt.give, tt.want), tt.same, "whether the values are one value")
			})
		}
	})
}

// TestInverseAllocs checks the allocation ceiling of Invert of a digit,
// and of the parts of an inverse.
func TestInverseAllocs(t *testing.T) {
	digit, upTo := engine.Integer(0, 9), sizes(t, 0, 3)
	items := [][]engine.Step{{stepOf(7)}}
	a, b := []engine.Step{stepOf(4)}, []engine.Step{stepOf(3)}
	pair := []int{1, 2}
	assert.MaxAllocs(t, func() { _, _ = engine.Invert(digit, 7) }, invertAllocs,
		"Invert allocates the steps and the replay that checks them")
	assert.MaxAllocs(t, func() { _, _, _ = digit.Inverse(7) }, inverseAllocs, "Inverse allocates its steps")
	seven := integers(7)
	decoded := func(c *engine.Case) { digit.Decode(c) }
	assert.MaxAllocs(t, func() { engine.Replay(decoded, seven, nil) }, decodeCaseAllocs,
		"a case that decodes a digit")
	assert.MaxAllocs(t, func() { _, _ = engine.IntegerStep(digitRange, choice.UintOf(3)) }, 0,
		"IntegerStep allocates nothing for an admitted integer")
	assert.MaxAllocs(t, func() { _, _ = engine.CollectionSteps(upTo, items) }, collectionStepsAllocs,
		"CollectionSteps allocates its steps")
	assert.MaxAllocs(t, func() { _ = engine.CompareSteps(a, b) }, 0, "CompareSteps allocates nothing")
	assert.MaxAllocs(t, func() { _ = engine.Absent(pair) }, 0, "Absent allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = engine.ListItems(pair) }, listItemsAllocs, "ListItems allocates its elements")
	assert.MaxAllocs(t, func() { _, _ = engine.IntegerValue(pair[0]) }, 0, "IntegerValue allocates nothing")
	assert.MaxAllocs(t, func() { _ = engine.SameValue(pair, pair) }, sameValueAllocs,
		"SameValue allocates the canonical key of each value")
}

// BenchmarkInverse measures Invert of a digit, and the parts of an inverse.
func BenchmarkInverse(b *testing.B) {
	digit := engine.Integer(0, 9)

	b.Run("Invert", func(b *testing.B) {
		var got []choice.Choice
		c := bench.Start(b).MaxAllocs(invertAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = engine.Invert(digit, 7)
		}
		assert.True(b, sameChoices(got, integers(7)), "the choice")
	})

	b.Run("Decode", func(b *testing.B) {
		var got int
		body := func(c *engine.Case) { got = digit.Decode(c) }
		seven := integers(7)
		c := bench.Start(b).MaxAllocs(decodeCaseAllocs)
		defer c.End()
		for c.Loop() {
			engine.Replay(body, seven, nil)
		}
		assert.Equal(b, got, 7, "the value")
	})

	b.Run("Inverse", func(b *testing.B) {
		var got []engine.Step
		c := bench.Start(b).MaxAllocs(inverseAllocs)
		defer c.End()
		for c.Loop() {
			got, _, _ = digit.Inverse(7)
		}
		assert.Length(b, got, 1, "one step")
	})

	b.Run("IntegerStep", func(b *testing.B) {
		var got engine.Step
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = engine.IntegerStep(digitRange, choice.UintOf(3))
		}
		assert.True(b, got.Value.Equal(integers(3)[0]), "the choice")
	})

	b.Run("CollectionSteps", func(b *testing.B) {
		var got []engine.Step
		upTo, items := sizes(b, 0, 3), [][]engine.Step{{stepOf(7)}}
		c := bench.Start(b).MaxAllocs(collectionStepsAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = engine.CollectionSteps(upTo, items)
		}
		assert.Length(b, got, 3, "a flag, the item and the stop")
	})

	b.Run("CompareSteps", func(b *testing.B) {
		var got int
		x, y := []engine.Step{stepOf(4)}, []engine.Step{stepOf(3)}
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.CompareSteps(x, y)
		}
		assert.Equal(b, got, 1, "the larger first step")
	})

	b.Run("Absent", func(b *testing.B) {
		var got bool
		pair := []int{1, 2}
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.Absent(pair)
		}
		assert.False(b, got, "a slice is present")
	})

	b.Run("ListItems", func(b *testing.B) {
		var got []any
		pair := []int{1, 2}
		c := bench.Start(b).MaxAllocs(listItemsAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = engine.ListItems(pair)
		}
		assert.Equal(b, got, []any{1, 2}, "the elements")
	})

	b.Run("IntegerValue", func(b *testing.B) {
		var got choice.Int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = engine.IntegerValue(7)
		}
		assert.Equal(b, got, choice.UintOf(7), "the integer")
	})

	b.Run("SameValue", func(b *testing.B) {
		var got bool
		pair := []int{1, 2}
		c := bench.Start(b).MaxAllocs(sameValueAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.SameValue(pair, pair)
		}
		assert.True(b, got, "a list is itself")
	})
}

// stepOf returns the step of v under the digits' bounds.
func stepOf(v uint64) engine.Step {
	return engine.Step{Bounds: choice.OfInteger(digitRange), Value: unsigned(v)}
}

// choicesOf returns the choice of each step, in order.
func choicesOf(steps []engine.Step) []choice.Choice {
	out := make([]choice.Choice, len(steps))
	for i, s := range steps {
		out[i] = s.Value
	}
	return out
}
