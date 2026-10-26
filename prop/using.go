// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sync"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// using is the generator of one type that [Using] states.
type using struct {
	// typ is the type.
	typ reflect.Type
	// erased is the generator with its type erased.
	erased engine.Generator[any]
	// typed is the Generator of the type.
	typed any
}

// Using makes g the generator of T in the run that it is passed to: of
// each draw of [Of] of T, of each T in a value that a draw of Of decodes,
// and of the input of a property form that is a T or contains one. It takes
// precedence over the generator that [Register] states and over T's shape.
// A second Using of one type replaces the first. A draw of any other
// generator, such as [Integer], decodes as the generator states. [Values]
// takes the option too, and applies it as a run does.
//
//	prop.ForAll(t, "a shipped order has a line", func(c *prop.Case) {
//		order := c.Draw(prop.Of[Order](), "order")
//		…
//	}, prop.Using(prop.SampledFrom(Pending, Paid, Shipped)))
//
// Inside g, a draw of Of of T takes the generator of T without g, with the
// run's other generators, so g does not draw itself without end.
// Using(prop.Of[Order]().Filter(valid)) narrows the orders that the run
// derives, and the run's generator of each other type applies inside them.
//
// A [Recursive] generator of T inside g is the exception. Inside its
// extension, a draw of Of takes each T from the generator of the position
// that the extension receives. The draws inside the stated generator of
// another type do so too. Once one value has drawn the base values that
// [MaxLeaves] allows, each further position takes the base, so each value
// ends.
//
// Of reads a type without the run's generators. A draw of Of reads the
// type again with them, once per run and type. A prop tag on a field whose
// type the run states then does not apply to any part of the field, and
// the draw panics inside its case with the message of Of.
//
// # Allocation contract
//
// Using allocates the generator with its type erased, its inverse, and the
// option: 4 allocations. A run allocates the list of its generators.
func Using[T any](g Generator[T]) Option {
	u := using{typ: reflect.TypeFor[T](), erased: engine.Erase(engine.Generator[T](g)), typed: g}
	return Option{set: func(c config) config {
		c.using = append(c.using, u)
		return c
	}, fromUsing: true}
}

// generators are the generators of types that a draw of [Of] reads through
// the engine in one run. Inside the stated generator of a type, the draws
// read the generators of the run without that type. Inside the extension of
// a [Recursive] generator of that type, they read the generator of the
// position as the generator of the type.
//
// # Concurrency
//
// Its methods are safe for concurrent use.
type generators struct {
	// stated are the generators by type. Each is the last generator of its
	// type that the run states, or the generator of a position.
	stated map[reflect.Type]using
	// inside are the types whose stated generators decode around the draws.
	inside map[reflect.Type]bool
	// erased are the generators of stated with their types erased. Each
	// decodes within the generators without its own type, and the reader
	// reads a type with them in place.
	erased map[reflect.Type]engine.Generator[any]
	// rest are the functions that return the generators without each type,
	// by type. Each calls without once.
	rest map[reflect.Type]func() any
	// read are the generators that the draws of Of read, by type: of T, a
	// Generator[T].
	read sync.Map
	// extensions are the generators of the draws inside the extensions of
	// recursive generators, by the generator of their positions: of a
	// *using, a *generators.
	extensions sync.Map
}

// generatorsOf returns the generators of stated inside the stated
// generators of the types of inside. A later generator of a type takes the
// place of an earlier one. It returns nil when stated and inside are empty.
func generatorsOf(stated []using, inside map[reflect.Type]bool) *generators {
	if len(stated) == 0 && len(inside) == 0 {
		return nil
	}
	gs := &generators{
		stated: make(map[reflect.Type]using, len(stated)),
		inside: inside,
		erased: make(map[reflect.Type]engine.Generator[any], len(stated)),
		rest:   make(map[reflect.Type]func() any, len(stated)),
	}
	for _, u := range stated {
		gs.stated[u.typ] = u
	}
	for t, u := range gs.stated {
		rest := sync.OnceValue(func() any { return gs.without(t) })
		gs.rest[t] = rest
		gs.erased[t] = engine.Within(u.erased, rest)
	}
	return gs
}

// without returns the generators inside the stated generator of t, as the
// engine passes them on. They are the generators of gs other than the one
// of t.
func (gs *generators) without(t reflect.Type) any {
	inside := make(map[reflect.Type]bool, len(gs.inside)+1)
	maps.Copy(inside, gs.inside)
	inside[t] = true
	rest := make([]using, 0, len(gs.stated))
	for typ, u := range gs.stated {
		if typ != t {
			rest = append(rest, u)
		}
	}
	return generatorsOf(rest, inside).engineUsing()
}

// extended returns the generators of the draws inside the extension of a
// recursive generator. position is the generator of the positions that the
// extension receives. The result is gs with position as the generator of
// its type, and position decodes with gs.
func (gs *generators) extended(position *using) *generators {
	if inner, ok := gs.extensions.Load(position); ok {
		return inner.(*generators)
	}
	inner := generatorsOf(append(slices.Collect(maps.Values(gs.stated)), *position), gs.inside)
	outer := func() any { return gs }
	inner.rest[position.typ] = outer
	inner.erased[position.typ] = engine.Within(position.erased, outer)
	stored, _ := gs.extensions.LoadOrStore(position, inner)
	return stored.(*generators)
}

// extending returns the generators of the draws inside the extension of a
// recursive generator whose positions position generates. outer are the
// generators where the extension decodes. Inside the stated generator of
// the type of position, the result is outer with position as the generator
// of the type. Anywhere else it is outer.
func (position *using) extending(outer any) any {
	gs, ok := outer.(*generators)
	if !ok || !gs.inside[position.typ] {
		return outer
	}
	return gs.extended(position)
}

// engineUsing returns gs as [engine.Settings.Using] states the generators
// of a run, and nil for a run that states none.
func (gs *generators) engineUsing() any {
	if gs == nil {
		return nil
	}
	return gs
}

// generatorOf returns the generator of T that a draw of [Of] reads under
// gs: the stated generator of T, which decodes within gs without T, and
// otherwise the generator of T read with the stated generators in place. It
// reads each type once, and returns the error of a type that the reader
// refuses.
func generatorOf[T any](gs *generators) (Generator[T], error) {
	t := reflect.TypeFor[T]()
	if g, ok := gs.read.Load(t); ok {
		return g.(Generator[T]), nil
	}
	var g Generator[T]
	if u, ok := gs.stated[t]; ok {
		g = Generator[T](engine.Within(engine.Generator[T](u.typed.(Generator[T])), gs.rest[t]))
	} else {
		read, err := of[T](registrations, gs.erased)
		if err != nil {
			return Generator[T]{}, err
		}
		g = read
	}
	stored, _ := gs.read.LoadOrStore(t, g)
	return stored.(Generator[T]), nil
}

// pickOf returns the generator of T that a draw of [Of] reads under using,
// what a run states for the generators of types. It returns false where
// using contains no generator of a type. It panics for a type that the
// reader refuses with the stated generators in place, as Of panics.
func pickOf[T any](using any) (engine.Generator[T], bool) {
	gs, ok := using.(*generators)
	if !ok || len(gs.stated) == 0 {
		return engine.Generator[T]{}, false
	}
	g, err := generatorOf[T](gs)
	if err != nil {
		panic(fmt.Sprintf("prop: Of[%v]: %v", reflect.TypeFor[T](), err))
	}
	return engine.Generator[T](g), true
}
