// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

import (
	"reflect"

	"go.dokimi.dev/assert/internal/equality"
)

// allocRuns is the number of calls whose allocations an allocation check
// averages.
const allocRuns = 100

// cached is a value with a cache of its own, which the transform byValue
// leaves out.
type cached struct {
	Value int
	size  int
}

// sealed keeps a cached value in an unexported field.
type sealed struct {
	c cached
}

// halved is a count that the transform halving halves.
type halved int

// transformOf returns the transform of the values of type T by f.
func transformOf[T, R any](f func(T) R) equality.Transform {
	return equality.Transform{
		In:    reflect.TypeFor[T](),
		Apply: func(v reflect.Value) reflect.Value { return reflect.ValueOf(f(v.Interface().(T))) },
	}
}

// byValue compares a cached value by its Value alone.
var byValue = equality.Rules{Transforms: []equality.Transform{transformOf(func(c cached) int { return c.Value })}}

// halving compares a halved count by its half, which is a halved count too.
var halving = equality.Rules{Transforms: []equality.Transform{transformOf(func(h halved) halved { return h / 2 })}}

// chained compares a cached value by its Value as a halved count, and a halved
// count by its quarter, so a second transform at one place would equate the
// counts 4 and 6.
var chained = equality.Rules{Transforms: []equality.Transform{
	transformOf(func(c cached) halved { return halved(c.Value) }),
	transformOf(func(h halved) halved { return h / 4 }),
}}

// link is a node of a list, which can contain itself.
type link struct {
	ID   int
	Next *link
}

// hidden is a struct with an unexported field.
type hidden struct {
	Shown int
	kept  int
}

// holder is a struct with a field of an interface type.
type holder struct {
	V any
}

// spot is a map key of two ints, which == compares as equal does.
type spot struct {
	X, Y int
}

// list returns a list of n links with the IDs 0 to n - 1, and the last link
// with the ID last.
func list(n, last int) *link {
	head := &link{ID: last}
	for i := n - 2; i >= 0; i-- {
		head = &link{ID: i, Next: head}
	}
	return head
}
