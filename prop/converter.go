// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"cmp"
	"math/big"
	"net/netip"
	"reflect"
	"slices"
	"time"
	"uuid"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The Go types that OfShape decodes the shapes of several Go types to.
var (
	anyType     = reflect.TypeFor[any]()
	anySlice    = reflect.TypeFor[[]any]()
	anyMap      = reflect.TypeFor[map[any]any]()
	fieldMap    = reflect.TypeFor[map[string]any]()
	variantType = reflect.TypeFor[Variant]()
)

// converter converts the neutral values of one shape to Go values and back.
// The reader builds one for the Go type that it reads, and OfShape one for
// the Go type that it decodes the shape to, each from the constructor of
// the shape in this file.
type converter struct {
	// plain is the Go type that OfShape decodes the shape to, which a
	// target of an interface type receives. It is an interface type for a
	// converter that sets such a target itself.
	plain reflect.Type
	// set stores the Go value of v, a neutral value of the shape, in dst, a
	// settable value of a Go type that the shape converts to.
	set func(dst reflect.Value, v any)
	// neutral returns the neutral value of src, a value of a Go type that
	// the shape converts to, and a fault of the kind engine.ErrCannotInvert
	// for a src that no neutral value states. A src of another Go type
	// passes unchanged, for the shape's inverse to refuse.
	neutral func(src reflect.Value) (any, error)
}

// store stores the Go value of v in dst: of dst's type, and of c.plain for
// a dst of an interface type.
func (c converter) store(dst reflect.Value, v any) {
	if dst.Kind() == reflect.Interface && c.plain.Kind() != reflect.Interface {
		into := reflect.New(c.plain).Elem()
		c.set(into, v)
		dst.Set(into)
		return
	}
	c.set(dst, v)
}

// encode returns the neutral value of src, as c.neutral does, of the value
// inside src when src is of an interface type and not nil.
func (c converter) encode(src reflect.Value) (any, error) {
	if src.Kind() == reflect.Interface && !src.IsNil() {
		src = src.Elem()
	}
	return c.neutral(src)
}

// invert returns the neutral value of src, a value of the Go type that the
// converter reads, as encode does. It returns a fault of the kind
// engine.ErrCannotInvert for a src that refers to itself, as selfReferent
// finds it, which no generator generates and whose conversion would not end.
func (c converter) invert(src reflect.Value) (any, error) {
	if err := selfReferent(src, map[reference]bool{}); err != nil {
		return nil, err
	}
	return c.encode(src)
}

// reference is a pointer, a map or a slice that a walk of a value is
// inside: the address it refers to, and its type.
type reference struct {
	// at is the address that the reference refers to.
	at uintptr
	// typ is the type of the reference.
	typ reflect.Type
}

// selfReferent returns a fault of the kind engine.ErrCannotInvert at the
// first pointer, map or slice in v that refers to a value that contains it,
// and nil when v contains none. inside are the references of the values
// that the walk is inside. The walk follows the exported fields of a struct,
// and the keys and the values of a map, as the converters do.
func selfReferent(v reflect.Value, inside map[reference]bool) error {
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if v.IsNil() {
			return nil
		}
		r := reference{at: v.Pointer(), typ: v.Type()}
		if inside[r] {
			return uninvertible("the value refers to a value that contains it")
		}
		inside[r] = true
		defer delete(inside, r)
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return selfReferent(v.Elem(), inside)
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if err := selfReferent(v.Index(i), inside); err != nil {
				return fault.At(err, fault.Index(i))
			}
		}
	case reflect.Map:
		for key, value := range v.Seq2() {
			if err := cmp.Or(selfReferent(key, inside), selfReferent(value, inside)); err != nil {
				return fault.At(err, fault.Key(key.Interface()))
			}
		}
	case reflect.Struct:
		for i := range v.NumField() {
			f := v.Type().Field(i)
			if !f.IsExported() {
				continue
			}
			if err := selfReferent(v.Field(i), inside); err != nil {
				return fault.At(err, fault.Field(f.Name))
			}
		}
	}
	return nil
}

