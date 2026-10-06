// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import "fmt"

// The id and the default of a recursive generator.
const (
	// recursiveID is the id of a recursive generator.
	recursiveID = "recursive"
	// DefaultMaxLeaves is the number of base values one recursive value
	// may draw when the caller states no number.
	DefaultMaxLeaves = 100
)

// recursion is a recursive generator: the base, the extension built over
// a position of the value being decoded, and the bound on its leaves.
type recursion[T any] struct {
	// base decodes a leaf.
	base Generator[T]
	// extend decodes an extension, whose positions are recursive values.
	extend Generator[T]
	// maxLeaves is the most base values one recursive value draws.
	maxLeaves int
}

// Recursive returns a generator of a base value, or an extension whose
// positions are recursive values. extend receives the generator of one
// position and returns the extension built over it.
//
// Each position decides between the base, 0, and the extension, 1, with
// an integer choice that decides structure. Once one value has drawn
// maxLeaves values from the base, every further position takes the base,
// with bounds [0, 0]. Its simplest value is the base's simplest. Each
// position runs backwards through the base, and through the extension when
// the base does not produce the value. It panics when maxLeaves is below 1.
func Recursive[T any](base Generator[T], extend func(self Generator[T]) Generator[T], maxLeaves int) Generator[T] {
	if maxLeaves < 1 {
		panic(fmt.Sprintf("prop: %s with %d leaves draws no base value", recursiveID, maxLeaves))
	}
	r := &recursion[T]{base: base, maxLeaves: maxLeaves}
	r.extend = extend(NewInvertible(recursiveID, r.position, r.invert))
	decode := func(c *Case) T {
		defer c.Enter(r)()
		return r.position(c)
	}
	return NewInvertible(recursiveID, decode, r.invert)
}

// position decodes one position of the value being decoded in c.
func (r *recursion[T]) position(c *Case) T {
	choices := bitBounds
	if c.Count(r) >= r.maxLeaves {
		choices = indices(1)
	}
	span := c.openSpan(recursiveID)
	defer c.closeSpan(span)
	if c.Structure(choices, 0).Magnitude() == 1 {
		return r.extend.decode(c)
	}
	c.Add(r)
	return r.base.decode(c)
}

// invert returns the steps of one position that decode to v: the base's
// choice and steps, or, when the base does not produce v, the extension's.
// It has no inverse for a value that neither produces while one of them has
// none. A value past the bound on its leaves fails the replay that [Invert]
// checks the steps with.
func (r *recursion[T]) invert(v any) ([]Step, T, error) {
	branches := [...]Generator[T]{r.base, r.extend}
	steps, t, produced, unknown := firstBranch(branches[:], bitBounds, v)
	if !produced {
		return nil, t, noBranch(unknown, "neither the base nor the extension produces %v", v)
	}
	return steps, t, nil
}
