// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// RecursiveOption configures [Recursive]. The zero RecursiveOption changes
// nothing.
type RecursiveOption struct {
	// maxLeaves is the most base values of one value, and 0 for the zero
	// option.
	maxLeaves int
}

// MaxLeaves bounds the base values that one value of [Recursive] draws, 100
// by default. It panics for n below 1.
func MaxLeaves(n int) RecursiveOption {
	if n < 1 {
		panic(fmt.Sprintf("prop: MaxLeaves(%d) is below 1", n))
	}
	return RecursiveOption{maxLeaves: n}
}

// Recursive returns a generator of a value of base, or of an extension
// whose positions are values of the generator itself. extend receives the
// generator of one position and returns the extension built over it.
//
// A choice that decides structure selects the base or the extension at
// each position, the base first. Once one value has drawn the most base
// values that [MaxLeaves] allows, every further position takes the base.
// Its simplest value is the base's simplest.
func Recursive[T any](
	base Generator[T],
	extend func(self Generator[T]) Generator[T],
	opts ...RecursiveOption,
) Generator[T] {
	maxLeaves := engine.DefaultMaxLeaves
	for _, o := range opts {
		if o.maxLeaves != 0 {
			maxLeaves = o.maxLeaves
		}
	}
	inner := func(self engine.Generator[T]) engine.Generator[T] {
		return engine.Generator[T](extend(Generator[T](self)))
	}
	return Generator[T](engine.Recursive(engine.Generator[T](base), inner, maxLeaves))
}