// unchanged returns src as it is, the neutral value of a value of no Go type
// that a converter converts, and nil for no value.
func unchanged(src reflect.Value) (any, error) {
	if !src.IsValid() {
		return nil, nil
	}
	return src.Interface(), nil
}

// uninvertible returns a fault of the kind engine.ErrCannotInvert whose
// reason is format with args.
func uninvertible(format string, args ...any) *fault.Error {
	return fault.Of(engine.ErrCannotInvert, format, args...)
}

// scalar returns the converter of a shape whose Go value is of one type:
// goValue returns the Go value of a neutral value, which converts to the
// type of a target, and of returns the neutral value of a src that is
// reports to be of the shape's Go type.
func scalar(
	plain reflect.Type,
	goValue func(v any) any,
	is func(src reflect.Value) bool,
	of func(src reflect.Value) (any, error),
) converter {
	return converter{
		plain: plain,
		set:   func(dst reflect.Value, v any) { assign(dst, goValue(v)) },
		neutral: func(src reflect.Value) (any, error) {
			if !is(src) {
				return unchanged(src)
			}
			return of(src)
		},
	}
}

// assign stores x in dst, a settable value of a type that x converts to. It
// stores an integer and a float by its kind, which allocates nothing, where
// a conversion allocates the converted number.
func assign(dst reflect.Value, x any) {
	switch x := x.(type) {
	case int64:
		dst.SetInt(x)
	case int32:
		dst.SetInt(int64(x))
	case uint64:
		dst.SetUint(x)
	case float64:
		dst.SetFloat(x)
	case float32:
		dst.SetFloat(float64(x))
	default:
		dst.Set(reflect.ValueOf(x).Convert(dst.Type()))
	}
}

// ofKind returns the test of a src of the kind kind.
func ofKind(kind reflect.Kind) func(src reflect.Value) bool {
	return func(src reflect.Value) bool { return src.Kind() == kind }
}

// ofType returns the test of a src of the type t.
func ofType(t reflect.Type) func(src reflect.Value) bool {
	return func(src reflect.Value) bool { return src.IsValid() && src.Type() == t }
}

// same returns the neutral value v of a shape whose Go value is the neutral
// value itself.
func same(v any) any { return v }

// The converters of the shapes whose Go value is one type, and whose
// conversions take no parameter of the shape.
var (
	boolConverter = scalar(reflect.TypeFor[bool](), same, ofKind(reflect.Bool),
		func(src reflect.Value) (any, error) { return src.Bool(), nil })
	stringConverter = scalar(reflect.TypeFor[string](), same, ofKind(reflect.String),
		func(src reflect.Value) (any, error) { return src.String(), nil })
	charConverter = scalar(reflect.TypeFor[rune](), func(v any) any { return char(v) }, ofKind(reflect.Int32),
		func(src reflect.Value) (any, error) { return charOf(rune(src.Int())) })
	bigIntConverter = scalar(bigIntType, same, ofType(bigIntType), func(src reflect.Value) (any, error) {
		n := src.Interface().(*big.Int)
		if n == nil {
			return nil, uninvertible("a nil %T is no integer", n)
		}
		return n, nil
	})
	uuidConverter = scalar(uuidType, func(v any) any { return uuidValue(v) }, ofType(uuidType),
		func(src reflect.Value) (any, error) { return uuidOf(src.Interface().(uuid.UUID)), nil })
	addressConverter = scalar(addrType, func(v any) any { return address(v) }, ofType(addrType),
		func(src reflect.Value) (any, error) { return addressOf(src.Interface().(netip.Addr)) })
	locationConverter = scalar(locationType, func(v any) any { return location(v) }, ofType(locationType),
		func(src reflect.Value) (any, error) { return zoneOf(src.Interface().(*time.Location)) })
	dateConverter = timeConverter(date, dateOf)
	// offsetConverter converts an offset's seconds to an integer, an int for
	// OfShape.
	offsetConverter = signedConverter(reflect.TypeFor[int]())
)

