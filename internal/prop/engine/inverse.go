// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"cmp"
	"errors"
	"reflect"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// ErrCannotInvert reports a value that no sequence of choices decodes to
// under a generator. Every value of a generator without an inverse is one.
var ErrCannotInvert = errors.New("engine: no choices decode to the value")

// Step is one choice of a generator that runs backwards: the bounds of the
// request that the generator's decode makes, and the choice that the
// request takes.
type Step struct {
	// Bounds are the bounds of the request.
	Bounds choice.Bounds
	// Value is the choice.
	Value choice.Choice
}

// Invert returns the choices that decode to value under g. value is a value
// of T, or the value that the typed literal of one decodes to, as
// [literal.Decode] returns it.
//
// Where more than one sequence of choices decodes to a value, the inverse
// takes the first in a fixed order:
//
//   - A one-of takes its first alternative that produces the value, and a
//     recursive value its base before its extension.
//   - A sampled-from takes the first equal value, and a permutation the
//     smallest index at each swap.
//   - An optional is absent before present, so an absent optional of an
//     optional is absent at the outer one.
//   - A dict's entries take the shortlex order of their own choices.
//
// A filter runs backwards through its source, and [Generator.MapBack]
// through the inverse that it states. Invert replays the choices it
// computes, and returns them only when the replay records them unchanged
// and decodes value.
//
// # Errors
//
// Invert returns a fault of the kind [ErrCannotInvert] for a value that g
// does not produce. Its path leads to the part of the value that no choice
// produces, as [2] does to the third element of a list. A generator built
// with [Generator.Map], [Generator.Bind] or [Composite] has no inverse, and
// Invert returns the fault for each of its values.
func Invert[T any](g Generator[T], value T) ([]choice.Choice, error) {
	return invert(g, value)
}

// invert returns the choices that decode to v under g, as [Invert] does,
// for v a value of T or the value that the typed literal of one decodes
// to. The replay is capped at the cost of the choices, so a decode that
// requests more of them is past its cap.
func invert[T any](g Generator[T], v any) ([]choice.Choice, error) {
	steps, normal, err := g.inverse(v)
	if err != nil {
		return nil, err
	}
	choices := make([]choice.Choice, len(steps))
	cost := 0
	for i, s := range steps {
		choices[i] = s.Value
		cost += 1 + len(s.Value.Sequence)
	}
	var decoded T
	e := execute(func(c *Case) { decoded = g.decode(c) }, replaying{choices: choices}, Settings{MaxChoices: cost})
	if e.Status != CasePassed {
		return nil, uninvertible("the choices of %v decode to no value", v)
	}
	if !slices.EqualFunc(e.Case.record(), choices, choice.Choice.Equal) ||
		canonicalKey(decoded) != canonicalKey(normal) {
		return nil, uninvertible("the choices of %v decode to %v", v, decoded)
	}
	return choices, nil
}

// uninvertible returns a fault of the kind ErrCannotInvert whose reason is
// format with args, as fmt.Sprintf formats them.
func uninvertible(format string, args ...any) *fault.Error {
	return fault.Of(ErrCannotInvert, format, args...)
}

// cannotInvert returns err, the error of the inverse of the generator name,
// as an error of the kind ErrCannotInvert. It returns err when err has the
// kind, and otherwise a fault of the kind whose cause is err, such as the
// error of the inverse that [Generator.MapBack] states.
func cannotInvert(name string, err error) error {
	if errors.Is(err, ErrCannotInvert) {
		return err
	}
	return uninvertible("the inverse of %s refuses the value", name).Because(err)
}

// stepOf returns the step of the integer i under b, which admits it.
func stepOf(b choice.IntegerBounds, i choice.Int) Step {
	return Step{Bounds: choice.OfInteger(b), Value: choice.Choice{Kind: choice.Integer, Integer: i}}
}

