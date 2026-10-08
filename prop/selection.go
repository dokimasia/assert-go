// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import "go.dokimi.dev/assert/internal/prop/engine"

// SampledFrom returns a generator of one of values: an index that decides
// structure. Its simplest value is the first. The generator keeps a copy of
// values. It panics when values is empty.
func SampledFrom[T any](values ...T) Generator[T] {
	return Generator[T](engine.SampledFrom(values...))
}

// OneOf returns a generator of a value of one of gens: an index that
// decides structure, then the choices of the generator it chooses. Its
// simplest value is the first generator's simplest. It panics when gens is
// empty.
func OneOf[T any](gens ...Generator[T]) Generator[T] {
	inner := make([]engine.Generator[T], len(gens))
	for i, g := range gens {
		inner[i] = engine.Generator[T](g)
	}
	return Generator[T](engine.OneOf(inner...))
}

// Optional returns a generator of a pointer to a value of of, or nil: a
// choice of presence that decides structure, then the value's choices when
// present. Its simplest value is nil, and an edge case makes the value
// present.
func Optional[T any](of Generator[T]) Generator[*T] {
	return Generator[*T](engine.Optional(engine.Generator[T](of)))
}

// Permutation returns a generator of the orderings of values: one choice
// per position i but the last, of the index in [i, len(values) - 1] that
// position i swaps with. Its simplest value is values in their stated
// order. The generator keeps a copy of values.
func Permutation[T any](values ...T) Generator[[]T] {
	return Generator[[]T](engine.Permutation(values...))
}
