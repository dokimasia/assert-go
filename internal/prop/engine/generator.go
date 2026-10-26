// Copyright Dokimasia B.V. 2026
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
	// mapName names a mapped generator in the error of its missing inverse.
	// A mapped generator keeps the id of the generator it maps.
	mapName = "map"
	// filterAttempts is the most values a filter decodes before it
	// rejects the case.
	filterAttempts = 3
)

// Generator is a domain of T, and how a case decodes one of its values.
//
// Every method of a Generator returns a new generator and changes none, so
// one generator serves any number of cases at once when its decode is safe
// for concurrent use.
type Generator[T any] struct {
	// id is the generator's id in the definition.
	id string
	// decode asks a case for choices, inside the generator's spans, and
	// returns the value they decode to.
	decode func(*Case) T
	// invert returns the steps that decode to v in the inversion in and the
	// value they decode to, for v a value of T or the value that the typed
	// literal of one decodes to. It returns an error for a v that the
	// generator does not produce, which inverse makes a fault of the kind
	// ErrCannotInvert. It is nil for a generator without an inverse.
	invert func(in Inversion, v any) ([]Step, T, error)
	// erased is the generator with its type erased.
	erased erased
	// pick returns the generator that a run states in place of this one,
	// from what the run states for the generators of types, for a generator
	// of [Picking]. It is nil for any other generator.
	pick func(using any) (Generator[T], bool)
}

// erased is a generator with its type erased, as a draw records it for
// the explain phase.
type erased struct {
	// name names the generator in the error of its missing inverse.
	name string
	// decode decodes a value as the typed generator does.
	decode func(*Case) any
	// integer reports whether the generator is an integer or a duration,
	// whose draw the explain phase steps towards its target.
	integer bool
	// bounds are the bounds of an integer or a duration.
	bounds choice.IntegerBounds
	// value returns an integer or a duration's value for a choice value.
	value func(choice.Int) any
	// neutral returns the value of the generator that a value maps from, as
	// that generator states it, for a generator that MapBack built. It is
	// nil for a generator whose values state themselves.
	neutral func(v any) any
}

// neutralOf returns v as the generator states it: the value that v maps
// from for a generator that MapBack built, and v itself for any other.
func (e erased) neutralOf(v any) any {
	if e.neutral == nil {
		return v
	}
	return e.neutral(v)
}

// NewGenerator returns a generator with id that decodes with decode, and
// has no inverse. decode opens the generator's spans itself, as every
// generator of the definition opens a span labelled with its id around its
// choices. A generator built outside this package, such as string-matching,
// makes its choices with [Case.Integer], [Case.Structure], [Case.Span] and
// [Collect].
func NewGenerator[T any](id string, decode func(*Case) T) Generator[T] {
	return Generator[T]{
		id:     id,
		decode: decode,
		erased: erased{name: id, decode: func(c *Case) any { return decode(c) }},
	}
}

// NewInvertible returns a generator with id that decodes with decode, as
// [NewGenerator] does, and runs backwards with invert. invert returns the
// steps whose choices decode to v and the value that they decode to. v is a
// value of T, or the value that the typed literal of one decodes to. invert
// passes in to the inverse of each generator inside the generator, through
// [Generator.Inverse]. [Invert] checks the steps against a replay of decode.
//
// invert returns an error for a v that decode never returns: a fault of the
// kind [ErrCannotInvert] at the part of v that no choice produces. The
// generator's inverse makes any other error the cause of such a fault.
func NewInvertible[T any](id string, decode func(*Case) T,
	invert func(in Inversion, v any) ([]Step, T, error),
) Generator[T] {
	g := NewGenerator(id, decode)
	g.invert = invert
	return g
}

// Erase returns g as a generator of any: the same id, choices, spans,
// explain step and inverse, with each value of g as an any.
func Erase[T any](g Generator[T]) Generator[any] {
	e := Generator[any]{id: g.id, decode: g.erased.decode, erased: g.erased}
	if g.invert != nil {
		e.invert = func(in Inversion, v any) ([]Step, any, error) {
			steps, t, err := g.invert(in, v)
			return steps, t, err
		}
	}
	return e
}

