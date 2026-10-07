// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"math/big"
	"net/netip"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The keys of a shape file that the reader writes.
const (
	// shapeKey names a shape's id.
	shapeKey = "shape"
	// ofKey states the shape of a collection's elements or an optional's
	// value.
	ofKey = "of"
	// keyKey states the shape of a map's keys.
	keyKey = "key"
	// nameKey states the definition that a ref names.
	nameKey = "name"
	// fieldsKey states the fields of a record.
	fieldsKey = "fields"
	// variantsKey states the variants of an enum.
	variantsKey = "variants"
	// valuesKey states the typed literals of a literal shape.
	valuesKey = "values"
	// widthKey states the width of an int and of a float.
	widthKey = "width"
	// signedKey states whether an int is signed.
	signedKey = "signed"
	// sizeKey states the size of a fixed-list.
	sizeKey = "size"
	// definitionsKey states a shape file's definitions.
	definitionsKey = "definitions"
	// sourceKey states the language and the type a shape file was read from.
	sourceKey = "source"
)

// The parameters of the shapes that the reader writes.
const (
	// defaultUnit is the unit of a time.Time, a time.Duration and a WallTime
	// whose tag states none.
	defaultUnit = "ns"
	// wideInt is the width of a *big.Int.
	wideInt = 128
	// offsetBits are the fewest bits of an integer type that reads as an
	// offset, which counts up to 64,800 seconds either way.
	offsetBits = 32
	// float32Bits is the width of a float32.
	float32Bits = 32
)

// The types that the reader reads as a shape of their own.
var (
	timeType     = reflect.TypeFor[time.Time]()
	durationType = reflect.TypeFor[time.Duration]()
	locationType = reflect.TypeFor[*time.Location]()
	uuidType     = reflect.TypeFor[uuid.UUID]()
	addrType     = reflect.TypeFor[netip.Addr]()
	bigIntType   = reflect.TypeFor[*big.Int]()
	ratType      = reflect.TypeFor[*big.Rat]()
	wallType     = reflect.TypeFor[WallTime]()
)

// node is one shape of a shape file, as encoding/json writes it.
type node = map[string]any

// reader reads Go types into one shape file, and the converters of its
// values. It reads while the registry's mu is locked.
type reader struct {
	// reg is the registry that the read consults.
	reg *registry
	// using are the generators that a form states for types, which take
	// precedence over the registry.
	using map[reflect.Type]engine.Generator[any]
	// skip is the type whose own registration the read ignores: the type
	// whose values RegisterValues reads.
	skip reflect.Type
	// definitions are the shapes of the named types that refer to
	// themselves, by definition name.
	definitions node
	// externals are the generators of the refs that name a generator without
	// a shape, by name.
	externals map[string]engine.Generator[any]
	// externalPaths are the paths of the parts that such a generator
	// generates, in the order the read found them.
	externalPaths []fault.Path
	// open are the named types whose read has not ended.
	open map[reflect.Type]*selfReference
	// defined are the named types read as definitions.
	defined map[reflect.Type]*selfReference
	// zoned reports whether the shape contains a zone, whose locations the
	// conversions resolve.
	zoned bool
}

// selfReference is a named type that its read may visit again: its name as
// a definition, the cell of its converter, which the read fills once the
// type's shape is read, and whether the read visited it again.
type selfReference struct {
	// name is the definition's name.
	name string
	// cell is the converter of the type, once its read ends.
	cell *converter
	// recursive reports whether the read of the type visited the type again.
	recursive bool
}

// newReader returns a reader that consults reg, with the generators that
// using states for types.
func newReader(reg *registry, using map[reflect.Type]engine.Generator[any]) *reader {
	return &reader{
		reg:         reg,
		using:       using,
		definitions: node{},
		externals:   make(map[string]engine.Generator[any]),
		open:        make(map[reflect.Type]*selfReference),
		defined:     make(map[reflect.Type]*selfReference),
	}
}

// pathOf returns the path that the read of t starts at: t's name, and its
// literal for a type without one.
func pathOf(t reflect.Type) fault.Path {
	if t.Name() != "" {
		return fault.Path{fault.Field(t.Name())}
	}
	return fault.Path{fault.Field(t.String())}
}