// indexStep returns the step of index i under b, which admits it.
func indexStep(b choice.IntegerBounds, i int) Step {
	return stepOf(b, choice.UintOf(uint64(i)))
}

// bitStep returns the step of a choice in [0, 1]: 1 for true, and 0 for
// false.
func bitStep(set bool) Step {
	return Step{Bounds: choice.OfInteger(bitBounds), Value: bit(set)}
}

// IntegerStep returns the step of the integer i under b, and a fault of the
// kind [ErrCannotInvert] that names i and b when b does not admit i.
func IntegerStep(b choice.IntegerBounds, i choice.Int) (Step, error) {
	if !b.Admits(i) {
		return Step{}, uninvertible("%v is outside [%v, %v]", i, b.Lo(), b.Hi())
	}
	return stepOf(b, i), nil
}

// IntegerValue returns v as a choice's integer, for a value of any integer
// type, and false for any other value.
func IntegerValue(v any) (choice.Int, bool) {
	rv := reflect.ValueOf(v)
	if rv.CanInt() {
		return choice.IntOf(rv.Int()), true
	}
	if rv.CanUint() {
		return choice.UintOf(rv.Uint()), true
	}
	return choice.Int{}, false
}

// floatValue returns v as a float64, for a value of either float type, and
// false for any other value.
func floatValue(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	if rv.CanFloat() {
		return rv.Float(), true
	}
	return 0, false
}

// ListItems returns the elements of v, a slice or an array, and false for
// any other value.
func ListItems(v any) ([]any, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	items := make([]any, rv.Len())
	for i := range items {
		items[i] = rv.Index(i).Interface()
	}
	return items, true
}

// SameValue reports whether v is want, as the definition compares generated
// values: two values of one type by their type and value, with floats by
// their bits, so -0 differs from +0 and every NaN is one value. A value of
// another type, such as one that a typed literal decodes to, compares by its
// typed literal, as the definition compares two literals. A value that
// contains itself compares in finite time.
func SameValue(v, want any) bool {
	if reflect.TypeOf(v) == reflect.TypeOf(want) {
		return canonicalKey(v) == canonicalKey(want)
	}
	x, ok := neutral(v)
	y, ok2 := neutral(want)
	return ok && ok2 && literal.Canonical(x) == literal.Canonical(y)
}

// neutral returns the value that the typed literal of v decodes to, and
// false for a v without one.
func neutral(v any) (any, bool) {
	raw, ok := literal.Encode(v)
	if !ok {
		return nil, false
	}
	decoded, err := literal.Decode(raw)
	return decoded, err == nil
}

// CollectionSteps returns the steps of a collection of items under sizes,
// as [Elements] decodes one: per item a continue flag of 1 and the item's
// steps, then a stop flag of 0. It returns a fault of the kind
// [ErrCannotInvert] for a number of items that sizes does not admit.
func CollectionSteps(sizes choice.Sizes, items [][]Step) ([]Step, error) {
	if !sizes.Admits(len(items)) {
		return nil, uninvertible("%d elements are outside the sizes of the collection", len(items))
	}
	var steps []Step
	for i, item := range items {
		steps = append(steps, stepOf(sizes.FlagBounds(i), choice.UintOf(1)))
		steps = append(steps, item...)
	}
	return append(steps, stepOf(sizes.FlagBounds(len(items)), choice.Int{})), nil
}

// CompareSteps compares two sequences of steps in the shortlex order of
// their keys, as the shrinker orders choice sequences: the shorter first,
// then by the key of the first step that differs. It returns -1, 0 or +1.
// The elements of a set and the entries of a map run backwards in this
// order, so a value has one sequence of choices whatever the order of its
// elements.
func CompareSteps(a, b []Step) int {
	return cmp.Or(cmp.Compare(len(a), len(b)), slices.CompareFunc(a, b, func(x, y Step) int {
		return x.Bounds.Key(x.Value).Compare(y.Bounds.Key(y.Value))
	}))
}
