// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sync"

	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// registry is a test process's registrations of generators, values and
// variants, and the types that a read looked up while it was open.
//
// A read locks mu while it reads one type, so a registration never runs
// during a read.
type registry struct {
	// mu guards the fields below.
	mu sync.Mutex
	// closed reports whether the first run of a property has closed the
	// registry.
	closed bool
	// generators are the generators that Register states, by type.
	generators map[reflect.Type]*registered
	// values are the values that RegisterValues states, by type.
	values map[reflect.Type]*literals
	// variants are the variants that RegisterVariants states, by interface.
	variants map[reflect.Type][]variant
	// consulted are the types that a read looked up while the registry was
	// open, and nil once it closed.
	consulted map[reflect.Type]bool
}

// registrations are the registrations of the test process.
var registrations = newRegistry()

// newRegistry returns an open registry without a registration.
func newRegistry() *registry {
	return &registry{
		generators: make(map[reflect.Type]*registered),
		values:     make(map[reflect.Type]*literals),
		variants:   make(map[reflect.Type][]variant),
		consulted:  make(map[reflect.Type]bool),
	}
}

// registered is a generator that Register states for a type: the generator
// with its type erased, which a shape places behind a ref, and the
// generator itself, which Of returns.
type registered struct {
	// erased is the generator with its type erased.
	erased engine.Generator[any]
	// typed is the Generator of the type.
	typed any
}

// literals are the values that RegisterValues states for a type, in order:
// each value, the typed literal of each as a shape file states it, the
// value that each literal decodes to and its canonical key, and the
// converter of the type's literal shape.
type literals struct {
	// values are the values.
	values []reflect.Value
	// stated are the typed literals, as JSON values with their numbers as
	// json.Number.
	stated []any
	// decoded are the values that the typed literals decode to.
	decoded []any
	// keys are the canonical keys of the decoded values.
	keys []string
	// converter converts a decoded value to its value, and back.
	converter converter
}

// shape returns the literal shape of the values.
func (l *literals) shape() node {
	return node{shapeKey: "literal", "values": l.stated}
}

// variant is one variant that RegisterVariants states: its name, its type,
// and whether its type is a pointer to the type that names it.
type variant struct {
	// name is the variant's name.
	name string
	// typ is the variant's type.
	typ reflect.Type
	// pointer reports whether typ is a pointer to the named type.
	pointer bool
}

// base returns the type that names the variant and whose shape is its
// payload.
func (v variant) base() reflect.Type {
	if v.pointer {
		return v.typ.Elem()
	}
	return v.typ
}

// registration is what a registry states for one type.
type registration struct {
	// generator is the generator that Register states, or nil.
	generator *registered
	// values are the values that RegisterValues states, or nil.
	values *literals
	// variants are the variants that RegisterVariants states, or nil.
	variants []variant
}

// lookup returns what reg states for t, and nothing for skip. The caller
// has locked reg.mu. While reg is open, lookup records t, so a later
// registration of t panics.
func (reg *registry) lookup(t, skip reflect.Type) registration {
	if t == skip {
		return registration{}
	}
	if !reg.closed {
		reg.consulted[t] = true
	}
	return registration{generator: reg.generators[t], values: reg.values[t], variants: reg.variants[t]}
}

// admit panics unless reg admits a registration of t by the function named
// what: reg is open, t has no registration, and no read has looked t up.
// The caller has locked reg.mu.
func (reg *registry) admit(what string, t reflect.Type) {
	switch {
	case reg.closed:
		panic(fmt.Sprintf("prop: %s[%v] after the first run of a property, which closed the registry", what, t))
	case reg.generators[t] != nil || reg.values[t] != nil || reg.variants[t] != nil:
		panic(fmt.Sprintf("prop: %s[%v] registers %v a second time", what, t, t))
	case reg.consulted[t]:
		panic(fmt.Sprintf("prop: %s[%v] after a read of %v, whose generators would not see the registration",
			what, t, t))
	}
}

// close closes reg, after which a registration panics.
func (reg *registry) close() {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.closed, reg.consulted = true, nil
}

// typed returns the Generator that Register states for t, and false when it
// states none. It records t as lookup does.
func (reg *registry) typed(t reflect.Type) (any, bool) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if g := reg.lookup(t, nil).generator; g != nil {
		return g.typed, true
	}
	return nil, false
}

// Register makes g the generator of T in the test process: [Of] returns
// it, and a type that contains T generates T with it. g may lack a shape,
// as a generator built with Map, Filter or Composite does. [ShapeOf]
// returns an error for T and for a type that contains T, because a
// registered generator states no shape.
//
// A registration applies to the whole test process and precedes every
// property, in init or in TestMain. Register panics after the first run of
// [ForAll], [Fuzz] or a form, which closes the registry. It panics for a T
// that has a registration, and for a T that [Of], [ShapeOf] or a property
// has read, because the generators read before would not see the
// registration.
func Register[T any](g Generator[T]) {
	register(registrations, g)
}