// qualified returns t's name with its package path, and its literal for a
// type without a name.
func qualified(t reflect.Type) string {
	if t.Name() == "" {
		return t.String()
	}
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}

// refTo returns a ref to name.
func refTo(name string) node {
	return node{shapeKey: "ref", nameKey: name}
}

// refused returns a fault at the path at whose reason is format with args.
func refused(at fault.Path, format string, args ...any) error {
	return fault.At(fault.New(format, args...), at...)
}

// read returns the shape of t and its converter. k are the keys of a prop
// tag that t's parts may take, and at is the path of the part of type t. A
// named type that the read visits again becomes a definition, and each
// place it occurs a ref to it.
func (r *reader) read(t reflect.Type, k *tags, at fault.Path) (node, converter, error) {
	if g, ok := r.using[t]; ok {
		return r.external(t, g, at), passed, nil
	}
	reg := r.reg.lookup(t, r.skip)
	if reg.generator != nil {
		return r.external(t, reg.generator.erased, at), passed, nil
	}
	if reg.values != nil {
		return reg.values.shape(), reg.values.converter, nil
	}
	if d := r.open[t]; d != nil {
		d.recursive = true
		return refTo(d.name), deferred(d.cell), nil
	}
	if d := r.defined[t]; d != nil {
		return refTo(d.name), deferred(d.cell), nil
	}
	if t.Name() == "" {
		return r.content(t, reg.variants, k, at)
	}
	d := &selfReference{name: qualified(t), cell: new(converter)}
	r.open[t] = d
	n, conv, err := r.content(t, reg.variants, k, at)
	delete(r.open, t)
	if err != nil || !d.recursive {
		return n, conv, err
	}
	*d.cell = conv
	r.definitions[d.name] = n
	r.defined[t] = d
	return refTo(d.name), deferred(d.cell), nil
}

// external returns a ref to the generator g of t, which has no shape, at the
// path at.
func (r *reader) external(t reflect.Type, g engine.Generator[any], at fault.Path) node {
	name := qualified(t)
	r.externals[name] = g
	r.externalPaths = append(r.externalPaths, at)
	return refTo(name)
}

// content returns the shape of t and its converter, by the reading table:
// first the types that read as a shape of their own, then by t's kind.
func (r *reader) content(t reflect.Type, variants []variant, k *tags, at fault.Path) (node, converter, error) {
	switch t {
	case timeType:
		return r.timeShape(k, at)
	case durationType:
		return durationShape(k, at)
	case locationType:
		r.zoned = true
		return node{shapeKey: "zone"}, locationConverter, nil
	case uuidType:
		return node{shapeKey: "uuid"}, uuidConverter, nil
	case addrType:
		n := node{shapeKey: "ip-address"}
		k.copy(n, numeric, versionKey)
		return n, addressConverter, nil
	case bigIntType:
		n := node{shapeKey: "int", widthKey: wideInt, signedKey: true}
		k.copy(n, numeric, minKey, maxKey)
		return n, bigIntConverter, nil
	case ratType:
		return decimalShape(k, at)
	case wallType:
		return r.wallShape(k, at)
	}
	switch t.Kind() {
	case reflect.Bool:
		return node{shapeKey: "bool"}, boolConverter, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return signedShape(t, k, at)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := node{shapeKey: "int", widthKey: t.Bits(), signedKey: false}
		k.copy(n, numeric, minKey, maxKey)
		return n, integerConverter(t.Bits(), false), nil
	case reflect.Float32, reflect.Float64:
		return floatShape(t, k)
	case reflect.String:
		n := node{shapeKey: "string"}
		k.copy(n, numeric, minSizeKey, maxSizeKey)
		k.copy(n, verbatim, alphabetKey, patternKey)
		return n, stringConverter, nil
	case reflect.Slice, reflect.Array:
		return r.sequence(t, k, at)
	case reflect.Map:
		return r.mapping(t, k, at)
	case reflect.Pointer:
		return r.optional(t, k, at)
	case reflect.Struct:
		return r.record(t, at)
	case reflect.Interface:
		if variants != nil {
			return r.enum(t, variants, at)
		}
		return nil, converter{}, refused(at, "%v has no variants, which RegisterVariants states", t)
	}
	return nil, converter{}, refused(at, "%v is no type that a shape states", t)
}

