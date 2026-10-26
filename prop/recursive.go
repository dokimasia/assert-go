// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"reflect"

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
//
// Inside the generator of T that a run states with [Using], a draw of [Of]
// in the extension takes each T from the generator of the position. That
// applies to a draw of T, to a draw of a type that contains T, and to the
// draws inside the stated generator of another type. The base takes T
// without the stated generator, as Using states.
//
//	nodes := prop.Recursive(prop.Just[Node](Leaf{}), func(prop.Generator[Node]) prop.Generator[Node] {
//		return prop.Composite(func(c *prop.Case) Node {
//			return Branch{Children: c.Draw(prop.Of[[]Node](), "children")}
//		})
//	})
//	prop.ForAll(t, "a tree has a depth", body, prop.Using(nodes))
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
		position := &using{typ: reflect.TypeFor[T](), erased: engine.Erase(self), typed: Generator[T](self)}
		return engine.Rebind(engine.Generator[T](extend(Generator[T](self))), position.extending)
	}
	return Generator[T](engine.Recursive(engine.Generator[T](base), inner, maxLeaves))
}