// Picking returns g with a decode and an inverse that use the generator
// that pick returns, where pick returns one, and g where it does not. pick
// receives what the run states for the generators of types, as
// [Settings.Using] states it, and nil for a run that states none. So one
// generator of a type serves every run, each with the generators that it
// states. A draw of the result records the generator that pick returns. The
// id of the result is g's, and so are its explain step and the value that
// [Drawn.Neutral] returns for a draw that pick leaves to g.
func Picking[T any](g Generator[T], pick func(using any) (Generator[T], bool)) Generator[T] {
	picking := g
	picking.pick = pick
	picking.decode = func(c *Case) T {
		if picked, ok := pick(c.currentUsing()); ok {
			return picked.decode(c)
		}
		return g.decode(c)
	}
	picking.erased.decode = func(c *Case) any { return picking.decode(c) }
	picking.invert = func(in Inversion, v any) ([]Step, T, error) {
		if picked, ok := pick(in.using); ok {
			return picked.inverse(in, v)
		}
		return g.inverse(in, v)
	}
	return picking
}

// Within returns g with a decode and an inverse that run in place of what
// the run states for the generators of types what using returns, so a
// generator of [Picking] inside g picks from that. using is called each time
// the result decodes or runs backwards. The id, the explain step, the
// neutral values and the inverse of the result are g's, and so is the
// absence of an inverse.
func Within[T any](g Generator[T], using func() any) Generator[T] {
	return Rebind(g, func(any) any { return using() })
}

// Rebind returns g with a decode and an inverse that run under the
// generators of types that using returns. using receives what the run states
// for the generators of types where the result decodes or runs backwards,
// and Rebind calls it each time. A generator of [Picking] inside g picks from
// what using returns. The id, the explain step, the neutral values and the
// inverse of the result are g's, and so is the absence of an inverse.
func Rebind[T any](g Generator[T], using func(outer any) any) Generator[T] {
	rebound := g
	rebound.pick = nil
	rebound.decode = func(c *Case) T {
		outer := c.swapUsing(using(c.currentUsing()))
		defer c.swapUsing(outer)
		return g.decode(c)
	}
	rebound.erased.decode = func(c *Case) any { return rebound.decode(c) }
	if g.invert != nil {
		rebound.invert = func(in Inversion, v any) ([]Step, T, error) {
			return g.invert(Inversion{using: using(in.using)}, v)
		}
	}
	return rebound
}

// ID returns the generator's id in the definition.
func (g Generator[T]) ID() string {
	return g.id
}

// Decode asks c for g's choices, inside g's spans, and returns the value
// they decode to. It records no draw: a generator that decodes another one
// inside its own value calls it, and a body draws with [Draw].
func (g Generator[T]) Decode(c *Case) T {
	return g.decode(c)
}

// Inverse returns the steps whose choices decode to v under g in the
// inversion in, and the value they decode to, as a generator that runs
// another one backwards inside its own value needs them. It returns a fault
// of the kind [ErrCannotInvert] at the part of v that no choice produces,
// and one of the kind [ErrNoInverse] for a g without an inverse. [Invert]
// checks the steps against a replay, and Inverse does not.
func (g Generator[T]) Inverse(in Inversion, v any) ([]Step, T, error) {
	return g.inverse(in, v)
}

// inverse returns the steps that decode to v in the inversion in and the
// value they decode to, a fault of the kind ErrCannotInvert for a v that g
// does not produce, and one of the kind ErrNoInverse for a g without an
// inverse.
func (g Generator[T]) inverse(in Inversion, v any) ([]Step, T, error) {
	if g.invert == nil {
		var zero T
		return nil, zero, noInverse("%s has no inverse", g.erased.name)
	}
	steps, t, err := g.invert(in, v)
	if err != nil {
		return steps, t, cannotInvert(g.erased.name, err)
	}
	return steps, t, nil
}

// Map returns a generator of f applied to each value of g. It makes g's
// choices and adds no span of its own, so a mapped integer or duration is
// still one for the explain phase: its nearest passing value is f of the
// value one step towards the target. It has no inverse, because f has
// none. [Generator.MapBack] states one.
func (g Generator[T]) Map[U any](f func(T) U) Generator[U] {
	mapped := NewGenerator(g.id, func(c *Case) U { return f(g.decode(c)) })
	mapped.erased.name = mapName
	if g.erased.integer {
		mapped.erased.integer, mapped.erased.bounds = true, g.erased.bounds
		mapped.erased.value = func(i choice.Int) any { return f(g.erased.value(i).(T)) }
	}
	return mapped
}