// integerTypes are the Go types of the int shape of each width up to 64,
// signed and unsigned, which OfShape decodes them to.
var integerTypes = map[bool]map[int]reflect.Type{
	true: {
		8: reflect.TypeFor[int8](), 16: reflect.TypeFor[int16](),
		32: reflect.TypeFor[int32](), 64: reflect.TypeFor[int64](),
	},
	false: {
		8: reflect.TypeFor[uint8](), 16: reflect.TypeFor[uint16](),
		32: reflect.TypeFor[uint32](), 64: reflect.TypeFor[uint64](),
	},
}

// integerConverter returns the converter of an int shape of the width and
// the sign: to an integer type, and at width 128 to a *big.Int.
func integerConverter(width int, signed bool) converter {
	plain, sized := integerTypes[signed][width]
	switch {
	case !sized:
		return bigIntConverter
	case signed:
		return signedConverter(plain)
	}
	return unsignedConverter(plain)
}

// signedConverter returns the converter of a signed int shape, whose Go
// value is of the type plain for OfShape.
func signedConverter(plain reflect.Type) converter {
	return scalar(plain, same, reflect.Value.CanInt, func(src reflect.Value) (any, error) { return src.Int(), nil })
}

// unsignedConverter returns the converter of an unsigned int shape, whose
// Go value is of the type plain for OfShape.
func unsignedConverter(plain reflect.Type) converter {
	return scalar(plain, same, reflect.Value.CanUint, func(src reflect.Value) (any, error) { return src.Uint(), nil })
}

// floatConverter returns the converter of a float shape whose Go value is
// of the type plain: a float32 or a float64.
func floatConverter(plain reflect.Type) converter {
	if plain.Kind() == reflect.Float32 {
		return scalar(plain, same, ofKind(reflect.Float32),
			func(src reflect.Value) (any, error) { return float32(src.Float()), nil })
	}
	return scalar(plain, same, ofKind(reflect.Float64),
		func(src reflect.Value) (any, error) { return src.Float(), nil })
}

// timeConverter returns the converter of a shape whose Go value is a
// time.Time, with the conversions to and from its neutral value.
func timeConverter(value func(any) time.Time, of func(time.Time) (any, error)) converter {
	return scalar(timeType, func(v any) any { return value(v) }, ofType(timeType),
		func(src reflect.Value) (any, error) { return of(src.Interface().(time.Time)) })
}

// durationConverter returns the converter of a shape whose Go value is a
// time.Duration, with the conversions to and from its neutral value.
func durationConverter(value func(any) time.Duration, of func(time.Duration) (any, error)) converter {
	return scalar(durationType, func(v any) any { return value(v) }, ofType(durationType),
		func(src reflect.Value) (any, error) { return of(time.Duration(src.Int())) })
}

// decimalConverter returns the converter of a decimal shape of the scale s.
func decimalConverter(s scale) converter {
	return scalar(ratType, func(v any) any { return s.decimal(v) }, ofType(ratType),
		func(src reflect.Value) (any, error) { return s.decimalOf(src.Interface().(*big.Rat)) })
}

// wallConverter returns the converter of a wall-time shape at the unit u.
func wallConverter(u timeUnit) converter {
	return scalar(wallType, func(v any) any { return u.wall(v) }, ofType(wallType),
		func(src reflect.Value) (any, error) { return u.wallOf(src.Interface().(WallTime)) })
}

