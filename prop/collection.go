// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import "go.dokimi.dev/assert/internal/prop/engine"

// listing is what the options of one [List] state.
type listing struct {
	// lengths are the bounds on the list's length.
	lengths lengths
	// unique reports whether the list discards an element equal to an
	// earlier one.
	unique bool
}

// ListOption configures [List]: a [SizeOption], or [Unique]. A later option
// overrides an earlier one of the same bound.
type ListOption interface {
	// applyList applies the option to the options of a list.
	applyList(l *listing)
}

// Unique makes [List] discard an element equal to an earlier one. Two
// values are equal when they have the same type and the same value, with
// floats compared by their bits: -0 differs from +0, and every NaN is one
// value. After a discard the list decides again whether to continue, and
// after ten discards in a row it stops. A list still below its shortest
// length then rejects the case.
func Unique() ListOption {
	return unique{}
}

// unique is the option that [Unique] returns.
type unique struct{}

// applyList makes the list discard repeated elements.
func (unique) applyList(l *listing) {
	l.unique = true
}

// List returns a generator of lists of the values of of, with lengths that
// the options admit: per element a choice to continue that decides
// structure, then the element in a span labelled element, and a final
// choice to stop. Its simplest value is the shortest list of simplest
// elements. It panics when the options state no length.
func List[T any](of Generator[T], opts ...ListOption) Generator[[]T] {
	var l listing
	for _, o := range opts {
		o.applyList(&l)
	}
	sizes := l.lengths.sizes()
	if l.unique {
		return Generator[[]T](engine.UniqueList(engine.Generator[T](of), sizes))
	}
	return Generator[[]T](engine.List(engine.Generator[T](of), sizes))
}

// Dict returns a generator of maps from values of keys to values of
// values, with lengths that the options admit, decoded as a unique list of
// entries compared by key: per entry a choice to continue, then the key and
// the value in a span labelled entry. Its simplest value is the map of the
// fewest entries.
//
// A map with float keys stores what Go's equality allows: one entry for the
// two zeros, and one for each NaN. The draw keeps the two zeros apart and
// reads every NaN as one key, as the definition does, so such a map can
// contain fewer entries than the draw decoded. It panics when the options
// state no length.
func Dict[K comparable, V any](keys Generator[K], values Generator[V], opts ...SizeOption) Generator[map[K]V] {
	return Generator[map[K]V](engine.Dict(engine.Generator[K](keys), engine.Generator[V](values), sizesOf(opts)))
}
