// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/shape"
)

// The operations whose faults name them.
const (
	shapeOfOp = "prop.ShapeOf"
	ofShapeOp = "prop.OfShape"
)

// ErrShape reports a shape that the definition's rules refuse: a shape
// file that [OfShape] reads, or a constraint of a prop tag in a type that
// [ShapeOf] reads. errors.Is matches every fault of this kind.
var ErrShape = shape.ErrShape

// derived is a Go type read into a shape file: the file's root shape with
// its definitions, the generators of the refs that name generators without
// a shape, the paths of the parts those generate, and the converter of the
// root.
type derived struct {
	// root is the root shape, with the definitions when there are any.
	root node
	// externals are the generators without a shape, by the name of their
	// refs.
	externals map[string]engine.Generator[any]
	// externalPaths are the paths of the parts that the externals generate.
	externalPaths []fault.Path
	// converter converts the root's neutral values to values of the type.
	converter converter
}

// derive reads t into a shape file under reg, with the generators that
// using states for types. It returns a fault at the part of a type that the
// reader refuses, and the fault of a zone that the platform's time-zone
// database lacks, when the shape contains a zone.
func derive(reg *registry, t reflect.Type, using map[reflect.Type]engine.Generator[any]) (derived, error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	r := newReader(reg, using)
	root, conv, err := r.read(t, nil, pathOf(t))
	if err != nil {
		return derived{}, err
	}
	if len(r.definitions) > 0 {
		root[definitionsKey] = r.definitions
	}
	if r.zoned {
		_, err = locations()
	}
	return derived{root: root, externals: r.externals, externalPaths: r.externalPaths, converter: conv}, err
}

// generator returns the generator of the shape file's neutral values, and
// an error for a constraint that the definition refuses.
func (d derived) generator() (engine.Generator[any], error) {
	document, _ := json.Marshal(d.root)
	return shape.ReadWith(document, d.externals)
}

// derivations are the generators that Of returned, by type.
var derivations sync.Map

// Of returns the generator of T: the generator that [Register] states for
// T, and otherwise the generator of T's shape, which the reader reads from
// T as the package documentation states. Two types of one shape generate
// the same values from one seed in every language of the definition. A
// draw from the generator records its value in a store entry as T's shape
// states it, so [Draws] reads the entry back.
//
// It panics for a type that the reader refuses, and the panic names the
// field. A call reads T once per process. The first call of Of for a type
// reads the registrations at that time, so a registration of a type that T
// contains panics after it.
//
// # Allocation contract
//
// Of allocates nothing for a type that a call read before. The first call
// for a type allocates the read of T and the generator of its shape.
func Of[T any]() Generator[T] {
	g, err := cachedOf[T]()
	if err != nil {
		panic(fmt.Sprintf("prop: Of[%v]: %v", reflect.TypeFor[T](), err))
	}
	return g
}

// cachedOf returns the generator of T under the process's registry, as
// [Of] does, which it reads once per process, and the error of a type that
// does not read.
func cachedOf[T any]() (Generator[T], error) {
	t := reflect.TypeFor[T]()
	if g, ok := derivations.Load(t); ok {
		return g.(Generator[T]), nil
	}
	g, err := of[T](registrations, nil)
	if err != nil {
		return Generator[T]{}, err
	}
	stored, _ := derivations.LoadOrStore(t, g)
	return stored.(Generator[T]), nil
}

// of returns the generator of T under reg, with the generators that using
// states for the types that T contains: the generator that reg states for
// T, and otherwise the generator of T's shape.
func of[T any](reg *registry, using map[reflect.Type]engine.Generator[any]) (Generator[T], error) {
	t := reflect.TypeFor[T]()
	if typed, ok := reg.typed(t); ok {
		return typed.(Generator[T]), nil
	}
	d, err := derive(reg, t, using)
	if err != nil {
		return Generator[T]{}, err
	}
	g, err := d.generator()
	if err != nil {
		return Generator[T]{}, err
	}
	conv := d.converter
	return Generator[T](g.MapBack(func(v any) T {
		var out T
		conv.store(reflect.ValueOf(&out).Elem(), v)
		return out
	}, func(value T) (any, error) {
		return conv.encode(reflect.ValueOf(&value).Elem())
	})), nil
}