// bytesConverter is the converter of a bytes shape: to a slice or an array
// of bytes, of which an empty slice is nil.
var bytesConverter = converter{
	plain: reflect.TypeFor[[]byte](),
	set: func(dst reflect.Value, v any) {
		b := reflect.ValueOf(v)
		if dst.Kind() == reflect.Array {
			reflect.Copy(dst, b)
			return
		}
		if b.Len() > 0 {
			dst.Set(b.Convert(dst.Type()))
		}
	},
	neutral: func(src reflect.Value) (any, error) {
		if !literal.IsBytes(src.Type()) {
			return unchanged(src)
		}
		b := make([]byte, src.Len())
		reflect.Copy(reflect.ValueOf(b), src)
		return b, nil
	},
}

// passed is the converter of a generator without a shape, whose values are
// Go values already.
var passed = converter{
	plain: anyType,
	set: func(dst reflect.Value, v any) {
		if v != nil {
			dst.Set(reflect.ValueOf(v))
		}
	},
	neutral: unchanged,
}

// deferred returns the converter that converts as cell does, once the read
// of its shape has filled it.
func deferred(cell *converter) converter {
	return converter{
		plain:   anyType,
		set:     func(dst reflect.Value, v any) { cell.store(dst, v) },
		neutral: func(src reflect.Value) (any, error) { return cell.encode(src) },
	}
}

// sequenceConverter returns the converter of a list or a fixed-list, with
// elem the converter of its elements: to a slice, of which an empty one is
// nil, or to an array. The fault of an element is at its index.
func sequenceConverter(elem converter) converter {
	return converter{
		plain: anySlice,
		set: func(dst reflect.Value, v any) {
			items := v.([]any)
			if dst.Kind() == reflect.Slice {
				if len(items) == 0 {
					return
				}
				dst.Set(reflect.MakeSlice(dst.Type(), len(items), len(items)))
			}
			for i, item := range items {
				elem.store(dst.Index(i), item)
			}
		},
		neutral: func(src reflect.Value) (any, error) {
			if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
				return unchanged(src)
			}
			out := make([]any, src.Len())
			for i := range out {
				v, err := elem.encode(src.Index(i))
				if err != nil {
					return nil, fault.At(err, fault.Index(i))
				}
				out[i] = v
			}
			return out, nil
		},
	}
}

// setConverter returns the converter of a set, with elem the converter of
// its elements: to a map whose keys are its elements and whose values are
// empty structs, and to a slice as a list's converter converts. The fault
// of an element of a map is at the element.
func setConverter(elem converter) converter {
	list := sequenceConverter(elem)
	return converter{
		plain: anySlice,
		set: func(dst reflect.Value, v any) {
			if dst.Kind() != reflect.Map {
				list.set(dst, v)
				return
			}
			items := v.([]any)
			t := dst.Type()
			member := reflect.New(t.Elem()).Elem()
			m := reflect.MakeMapWithSize(t, len(items))
			for _, item := range items {
				key := reflect.New(t.Key()).Elem()
				elem.store(key, item)
				m.SetMapIndex(key, member)
			}
			dst.Set(m)
		},
		neutral: func(src reflect.Value) (any, error) {
			if src.Kind() != reflect.Map {
				return list.neutral(src)
			}
			out := make([]any, 0, src.Len())
			for key := range src.Seq() {
				v, err := elem.encode(key)
				if err != nil {
					return nil, fault.At(err, fault.Key(key.Interface()))
				}
				out = append(out, v)
			}
			return out, nil
		},
	}
}