// register registers g as the generator of T in reg, as [Register] does.
func register[T any](reg *registry, g Generator[T]) {
	t := reflect.TypeFor[T]()
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.admit("Register", t)
	reg.generators[t] = &registered{erased: engine.Erase(engine.Generator[T](g)), typed: g}
}

// RegisterValues makes T's shape a literal over values, in the order given,
// so a generator of T generates one of them, and the first is the
// simplest. Each value is stated as the typed literal of its value under
// T's shape without the registration: an int for a type over an integer, a
// string for one over a string, and a record for a struct. [ShapeOf] writes
// the literals.
//
// It panics as [Register] does, and for no value, for two values of one
// literal, and for a value that no typed literal states.
func RegisterValues[T any](values ...T) {
	registerValues(registrations, values...)
}

// registerValues registers values as the values of T in reg, as
// [RegisterValues] does.
func registerValues[T any](reg *registry, values ...T) {
	t := reflect.TypeFor[T]()
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.admit("RegisterValues", t)
	if len(values) == 0 {
		panic(fmt.Sprintf("prop: RegisterValues[%v] states no value", t))
	}
	r := newReader(reg, nil)
	r.skip = t
	_, conv, err := r.read(t, nil, pathOf(t))
	if err == nil && len(r.externalPaths) > 0 {
		err = refused(r.externalPaths[0], "the type has a registered generator, whose values state no typed literal")
	}
	if err != nil {
		panic(fmt.Sprintf("prop: RegisterValues[%v]: %v", t, err))
	}
	l := &literals{}
	for i := range values {
		value := reflect.New(t).Elem()
		value.Set(reflect.ValueOf(&values[i]).Elem())
		neutral, err := conv.encode(value)
		if err != nil {
			panic(fmt.Sprintf("prop: RegisterValues[%v]: value %d: %v", t, i, err))
		}
		raw, ok := literal.Encode(neutral)
		if !ok {
			panic(fmt.Sprintf("prop: RegisterValues[%v]: value %d states no typed literal", t, i))
		}
		decoded, _ := literal.Decode(raw)
		key := literal.Canonical(decoded)
		if j := slices.Index(l.keys, key); j >= 0 {
			panic(fmt.Sprintf("prop: RegisterValues[%v]: values %d and %d state one typed literal", t, j, i))
		}
		l.values = append(l.values, value)
		l.stated = append(l.stated, jsonValue(raw))
		l.decoded = append(l.decoded, decoded)
		l.keys = append(l.keys, key)
	}
	l.converter = literalConverter(l.values, l.decoded)
	reg.values[t] = l
}

// jsonValue returns the JSON value of raw, valid JSON, with its numbers as
// json.Number.
func jsonValue(raw []byte) any {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	_ = d.Decode(&v)
	return v
}

// RegisterVariants makes the interface I's shape an enum whose variants are
// the types of variants, in the order given, so a generator of I generates
// a value of one of them, and the first is the simplest. A variant is named
// by its type's name without the package path, and a pointer by the name
// of the type it points to. Its payload is the shape of that type. A struct
// of no field that the reader reads has no payload, and its value is the
// zero value.
//
// It panics as [Register] does, and for an I that is no interface, no
// variant, a nil variant, a variant whose type has no name, and two
// variants of one name.
func RegisterVariants[I any](variants ...I) {
	registerVariants(registrations, variants...)
}

// registerVariants registers variants as the variants of I in reg, as
// [RegisterVariants] does.
func registerVariants[I any](reg *registry, variants ...I) {
	t := reflect.TypeFor[I]()
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.admit("RegisterVariants", t)
	if t.Kind() != reflect.Interface {
		panic(fmt.Sprintf("prop: RegisterVariants[%v]: %v is no interface", t, t))
	}
	if len(variants) == 0 {
		panic(fmt.Sprintf("prop: RegisterVariants[%v] states no variant", t))
	}
	out := make([]variant, len(variants))
	for i := range variants {
		v := reflect.ValueOf(&variants[i]).Elem()
		if v.IsNil() {
			panic(fmt.Sprintf("prop: RegisterVariants[%v]: variant %d is nil", t, i))
		}
		typ := v.Elem().Type()
		out[i] = variant{typ: typ, pointer: typ.Kind() == reflect.Pointer}
		out[i].name = out[i].base().Name()
		if out[i].name == "" {
			panic(fmt.Sprintf("prop: RegisterVariants[%v]: variant %d is of %v, whose type has no name", t, i, typ))
		}
		if j := slices.IndexFunc(out[:i], func(earlier variant) bool { return earlier.name == out[i].name }); j >= 0 {
			panic(fmt.Sprintf("prop: RegisterVariants[%v]: variants %d and %d are both named %s", t, j, i, out[i].name))
		}
	}
	reg.variants[t] = out
}