// signedShape returns the shape of a signed integer type t: an int of its
// width, an offset for the tag offset, or a char for the tag char of a rune.
func signedShape(t reflect.Type, k *tags, at fault.Path) (node, converter, error) {
	chosen, err := k.choice(offsetKey, charKey)
	switch {
	case err != nil:
		return nil, converter{}, fault.At(err, at...)
	case chosen == offsetKey && t.Bits() < offsetBits:
		return nil, converter{}, refused(at, "an offset needs an integer of %d bits or more, and %v has %d",
			offsetBits, t, t.Bits())
	case chosen == offsetKey:
		n := node{shapeKey: "offset"}
		k.copy(n, numeric, minKey, maxKey)
		return n, offsetConverter, nil
	case chosen == charKey && t.Kind() != reflect.Int32:
		return nil, converter{}, refused(at, "a char is a rune, and %v is no int32", t)
	case chosen == charKey:
		n := node{shapeKey: "char"}
		k.copy(n, verbatim, alphabetKey)
		return n, charConverter, nil
	}
	n := node{shapeKey: "int", widthKey: t.Bits(), signedKey: true}
	k.copy(n, numeric, minKey, maxKey)
	return n, integerConverter(t.Bits(), true), nil
}

// floatShape returns the shape of a float type t: a float of its width.
func floatShape(t reflect.Type, k *tags) (node, converter, error) {
	n := node{shapeKey: "float", widthKey: t.Bits()}
	k.copy(n, numeric, minKey, maxKey)
	for _, key := range []string{allowNaNKey, allowInfinityKey} {
		if k.flag(key) {
			n[key] = true
		}
	}
	if t.Bits() == float32Bits {
		return n, floatConverter(reflect.TypeFor[float32]()), nil
	}
	return n, floatConverter(reflect.TypeFor[float64]()), nil
}

// unitTag returns the unit that k states for a time shape at the path at,
// ns by default, and a fault for a unit that the definition does not have.
func unitTag(k *tags, at fault.Path) (string, timeUnit, error) {
	name, ok := k.take(unitKey)
	if !ok {
		name = defaultUnit
	}
	u, known := unitOf(name)
	if !known {
		return "", timeUnit{}, refused(at, "the unit %q is none of s, ms, us and ns", name)
	}
	return name, u, nil
}

// timeShape returns the shape of a time.Time: an instant, or the date, the
// local date and time or the zoned date and time that its tag chooses.
func (r *reader) timeShape(k *tags, at fault.Path) (node, converter, error) {
	chosen, err := k.choice(dateKey, localKey, zonedKey)
	if err != nil {
		return nil, converter{}, fault.At(err, at...)
	}
	if chosen == dateKey {
		n := node{shapeKey: "date"}
		k.copy(n, verbatim, minKey, maxKey)
		return n, dateConverter, nil
	}
	name, u, err := unitTag(k, at)
	if err != nil {
		return nil, converter{}, err
	}
	switch chosen {
	case localKey:
		n := node{shapeKey: "local-date-time", unitKey: name}
		k.copy(n, verbatim, minKey, maxKey)
		return n, timeConverter(u.local, u.localOf), nil
	case zonedKey:
		r.zoned = true
		return node{shapeKey: "zoned-date-time", unitKey: name}, timeConverter(u.zoned, u.zonedOf), nil
	}
	n := node{shapeKey: "instant", unitKey: name}
	k.copy(n, verbatim, minKey, maxKey)
	return n, timeConverter(u.instant, u.instantOf), nil
}

// durationShape returns the shape of a time.Duration: a duration, or a time
// of day for the tag time-of-day.
func durationShape(k *tags, at fault.Path) (node, converter, error) {
	ofDay := k.flag(timeOfDayKey)
	name, u, err := unitTag(k, at)
	if err != nil {
		return nil, converter{}, err
	}
	if ofDay {
		n := node{shapeKey: "time-of-day", unitKey: name}
		k.copy(n, verbatim, minKey, maxKey)
		return n, durationConverter(u.duration, u.timeOfDayOf), nil
	}
	n := node{shapeKey: "duration", unitKey: name}
	k.copy(n, numeric, minKey, maxKey)
	return n, durationConverter(u.duration, u.durationOf), nil
}