// mapConverter returns the converter of a map, with keys and values the
// converters of its keys and its values: to a map. Its neutral value is a
// [literal.Pairs] of a Go map's entries or of the entries of the Pairs that
// a typed literal decodes to. The fault of a value is at its key, and a key
// that converts to no key of the shape has a fault of its own at the key.
func mapConverter(keys, values converter) converter {
	return converter{
		plain: anyMap,
		set: func(dst reflect.Value, v any) {
			pairs := v.(literal.Pairs)
			t := dst.Type()
			m := reflect.MakeMapWithSize(t, len(pairs.Entries))
			for _, e := range pairs.Entries {
				key, value := reflect.New(t.Key()).Elem(), reflect.New(t.Elem()).Elem()
				keys.store(key, e.Key)
				values.store(value, e.Value)
				m.SetMapIndex(key, value)
			}
			dst.Set(m)
		},
		neutral: func(src reflect.Value) (any, error) {
			entries, ok := entriesOf(src)
			if !ok {
				return unchanged(src)
			}
			out := literal.Pairs{Entries: make([]literal.Entry, len(entries))}
			for i, e := range entries {
				k, err := keys.encode(e[0])
				if err != nil {
					return nil, fault.At(uninvertible("the key converts to no key of the shape").Because(err),
						fault.Key(unwrapped(e[0])))
				}
				v, err := values.encode(e[1])
				if err != nil {
					return nil, fault.At(err, fault.Key(unwrapped(e[0])))
				}
				out.Entries[i] = literal.Entry{Key: k, Value: v}
			}
			return out, nil
		},
	}
}

// entriesOf returns the key and the value of each entry of src, a Go map or
// a [literal.Pairs], and false for any other value.
func entriesOf(src reflect.Value) ([][2]reflect.Value, bool) {
	if src.IsValid() && src.Type() == pairsType {
		pairs := src.Interface().(literal.Pairs)
		out := make([][2]reflect.Value, len(pairs.Entries))
		for i, e := range pairs.Entries {
			out[i] = [2]reflect.Value{reflect.ValueOf(e.Key), reflect.ValueOf(e.Value)}
		}
		return out, true
	}
	if src.Kind() != reflect.Map {
		return nil, false
	}
	out := make([][2]reflect.Value, 0, src.Len())
	for key, value := range src.Seq2() {
		out = append(out, [2]reflect.Value{key, value})
	}
	return out, true
}

// pairsType is the type of the value that a typed literal of a map decodes
// to.
var pairsType = reflect.TypeFor[literal.Pairs]()

// optionalConverter returns the converter of an optional, with inner the
// converter of its value: to a pointer of the type pointer, which the
// reader reads, or to the value or nil, which OfShape decodes to when
// pointer is nil.
func optionalConverter(inner converter, pointer reflect.Type) converter {
	return converter{
		plain: anyType,
		set: func(dst reflect.Value, v any) {
			if v == nil {
				return
			}
			if dst.Kind() != reflect.Pointer {
				inner.store(dst, v)
				return
			}
			p := reflect.New(dst.Type().Elem())
			inner.store(p.Elem(), v)
			dst.Set(p)
		},
		neutral: func(src reflect.Value) (any, error) {
			if pointer == nil || !src.IsValid() || src.Type() != pointer {
				if engine.Absent(unwrapped(src)) {
					return nil, nil
				}
				return inner.encode(src)
			}
			if src.IsNil() {
				return nil, nil
			}
			v, err := inner.encode(src.Elem())
			if err == nil && v == nil {
				return nil, uninvertible("%v points to an absent value, which an optional states as absent", src.Type())
			}
			return v, err
		},
	}
}

// unwrapped returns the value inside src, and nil for no value.
func unwrapped(src reflect.Value) any {
	if !src.IsValid() {
		return nil
	}
	return src.Interface()
}

// field is one field of a record: its index in a struct that the reader
// reads, its name in the record, and its converter.
type field struct {
	// index is the field's index in the struct.
	index int
	// name is the field's name in the record.
	name string
	// conv is the field's converter.
	conv converter
}