// ShapeOf returns the shape of T as a shape file: the shape that the reader
// reads from T, with its definitions and the field source, which states the
// language and T's qualified name, such as
// {"language": "go", "type": "example.com/shop.Order"}. The keys are sorted
// and indented by two spaces, and the text ends in a newline, so a golden
// file of the shape changes only when T does.
//
// The first call for a type reads the registrations at that time, as [Of]
// does.
//
// # Allocation contract
//
// ShapeOf reads T on every call. It allocates the read with the path of
// each part, the generator that checks the shape file, and the text: 387
// allocations for a struct of an integer, a list of structs of two fields,
// and a pointer to a string.
//
// # Errors
//
// It returns a fault of the operation prop.ShapeOf. The fault's path starts
// at T's name and leads to the part of T that the reader refuses, or that a
// registered generator generates, which states no shape. For a constraint
// that the definition refuses, the fault is of the kind [ErrShape], and its
// path leads through the shape file.
func ShapeOf[T any]() (string, error) {
	text, err := shapeOf(registrations, reflect.TypeFor[T]())
	if err != nil {
		return "", fault.In(shapeOfOp, err)
	}
	return text, nil
}

// shapeOf returns the shape file of t under reg, as [ShapeOf] does.
func shapeOf(reg *registry, t reflect.Type) (string, error) {
	d, err := derive(reg, t, nil)
	if err != nil {
		return "", err
	}
	if len(d.externalPaths) > 0 {
		return "", refused(d.externalPaths[0], "the type has a registered generator, which states no shape")
	}
	if _, err := d.generator(); err != nil {
		return "", err
	}
	d.root[sourceKey] = node{"language": "go", "type": qualified(t)}
	var b strings.Builder
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	_ = e.Encode(d.root)
	return b.String(), nil
}

// OfShape returns the generator of the shape file text, such as one that
// [ShapeOf] or another language wrote. It decodes each shape to a Go value:
//
//   - int: the integer type of its width and sign, from int8 to int64 or
//     from uint8 to uint64, and *big.Int at width 128.
//   - float: float32 or float64 by its width.
//   - bool, char, string and bytes: bool, rune, string and []byte.
//   - list, fixed-list and set: []any.
//   - map: map[any]any.
//   - record: map[string]any of its fields.
//   - enum: [Variant].
//   - optional: nil, or the value.
//   - literal: the value of its typed literal, with a record as a
//     map[string]any and a variant as a [Variant].
//   - uuid, ip-address and decimal: uuid.UUID, netip.Addr and *big.Rat.
//   - instant, date and local-date-time: time.Time in UTC, whose fields
//     state a local date and time.
//   - zoned-date-time: time.Time in its zone.
//   - time-of-day and duration: time.Duration, from midnight for a time of
//     day.
//   - offset: int, the seconds east of UTC.
//   - zone: *time.Location.
//   - wall-time: [WallTime].
//
// An empty list, fixed-list or set decodes to a nil []any. The generator
// runs backwards from such a value, and from the value that a typed literal
// of one decodes to.
//
// # Allocation contract
//
// OfShape allocates the read of the shape file, the parse of its JSON, and
// the converters with the path of each part: 160 allocations for a record
// of an int and a bool.
//
// # Errors
//
// It returns a fault of the operation prop.OfShape: of the kind [ErrShape]
// for text that the definition's rules refuse, at the part of the shape
// file that does not read, for a map whose key shape decodes to values
// that no Go map accepts as keys, such as lists, at the key shape, and for
// a zone that the platform's time-zone database lacks, at the zone's name.
func OfShape(text string) (Generator[any], error) {
	g, err := shape.Read([]byte(text))
	if err != nil {
		return Generator[any]{}, fault.In(ofShapeOp, err)
	}
	d := json.NewDecoder(strings.NewReader(text))
	d.UseNumber()
	var root node
	_ = d.Decode(&root)
	definitions, _ := root[definitionsKey].(node)
	w := &walker{definitions: definitions, defined: make(map[string]*converter)}
	conv, err := w.convert(root, nil)
	if err == nil && w.zoned {
		_, err = locations()
	}
	if err != nil {
		return Generator[any]{}, fault.In(ofShapeOp, err)
	}
	return Generator[any](g.MapBack(func(v any) any {
		dst := reflect.New(conv.plain).Elem()
		conv.store(dst, v)
		return dst.Interface()
	}, func(v any) (any, error) {
		return conv.encode(reflect.ValueOf(v))
	})), nil
}
