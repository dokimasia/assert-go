// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import "go.dokimi.dev/assert/internal/prop/choice"

// The ids of the combinators, which label their spans.
const (
	// filterID labels each attempt of a filter.
	filterID = "filter"
	// bindID labels the span around both generators of a bind.
	bindID = "bind"
	// compositeID labels the span around everything a composite draws.
	compositeID = "composite"
	// justID labels the span of a just.
	justID = "just"
	// filterAttempts is the most values a filter decodes before it
	// rejects the case.
	filterAttempts = 3
)

// Generator is a domain of T, and how a case decodes one of its values.
//
// A Generator is a value. Every method returns a new one and changes none,
// so one generator serves any number of cases at once, unless its decode
// says otherwise.
type Generator[T any] struct {
	// id is the generator's id in the definition.
	id string
	// decode asks a case for choices, inside the generator's spans, and
	// returns the value they decode to.
	decode func(*Case) T
	// erased is the generator with its type erased.
	erased erased
}

// erased is a generator with its type erased, as a draw records it for
// the explain phase.
type erased struct {
	// decode decodes a value as the typed generator does.
	decode func(*Case) any
	// integer reports whether the generator is an integer or a duration,
	// whose draw the explain phase steps towards its target.
	integer bool
	// bounds are the bounds of an integer or a duration.
	bounds choice.IntegerBounds
	// value returns an integer or a duration's value for a choice value.
	value func(choice.Int) any
}

// NewGenerator returns a generator with id that decodes with decode.
// decode opens the generator's spans itself, as every generator of the
// definition opens a span labelled with its id around its choices. A
// generator built outside this package, such as string-matching, makes its
// choices with [Case.Integer], [Case.Structure], [Case.Span] and
// [Collect].
func NewGenerator[T any](id string, decode func(*Case) T) Generator[T] {
	return Generator[T]{id: id, decode: decode, erased: erased{decode: func(c *Case) any { return decode(c) }}}
}

// ID returns the generator's id in the definition.
func (g Generator[T]) ID() string {
	return g.id
}

// Map returns a generator of f applied to each value of g. It makes g's
// choices and adds no span of its own, so a mapped integer or duration is
// still one for the explain phase: its nearest passing value is f of the
// value one step towards the target.
func (g Generator[T]) Map[U any](f func(T) U) Generator[U] {
	mapped := NewGenerator(g.id, func(c *Case) U { return f(g.decode(c)) })
	if g.erased.integer {
		mapped.erased.integer, mapped.erased.bounds = true, g.erased.bounds
		mapped.erased.value = func(i choice.Int) any { return f(g.erased.value(i).(T)) }
	}
	return mapped
}

// Filter returns a generator of g's values that keep reports true for.
// Each attempt decodes in a span of its own. A rejected attempt is removed
// from the case's record, so a replay of the record decodes the kept
// value at its first attempt. After three rejected attempts the filter
// rejects the case.
func (g Generator[T]) Filter(keep func(T) bool) Generator[T] {
	attempt := func(c *Case) T {
		span := c.openSpan(filterID)
		defer c.closeSpan(span)
		return g.decode(c)
	}
	return NewGenerator(filterID, func(c *Case) T {
		at := c.mark()
		v := attempt(c)
		for range filterAttempts - 1 {
			if keep(v) {
				return v
			}
			c.rewind(at)
			v = attempt(c)
		}
		c.Assume(keep(v))
		return v
	})
}

// Bind returns a generator that decodes a value of g, then a value of the
// generator that f returns for it, in one span around both.
func (g Generator[T]) Bind[U any](f func(T) Generator[U]) Generator[U] {
	return NewGenerator(bindID, func(c *Case) U {
		span := c.openSpan(bindID)
		defer c.closeSpan(span)
		return f(g.decode(c)).decode(c)
	})
}

// Composite returns a generator whose value f returns from the case it
// receives. f draws from other generators, in one span around all of
// them.
func Composite[T any](f func(*Case) T) Generator[T] {
	return NewGenerator(compositeID, func(c *Case) T {
		span := c.openSpan(compositeID)
		defer c.closeSpan(span)
		return f(c)
	})
}

// Just returns a generator of value alone. It makes no choice.
func Just[T any](value T) Generator[T] {
	return NewGenerator(justID, func(c *Case) T {
		c.closeSpan(c.openSpan(justID))
		return value
	})
}

// Draw returns a value of g and records it under label. The draw's span is
// the first span g opens. Two draws may share a label.
func Draw[T any](c *Case, g Generator[T], label string) T {
	span := c.nextSpan()
	v := g.decode(c)
	c.draw(label, v, span, g.erased)
	return v
}