// decimalShape returns the shape of a *big.Rat: a decimal of the scale that
// its tag states, which a decimal needs.
func decimalShape(k *tags, at fault.Path) (node, converter, error) {
	text, ok := k.take(scaleKey)
	if !ok {
		return nil, converter{}, refused(at, "a *big.Rat reads as a decimal, which needs the tag key scale")
	}
	digits, err := strconv.Atoi(text)
	if err != nil || digits < 0 {
		return nil, converter{}, refused(at, "the scale %q is no count of digits", text)
	}
	n := node{shapeKey: "decimal", scaleKey: digits}
	k.copy(n, verbatim, minKey, maxKey)
	return n, decimalConverter(scaleOf(digits)), nil
}

// wallShape returns the shape of a WallTime: a wall time.
func (r *reader) wallShape(k *tags, at fault.Path) (node, converter, error) {
	name, u, err := unitTag(k, at)
	if err != nil {
		return nil, converter{}, err
	}
	r.zoned = true
	return node{shapeKey: "wall-time", unitKey: name}, wallConverter(u), nil
}

// sequence returns the shape of a slice or an array: bytes for elements of
// the type byte, a list of a slice's elements, and a fixed-list of an
// array's. A type over uint8 is no byte, so its slices and arrays read as
// lists of integers. The keys of the tag that the list does not take apply
// to its elements.
func (r *reader) sequence(t reflect.Type, k *tags, at fault.Path) (node, converter, error) {
	if literal.IsBytes(t) {
		if t.Kind() == reflect.Array {
			return node{shapeKey: "bytes", minSizeKey: t.Len(), maxSizeKey: t.Len()}, bytesConverter, nil
		}
		n := node{shapeKey: "bytes"}
		k.copy(n, numeric, minSizeKey, maxSizeKey)
		return n, bytesConverter, nil
	}
	n := node{shapeKey: "list"}
	if t.Kind() == reflect.Array {
		n = node{shapeKey: "fixed-list", sizeKey: t.Len()}
	} else {
		k.copy(n, numeric, minSizeKey, maxSizeKey)
	}
	of, elem, err := r.read(t.Elem(), k, slices.Concat(at, fault.Path{fault.Element()}))
	if err != nil {
		return nil, converter{}, err
	}
	n[ofKey] = of
	return n, sequenceConverter(elem), nil
}

// mapping returns the shape of a map type t and its converter. A map to an
// empty struct reads as a set of its keys, and the keys of the tag that the
// set does not take apply to its elements. Any other map reads as a map,
// which gives no key of the tag to its keys or its values. The path of a
// map's keys ends in key, and of its values and a set's elements in [].
func (r *reader) mapping(t reflect.Type, k *tags, at fault.Path) (node, converter, error) {
	member := t.Elem()
	set := member.Kind() == reflect.Struct && member.NumField() == 0
	n := node{shapeKey: "map"}
	if set {
		n = node{shapeKey: "set"}
	}
	k.copy(n, numeric, minSizeKey, maxSizeKey)
	keyTags, keyPath := (*tags)(nil), slices.Concat(at, fault.Path{fault.Field(keyKey)})
	if set {
		keyTags, keyPath = k, slices.Concat(at, fault.Path{fault.Element()})
	}
	key, keys, err := r.read(t.Key(), keyTags, keyPath)
	if err != nil {
		return nil, converter{}, err
	}
	if !r.keyable(t.Key(), make(map[reflect.Type]bool)) {
		return nil, converter{}, refused(keyPath, "%v has a variant that no Go map accepts as a key", t.Key())
	}
	if set {
		n[ofKey] = key
		return n, setConverter(keys), nil
	}
	of, values, err := r.read(member, nil, slices.Concat(at, fault.Path{fault.Element()}))
	if err != nil {
		return nil, converter{}, err
	}
	n[keyKey], n[ofKey] = key, of
	return n, mapConverter(keys, values), nil
}

