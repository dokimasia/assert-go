// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"reflect"
	"sync"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// using is a generator that [Using] states: the generator of one type.
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
// generator, such as [Integer], decodes as the generator states.
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
	}}
}

// generators are the generators that the options of one run state for
// types, which a draw of [Of] reads through the engine. Inside the stated
// generator of a type, the draws read the generators of the run without
// that type.
//
// # Concurrency
//
// Its methods are safe for concurrent use.
type generators struct {
	// stated are the stated generators by type, the last of each type.
	stated map[reflect.Type]using
	// erased are the stated generators by type, with their types erased,
	// each of which decodes within the generators without its own type. The
	// reader reads a type with them in place.
	erased map[reflect.Type]engine.Generator[any]
	// rest return the generators without each type, by type, as without
	// returns them, once.
	rest map[reflect.Type]func() any
	// read are the generators that the draws of Of read, by type: of T, a
	// Generator[T].
	read sync.Map
}

// generatorsOf returns the generators that stated states, a later one of a
// type in place of an earlier one, and nil for none.
func generatorsOf(stated []using) *generators {
	if len(stated) == 0 {
		return nil
	}
	gs := &generators{
		stated: make(map[reflect.Type]using, len(stated)),
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

// without returns the generators of gs other than the one of t, as the
// engine passes them on, and nil when gs states no other type.
func (gs *generators) without(t reflect.Type) any {
	rest := make([]using, 0, len(gs.stated)-1)
	for typ, u := range gs.stated {
		if typ != t {
			rest = append(rest, u)
		}
	}
	return generatorsOf(rest).engineUsing()
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
// what a run states for the generators of types, and false for a run that
// states none. It panics for a type that the reader refuses with the stated
// generators in place, as Of panics.
func pickOf[T any](using any) (engine.Generator[T], bool) {
	gs, ok := using.(*generators)
	if !ok {
		return engine.Generator[T]{}, false
	}
	g, err := generatorOf[T](gs)
	if err != nil {
		panic(fmt.Sprintf("prop: Of[%v]: %v", reflect.TypeFor[T](), err))
	}
	return engine.Generator[T](g), true
}