// recordConverter returns the converter of a record of fields: to a struct
// of the type structured, which the reader reads, with the fields of the
// indices unread, which the record states no value for, and to a
// map[string]any of the fields, which OfShape decodes to when structured is
// nil. Its neutral value is a [literal.Record] of a struct's fields, of a
// map's entries, or of the fields of the Record that a typed literal
// decodes to. The fault of a field is at its name.
func recordConverter(fields []field, unread []int, structured reflect.Type) converter {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.name
	}
	return converter{
		plain: fieldMap,
		set: func(dst reflect.Value, v any) {
			r := v.(literal.Record)
			if dst.Kind() == reflect.Struct {
				for i, f := range fields {
					f.conv.store(dst.Field(f.index), r.Fields[i].Value)
				}
				return
			}
			m := reflect.MakeMapWithSize(dst.Type(), len(fields))
			for i, f := range fields {
				value := reflect.New(dst.Type().Elem()).Elem()
				f.conv.store(value, r.Fields[i].Value)
				m.SetMapIndex(reflect.ValueOf(f.name), value)
			}
			dst.Set(m)
		},
		neutral: func(src reflect.Value) (any, error) {
			var values []reflect.Value
			if structured != nil && src.IsValid() && src.Type() == structured {
				for _, i := range unread {
					if !src.Field(i).IsZero() {
						return nil, fault.At(
							uninvertible("the field is not zero, and the shape states no value for it"),
							fault.Field(structured.Field(i).Name),
						)
					}
				}
				for _, f := range fields {
					values = append(values, src.Field(f.index))
				}
			} else if stated, ok := fieldValues(unwrapped(src), names); ok {
				for _, v := range stated {
					values = append(values, reflect.ValueOf(v))
				}
			} else {
				return unchanged(src)
			}
			out := literal.Record{Fields: make([]literal.Field, len(fields))}
			for i, f := range fields {
				v, err := f.conv.encode(values[i])
				if err != nil {
					return nil, fault.At(err, fault.Field(f.name))
				}
				out.Fields[i] = literal.Field{Name: f.name, Value: v}
			}
			return out, nil
		},
	}
}

// fieldValues returns the value of each of the names, in order, of v: a
// [literal.Record] of those fields in that order, or a map[string]any of
// those keys. It reports false for any other value.
func fieldValues(v any, names []string) ([]any, bool) {
	values := make([]any, len(names))
	switch r := v.(type) {
	case literal.Record:
		if len(r.Fields) != len(names) {
			return nil, false
		}
		for i, f := range r.Fields {
			if f.Name != names[i] {
				return nil, false
			}
			values[i] = f.Value
		}
		return values, true
	case map[string]any:
		if len(r) != len(names) {
			return nil, false
		}
		for i, name := range names {
			value, present := r[name]
			if !present {
				return nil, false
			}
			values[i] = value
		}
		return values, true
	}
	return nil, false
}

// variantCase is one variant of an enum: its name, the converter of its
// payload, and nil for a variant without one, and for the reader the type
// that RegisterVariants states.
type variantCase struct {
	// name is the variant's name.
	name string
	// payload is the converter of the payload, and nil for a variant
	// without one.
	payload *converter
	// registered is the variant that RegisterVariants states, which the
	// reader reads.
	registered variant
}

// enumConverter returns the converter of an enum of cases: to a value of
// the interface iface, of the variant that RegisterVariants states, which
// the reader reads, and to a [Variant], which OfShape decodes to when iface
// is nil. Its neutral value is a [literal.Variant] of a variant's value, of
// a Variant, or of the Variant that a typed literal decodes to. The fault of
// a payload is at its variant.
func enumConverter(cases []variantCase, iface reflect.Type) converter {
	plain := variantType
	if iface != nil {
		plain = iface
	}
	return converter{
		plain: plain,
		set: func(dst reflect.Value, v any) {
			chosen := v.(literal.Variant)
			i := slices.IndexFunc(cases, func(c variantCase) bool { return c.name == chosen.Name })
			if iface == nil {
				out := Variant{Name: chosen.Name, HasPayload: chosen.HasPayload}
				if chosen.HasPayload {
					p := reflect.New(anyType).Elem()
					cases[i].payload.store(p, chosen.Payload)
					out.Payload = p.Interface()
				}
				dst.Set(reflect.ValueOf(out))
				return
			}
			registered := cases[i].registered
			value := reflect.New(registered.base())
			if cases[i].payload != nil {
				cases[i].payload.store(value.Elem(), chosen.Payload)
			}
			if !registered.pointer {
				value = value.Elem()
			}
			dst.Set(value)
		},
		neutral: func(src reflect.Value) (any, error) {
			if chosen, ok := variantOf(unwrapped(src)); ok {
				return chosenNeutral(cases, chosen)
			}
			if iface == nil {
				return unchanged(src)
			}
			return registeredNeutral(cases, iface, src)
		},
	}
}

