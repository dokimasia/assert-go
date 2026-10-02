// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import "go.dokimi.dev/assert/internal/prop/engine"

// Generator is a domain of values of T, and how a case decodes one of them
// from its choices. A body draws a value with [Case.Draw].
//
// A generator never reads a random source. It asks the case for choices
// with bounds, and the case supplies each value: drawn from the run's
// random source, read back from a stored case or a replay token, given by
// an edge case, or decoded from a fuzzer's bytes. Each generator opens a
// span labelled with its id around its choices, and the shrinker edits
// whole spans.
//
// # Concurrency
//
// A Generator is a value whose methods return a new generator and change
// none, so one generator serves any number of cases at once, provided the
// functions it was built from are safe to call concurrently.
//
// # Allocation contract
//
// A method allocates the closure of the generator it returns. A decode
// allocates what its value needs, and the case records its choices and
// spans.
type Generator[T any] engine.Generator[T]

// Map returns a generator of f applied to each value of g. It makes g's
// choices and adds no span of its own, so a value of the result shrinks as
// the value of g it was mapped from. A mapped integer or duration states
// its nearest passing value as f of the value one step towards the target.
func (g Generator[T]) Map[U any](f func(T) U) Generator[U] {
	return Generator[U](engine.Generator[T](g).Map(f))
}

// Filter returns a generator of the values of g that keep reports true
// for. Each attempt decodes in a span labelled filter. A rejected attempt is
// removed from the case's record, so a replay decodes the kept value at its
// first attempt. After three attempts that keep rejects, the filter rejects
// the case.
func (g Generator[T]) Filter(keep func(T) bool) Generator[T] {
	return Generator[T](engine.Generator[T](g).Filter(keep))
}

// Bind returns a generator that decodes a value of g, then a value of the
// generator that f returns for it, in one span labelled bind around both.
func (g Generator[T]) Bind[U any](f func(T) Generator[U]) Generator[U] {
	return Generator[U](engine.Generator[T](g).Bind(func(v T) engine.Generator[U] {
		return engine.Generator[U](f(v))
	}))
}

// Composite returns a generator whose value f returns from the case it
// receives. f draws from other generators with [Case.Draw], in one span
// labelled composite around its draws. Its draws record their values as a
// body's draws do.
func Composite[T any](f func(*Case) T) Generator[T] {
	return Generator[T](engine.Composite(func(c *engine.Case) T { return f((*Case)(c)) }))
}

// Just returns a generator of value alone. It makes no choice, and opens an
// empty span labelled just.
func Just[T any](value T) Generator[T] {
	return Generator[T](engine.Just(value))
}
