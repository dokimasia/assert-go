// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"slices"

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
// decides structure. Its simplest value is the first. It panics when
// values is empty.
func SampledFrom[T any](values ...T) Generator[T] {
	if len(values) == 0 {
		panic("prop: " + sampledFromID + " of no value")
	}
	stated := slices.Clone(values)
	bounds := indices(len(stated))
	return NewGenerator(sampledFromID, func(c *Case) T {
		span := c.openSpan(sampledFromID)
		defer c.closeSpan(span)
		return stated[c.Structure(bounds, 0).Magnitude()]
	})
}

// OneOf returns a generator of a value of one of gens: an integer index
// that decides structure, then that generator's choices. Its simplest
// value is the first generator's simplest. It panics when gens is empty.
func OneOf[T any](gens ...Generator[T]) Generator[T] {
	if len(gens) == 0 {
		panic("prop: " + oneOfID + " of no generator")
	}
	stated := slices.Clone(gens)
	bounds := indices(len(stated))
	return NewGenerator(oneOfID, func(c *Case) T {
		span := c.openSpan(oneOfID)
		defer c.closeSpan(span)
		return stated[c.Structure(bounds, 0).Magnitude()].decode(c)
	})
}

// Optional returns a generator of a pointer to a value of of, or nil: a
// presence choice in [0, 1] that decides structure, then the value's
// choices when present. Its simplest value is nil, and the edge phase
// makes the value present.
func Optional[T any](of Generator[T]) Generator[*T] {
	return NewGenerator(optionalID, func(c *Case) *T {
		span := c.openSpan(optionalID)
		defer c.closeSpan(span)
		if c.Structure(bitBounds, 1).Magnitude() == 0 {
			return nil
		}
		v := of.decode(c)
		return &v
	})
}

// Permutation returns a generator of the orderings of values: one swap
// choice per position. Position i, from 0 to len(values) - 2, swaps with
// the index the case chooses in [i, len(values) - 1]. The target of that
// choice is i, so the simplest value is values in their stated order.
func Permutation[T any](values ...T) Generator[[]T] {
	stated := slices.Clone(values)
	return NewGenerator(permutationID, func(c *Case) []T {
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
	})
}

// indices returns the bounds [0, n - 1] of an index into n values, for n
// of 1 or more.
func indices(n int) choice.IntegerBounds {
	return choice.MustIntegerBounds(choice.Int{}, choice.UintOf(uint64(n-1)))
}