// variantOf returns v, a [Variant] or a [literal.Variant], as a
// literal.Variant, and false for any other value.
func variantOf(v any) (literal.Variant, bool) {
	switch chosen := v.(type) {
	case Variant:
		return literal.Variant{Name: chosen.Name, Payload: chosen.Payload, HasPayload: chosen.HasPayload}, true
	case literal.Variant:
		return chosen, true
	}
	return literal.Variant{}, false
}

// chosenNeutral returns the neutral value of chosen, a variant whose payload
// is a Go value or the value that a typed literal decodes to. A variant
// that the enum does not have, and a payload that its variant does not
// take, pass unchanged, for the shape's inverse to refuse.
func chosenNeutral(cases []variantCase, chosen literal.Variant) (any, error) {
	i := slices.IndexFunc(cases, func(c variantCase) bool { return c.name == chosen.Name })
	if i < 0 || !chosen.HasPayload || cases[i].payload == nil {
		return chosen, nil
	}
	payload, err := cases[i].payload.encode(reflect.ValueOf(chosen.Payload))
	if err != nil {
		return nil, fault.At(err, fault.Variant(chosen.Name))
	}
	chosen.Payload = payload
	return chosen, nil
}

// registeredNeutral returns the neutral value of src, a value of the
// interface iface: the variant that RegisterVariants states for its type.
func registeredNeutral(cases []variantCase, iface reflect.Type, src reflect.Value) (any, error) {
	if src.Kind() == reflect.Interface {
		return nil, uninvertible("a nil %v is no variant", iface)
	}
	i := slices.IndexFunc(cases, func(c variantCase) bool { return c.registered.typ == src.Type() })
	if i < 0 {
		return nil, uninvertible("%v is no variant of %v that RegisterVariants states", src.Type(), iface)
	}
	c := cases[i]
	value := src
	if c.registered.pointer {
		if value.IsNil() {
			return nil, uninvertible("a nil %v is no variant", value.Type())
		}
		value = value.Elem()
	}
	if c.payload == nil {
		if !value.IsZero() {
			return nil, uninvertible("%v is not zero, and the variant %s has no payload", value.Type(), c.name)
		}
		return literal.Variant{Name: c.name}, nil
	}
	payload, err := c.payload.encode(value)
	if err != nil {
		return nil, fault.At(err, fault.Variant(c.name))
	}
	return literal.Variant{Name: c.name, Payload: payload, HasPayload: true}, nil
}

// literalConverter returns the converter of a literal shape: from the
// neutral value of one of decoded, the values that its typed literals decode
// to, to the Go value at its index in values, and back. A Go value matches
// a value of values as the property engine compares generated values, so
// -0 matches -0 and not +0. A value that is none of values passes
// unchanged, for the shape's inverse to refuse.
func literalConverter(values []reflect.Value, decoded []any) converter {
	keys := make([]string, len(decoded))
	for i, v := range decoded {
		keys[i] = literal.Canonical(v)
	}
	return converter{
		plain: anyType,
		set: func(dst reflect.Value, v any) {
			if value := values[slices.Index(keys, literal.Canonical(v))]; value.IsValid() {
				dst.Set(value)
			}
		},
		neutral: func(src reflect.Value) (any, error) {
			given := unwrapped(src)
			i := slices.IndexFunc(values, func(value reflect.Value) bool {
				return engine.SameValue(given, unwrapped(value))
			})
			if i < 0 {
				return unchanged(src)
			}
			return decoded[i], nil
		},
	}
}