// MapBack returns a generator of f applied to each value of g, as
// [Generator.Map] does, which runs backwards through back. back returns the
// value of g that f maps to u, or an error for a u that f never returns.
//
// The inverse takes a U through back and then g's inverse. It takes any
// other value, such as the value that a typed literal decodes to, through
// g's inverse alone, and maps the value that g's choices decode to with f.
// An error of back is the cause of the inverse's fault, unless it is a
// fault of the kind [ErrCannotInvert] itself. [Drawn.Neutral] of a draw from
// the result is the value of g that back returns, as g states it.
func (g Generator[T]) MapBack[U any](f func(T) U, back func(U) (T, error)) Generator[U] {
	mapped := g.Map(f)
	mapped.erased.neutral = func(v any) any {
		u, _ := v.(U)
		t, err := back(u)
		if err != nil {
			return v
		}
		return g.erased.neutralOf(t)
	}
	mapped.invert = func(in Inversion, v any) ([]Step, U, error) {
		var zero U
		if u, ok := v.(U); ok {
			t, err := back(u)
			if err != nil {
				return nil, zero, err
			}
			v = t
		}
		steps, t, err := g.inverse(in, v)
		if err != nil {
			return nil, zero, err
		}
		return steps, f(t), nil
	}
	return mapped
}

// Filter returns a generator of g's values that keep reports true for.
// Each attempt decodes in a span of its own. A rejected attempt is removed
// from the case's record, so a replay of the record decodes the kept
// value at its first attempt. After three rejected attempts the filter
// rejects the case. It runs backwards through g's inverse, for a value
// that keep reports true for, and states each value as g does.
func (g Generator[T]) Filter(keep func(T) bool) Generator[T] {
	attempt := func(c *Case) T {
		span := c.openSpan(filterID)
		defer c.closeSpan(span)
		return g.decode(c)
	}
	decode := func(c *Case) T {
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
	}
	filtered := NewInvertible(filterID, decode, func(in Inversion, v any) ([]Step, T, error) {
		steps, t, err := g.inverse(in, v)
		if err != nil {
			return nil, t, err
		}
		if !keep(t) {
			return nil, t, uninvertible("the filter rejects %v", t)
		}
		return steps, t, nil
	})
	filtered.erased.neutral = g.erased.neutral
	return filtered
}

// Bind returns a generator that decodes a value of g, then a value of the
// generator that f returns for it, in one span around both. It has no
// inverse.
func (g Generator[T]) Bind[U any](f func(T) Generator[U]) Generator[U] {
	return NewGenerator(bindID, func(c *Case) U {
		span := c.openSpan(bindID)
		defer c.closeSpan(span)
		return f(g.decode(c)).decode(c)
	})
}

// Composite returns a generator whose value f returns from the case it
// receives. f draws from other generators, in one span around them all. It
// has no inverse.
func Composite[T any](f func(*Case) T) Generator[T] {
	return NewGenerator(compositeID, func(c *Case) T {
		span := c.openSpan(compositeID)
		defer c.closeSpan(span)
		return f(c)
	})
}

// Just returns a generator of value alone. It makes no choice, and runs
// backwards from a value equal to value.
func Just[T any](value T) Generator[T] {
	decode := func(c *Case) T {
		c.closeSpan(c.openSpan(justID))
		return value
	}
	return NewInvertible(justID, decode, func(_ Inversion, v any) ([]Step, T, error) {
		if !SameValue(v, value) {
			return nil, value, uninvertible("%v is not %v", v, value)
		}
		return nil, value, nil
	})
}

// Draw returns a value of g and records it under label. The draw's span is
// the first span g opens. Two draws may share a label. Every request of the
// draw is made under its label. In the case of [Settings.Draws], the draw
// first takes the next entry, and its choices are the ones that decode to
// the entry's value. In an example of values, the draw takes the next value
// while one is left, and makes no choice and opens no span. A draw of a
// generator of [Picking] draws from the generator that its pick returns,
// where it returns one. The draw keeps what the run states for the
// generators of types where it runs, so the explain phase decodes its
// generator again in the same way.
func Draw[T any](c *Case, g Generator[T], label string) T {
	if g.pick != nil {
		if picked, ok := g.pick(c.currentUsing()); ok {
			g = picked
		}
	}
	if c.valuing != nil {
		if v, ok := c.stated(); ok {
			// A form states the values of an example as values of T, so the
			// assertion fails only for a nil interface value, whose T is the
			// zero value.
			t, _ := v.(T)
			c.draw(label, t, c.nextSpan(), g.erased)
			return t
		}
	}
	if c.inverting != nil {
		in := Inversion{using: c.currentUsing()}
		c.enterDraw(label, func(v any) ([]choice.Choice, error) { return invert(in, g, v) })
	}
	span := c.nextSpan()
	defer c.closeDraw(c.openDraw(label))
	v := g.decode(c)
	c.draw(label, v, span, g.erased)
	return v
}