// keyable reports whether every value of t that a read produces is a key of
// a Go map: whether every interface inside t has only variants of such
// types. The values of a generator without a shape are the generator's to
// choose.
func (r *reader) keyable(t reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Interface:
		for _, v := range r.reg.lookup(t, r.skip).variants {
			if !v.typ.Comparable() || !r.keyable(v.typ, seen) {
				return false
			}
		}
	case reflect.Struct:
		for f := range t.Fields() {
			if !r.keyable(f.Type, seen) {
				return false
			}
		}
	case reflect.Array:
		return r.keyable(t.Elem(), seen)
	}
	return true
}

// optional returns the shape of a pointer type t: an optional of the shape
// of the type it points to, which takes the keys of the tag.
func (r *reader) optional(t reflect.Type, k *tags, at fault.Path) (node, converter, error) {
	of, inner, err := r.read(t.Elem(), k, at)
	if err != nil {
		return nil, converter{}, err
	}
	return node{shapeKey: "optional", ofKey: of}, optionalConverter(inner, t), nil
}

// fieldName returns the name of f in a record: the name that its json tag
// states, and its Go name for a tag that states none.
func fieldName(f reflect.StructField) string {
	text := f.Tag.Get("json")
	name, _, _ := strings.Cut(text, ",")
	if text == "-" || name == "" {
		return f.Name
	}
	return name
}

// readable reports whether the reader reads f: an exported field whose prop
// tag is not -.
func readable(f reflect.StructField) bool {
	return f.IsExported() && f.Tag.Get("prop") != "-"
}

// record returns the shape of a struct type t and its converter. The shape
// is a record of the fields that the reader reads, in declaration order,
// each with the keys of its prop tag. A field that it does not read keeps
// its zero value. The path of a field ends in its Go name.
func (r *reader) record(t reflect.Type, at fault.Path) (node, converter, error) {
	var fields []field
	var shapes []any
	var unread []int
	for i := range t.NumField() {
		f := t.Field(i)
		if !readable(f) {
			unread = append(unread, i)
			continue
		}
		fieldAt := slices.Concat(at, fault.Path{fault.Field(f.Name)})
		k, err := parseTags(f.Tag.Get("prop"))
		if err != nil {
			return nil, converter{}, fault.At(err, fieldAt...)
		}
		n, conv, err := r.read(f.Type, k, fieldAt)
		if err != nil {
			return nil, converter{}, err
		}
		if key := k.left(); key != "" {
			return nil, converter{}, refused(fieldAt, "the tag key %s applies to no part of %v", key, f.Type)
		}
		name := fieldName(f)
		if j := slices.IndexFunc(fields, func(earlier field) bool { return earlier.name == name }); j >= 0 {
			return nil, converter{}, refused(at, "the fields %s and %s are both named %q",
				t.Field(fields[j].index).Name, f.Name, name)
		}
		fields = append(fields, field{index: i, name: name, conv: conv})
		shapes = append(shapes, []any{name, n})
	}
	if len(fields) == 0 {
		return nil, converter{}, refused(at, "%v has no exported field to read", t)
	}
	return node{shapeKey: "record", fieldsKey: shapes}, recordConverter(fields, unread, t), nil
}

// enum returns the shape of an interface type t with registered variants:
// an enum of the variants, in order, each with the shape of its type as its
// payload, and none for a struct of no field that the reader reads. The
// path of a payload ends in its variant.
func (r *reader) enum(t reflect.Type, variants []variant, at fault.Path) (node, converter, error) {
	shapes := make([]any, len(variants))
	cases := make([]variantCase, len(variants))
	for i, v := range variants {
		cases[i] = variantCase{name: v.name, registered: v}
		base := v.base()
		if !hasPayload(base) {
			shapes[i] = []any{v.name, nil}
			continue
		}
		n, conv, err := r.read(base, nil, slices.Concat(at, fault.Path{fault.Variant(v.name)}))
		if err != nil {
			return nil, converter{}, err
		}
		shapes[i], cases[i].payload = []any{v.name, n}, &conv
	}
	return node{shapeKey: "enum", variantsKey: shapes}, enumConverter(cases, t), nil
}

// hasPayload reports whether a variant of type t has a payload: whether t
// is no struct, or a struct with a field that the reader reads.
func hasPayload(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return true
	}
	for f := range t.Fields() {
		if readable(f) {
			return true
		}
	}
	return false
}
