// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"reflect"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// The ids of the generators that select among stated values or
// generators.
const (
	// sampledFromID is the id of a sampled-from.
	sampledFromID = "sampled-from"
	// oneOfID is the id of a one-of.
	oneOfID = "one-of"
	// optionalID is the id of an optional.
	optionalID = "optional"
	// permutationID is the id of a permutation.
	permutationID = "permutation"
)

// SampledFrom returns a generator of one of values: an integer index that
// decides structure. Its simplest value is the first. It runs backwards
// through the index of the first value equal to the given one. It panics
// when values is empty.
func SampledFrom[T any](values ...T) Generator[T] {
	if len(values) == 0 {
		panic("prop: " + sampledFromID + " of no value")
	}
	stated := slices.Clone(values)
	bounds := indices(len(stated))
	decode := func(c *Case) T {
		span := c.openSpan(sampledFromID)
		defer c.closeSpan(span)
		return stated[c.Structure(bounds, 0).Magnitude()]
	}
	return NewInvertible(sampledFromID, decode, func(v any) ([]Step, T, error) {
		for i, s := range stated {
			if SameValue(v, s) {
				return []Step{indexStep(bounds, i)}, s, nil
			}
		}
		var zero T
		return nil, zero, uninvertible("%v is none of the %d values", v, len(stated))
	})
}

// OneOf returns a generator of a value of one of gens: an integer index
// that decides structure, then that generator's choices. Its simplest
// value is the first generator's simplest. It runs backwards through the
// first generator whose inverse produces the value, and has no inverse for
// a value that none produces while one of gens has none. It panics when
// gens is empty.
func OneOf[T any](gens ...Generator[T]) Generator[T] {
	if len(gens) == 0 {
		panic("prop: " + oneOfID + " of no generator")
	}
	stated := slices.Clone(gens)
	bounds := indices(len(stated))
	decode := func(c *Case) T {
		span := c.openSpan(oneOfID)
		defer c.closeSpan(span)
		return stated[c.Structure(bounds, 0).Magnitude()].decode(c)
	}
	return NewInvertible(oneOfID, decode, func(v any) ([]Step, T, error) {
		steps, t, produced, unknown := firstBranch(stated, bounds, v)
		if !produced {
			return nil, t, noBranch(unknown, "none of the %d alternatives produces %v", len(stated), v)
		}
		return steps, t, nil
	})
}

// Optional returns a generator of a pointer to a value of of, or nil: a
// presence choice in [0, 1] that decides structure, then the value's
// choices when present. Its simplest value is nil, and the edge phase
// makes the value present.
//
// It runs backwards from nil, a nil pointer, a nil slice or a nil map as
// absent, from a *T through the value it points to, and from any other
// value, such as one that a typed literal decodes to, as the present value
// itself.
func Optional[T any](of Generator[T]) Generator[*T] {
	decode := func(c *Case) *T {
		span := c.openSpan(optionalID)
		defer c.closeSpan(span)
		if c.Structure(bitBounds, 1).Magnitude() == 0 {
			return nil
		}
		v := of.decode(c)
		return &v
	}
	return NewInvertible(optionalID, decode, func(v any) ([]Step, *T, error) {
		if Absent(v) {
			return []Step{bitStep(false)}, nil, nil
		}
		if p, ok := v.(*T); ok {
			v = *p
		}
		steps, present, err := of.inverse(v)
		if err != nil {
			return nil, nil, err
		}
		return append([]Step{bitStep(true)}, steps...), &present, nil
	})
}

// Absent reports whether v is the value of an absent optional: nil, or a
// nil pointer, slice, map or interface. A typed literal of null decodes to
// nil, and a list or a map literal of null to a nil slice or map.
func Absent(v any) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

// Permutation returns a generator of the orderings of values: one swap
// choice per position. Position i, from 0 to len(values) - 2, swaps with
// the index the case chooses in [i, len(values) - 1]. The target of that
// choice is i, so the simplest value is values in their stated order. It
// runs backwards through the smallest index at each swap that orders the
// values as the given list does, and the fault of an element that no swap
// puts in its place is at the element's index.
func Permutation[T any](values ...T) Generator[[]T] {
	stated := slices.Clone(values)
	decode := func(c *Case) []T {
		ordered := slices.Clone(stated)
		last := len(ordered) - 1
		span := c.openSpan(permutationID)
		defer c.closeSpan(span)
		for i := range last {
			swap := choice.MustIntegerBounds(choice.UintOf(uint64(i)), choice.UintOf(uint64(last)))
			j := c.Integer(swap).Magnitude()
			ordered[i], ordered[j] = ordered[j], ordered[i]
		}
		return ordered
	}
	return NewInvertible(permutationID, decode, func(v any) ([]Step, []T, error) {
		items, ok := ListItems(v)
		if !ok || len(items) != len(stated) {
			return nil, nil, uninvertible("%v is no list of %d values", v, len(stated))
		}
		ordered := slices.Clone(stated)
		last := len(ordered) - 1
		var steps []Step
		for i := range last {
			j := slices.IndexFunc(ordered[i:], func(s T) bool { return SameValue(items[i], s) })
			if j < 0 {
				return nil, nil, fault.At(uninvertible("the element is none of the values left to order"),
					fault.Index(i))
			}
			swap := choice.MustIntegerBounds(choice.UintOf(uint64(i)), choice.UintOf(uint64(last)))
			steps = append(steps, stepOf(swap, choice.UintOf(uint64(i+j))))
			ordered[i], ordered[i+j] = ordered[i+j], ordered[i]
		}
		if last >= 0 && !SameValue(items[last], ordered[last]) {
			return nil, nil, fault.At(uninvertible("the element is not the value left to order"), fault.Index(last))
		}
		return steps, ordered, nil
	})
}

// indices returns the bounds [0, n - 1] of an index into n values, for n
// of 1 or more.
func indices(n int) choice.IntegerBounds {
	return choice.MustIntegerBounds(choice.Int{}, choice.UintOf(uint64(n-1)))
}
