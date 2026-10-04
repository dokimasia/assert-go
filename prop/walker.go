// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"reflect"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
)

// walker reads the shapes of one shape file, which the shape package has
// read, into the converters to the Go values that OfShape decodes them to.
type walker struct {
	// definitions are the file's definitions, by name.
	definitions node
	// defined are the converters of the definitions that a ref names, by
	// name. A converter's cell is in place before its definition is read,
	// so a ref inside the definition finds it.
	defined map[string]*converter
	// zoned reports whether the file contains a zone, whose locations the
	// conversions resolve.
	zoned bool
}

// convert returns the converter of the shape n at the path at, which leads
// through the shape file to n. It returns a fault at the key of a map whose
// key shape decodes to values that no Go map accepts as keys.
func (w *walker) convert(n node, at fault.Path) (converter, error) {
	kind, _ := n[shapeKey].(string)
	switch kind {
	case "bool":
		return boolConverter, nil
	case "int":
		width, _ := n[widthKey].(json.Number).Int64()
		signed, _ := n[signedKey].(bool)
		return integerConverter(int(width), signed), nil
	case "float":
		if width, _ := n[widthKey].(json.Number).Int64(); width == float32Bits {
			return floatConverter(reflect.TypeFor[float32]()), nil
		}
		return floatConverter(reflect.TypeFor[float64]()), nil
	case "string":
		return stringConverter, nil
	case "char":
		return charConverter, nil
	case "bytes":
		return bytesConverter, nil
	case "list", "fixed-list", "set":
		of, err := w.convert(n[ofKey].(node), slices.Concat(at, fault.Path{fault.Field(ofKey)}))
		if err != nil {
			return converter{}, err
		}
		if kind == "set" {
			return setConverter(of), nil
		}
		return sequenceConverter(of), nil
	case "map":
		return w.mapping(n, at)
	case "optional":
		of, err := w.convert(n[ofKey].(node), slices.Concat(at, fault.Path{fault.Field(ofKey)}))
		return optionalConverter(of, nil), err
	case "record":
		return w.record(n, at)
	case "enum":
		return w.enum(n, at)
	case "literal":
		return literalOf(n), nil
	case "ref":
		return w.ref(n)
	case "uuid":
		return uuidConverter, nil
	case "ip-address":
		return addressConverter, nil
	case "decimal":
		digits, _ := n[scaleKey].(json.Number).Int64()
		return decimalConverter(scaleOf(int(digits))), nil
	case "date":
		return dateConverter, nil
	case "offset":
		return offsetConverter, nil
	case "zone":
		w.zoned = true
		return locationConverter, nil
	}
	return w.timed(kind, n), nil
}

// timed returns the converter of a time shape n of a unit: an instant, a
// local date and time, a time of day, a duration, a zoned date and time or
// a wall time.
func (w *walker) timed(kind string, n node) converter {
	name, _ := n[unitKey].(string)
	u, _ := unitOf(name)
	switch kind {
	case "instant":
		return timeConverter(u.instant, u.instantOf)
	case "local-date-time":
		return timeConverter(u.local, u.localOf)
	case "time-of-day":
		return durationConverter(u.duration, u.timeOfDayOf)
	case "duration":
		return durationConverter(u.duration, u.durationOf)
	case "zoned-date-time":
		w.zoned = true
		return timeConverter(u.zoned, u.zonedOf)
	}
	w.zoned = true
	return wallConverter(u)
}

// mapping returns the converter of a map n at the path at. It returns a
// fault at its key for a key shape whose Go values no Go map accepts as
// keys.
func (w *walker) mapping(n node, at fault.Path) (converter, error) {
	keyAt := slices.Concat(at, fault.Path{fault.Field(keyKey)})
	keys, err := w.convert(n[keyKey].(node), keyAt)
	if err != nil {
		return converter{}, err
	}
	values, err := w.convert(n[ofKey].(node), slices.Concat(at, fault.Path{fault.Field(ofKey)}))
	if err != nil {
		return converter{}, err
	}
	if !w.keyable(n[keyKey].(node), make(map[string]bool)) {
		return converter{}, refused(keyAt, "the shape decodes to values that no Go map accepts as keys")
	}
	return mapConverter(keys, values), nil
}

// keyable reports whether every Go value of the shape n is a key of a Go
// map. seen are the definitions that the walk has entered.
func (w *walker) keyable(n node, seen map[string]bool) bool {
	switch n[shapeKey] {
	case "bytes", "list", "fixed-list", "set", "map", "record":
		return false
	case "optional":
		return w.keyable(n[ofKey].(node), seen)
	case "enum":
		for _, item := range n[variantsKey].([]any) {
			payload, shaped := item.([]any)[1].(node)
			if shaped && !w.keyable(payload, seen) {
				return false
			}
		}
	case "literal":
		for _, v := range decodedLiterals(n) {
			if v != nil && !reflect.ValueOf(plainOf(v)).Comparable() {
				return false
			}
		}
	case "ref":
		name := n[nameKey].(string)
		if seen[name] {
			return true
		}
		seen[name] = true
		return w.keyable(w.definitions[name].(node), seen)
	}
	return true
}

// record returns the converter of a record n at the path at: to a
// map[string]any of its fields' Go values. The path of a field is the pair
// of its name and its shape.
func (w *walker) record(n node, at fault.Path) (converter, error) {
	items := n[fieldsKey].([]any)
	fields := make([]field, len(items))
	for i, item := range items {
		pair := item.([]any)
		fieldAt := slices.Concat(at, fault.Path{fault.Field(fieldsKey), fault.Index(i), fault.Index(1)})
		conv, err := w.convert(pair[1].(node), fieldAt)
		if err != nil {
			return converter{}, err
		}
		fields[i] = field{index: i, name: pair[0].(string), conv: conv}
	}
	return recordConverter(fields, nil, nil), nil
}

// enum returns the converter of an enum n at the path at: to a [Variant] of
// its payload's Go value. The path of a payload is the pair of its variant's
// name and its shape.
func (w *walker) enum(n node, at fault.Path) (converter, error) {
	items := n[variantsKey].([]any)
	cases := make([]variantCase, len(items))
	for i, item := range items {
		pair := item.([]any)
		cases[i].name = pair[0].(string)
		if pair[1] == nil {
			continue
		}
		payloadAt := slices.Concat(at, fault.Path{fault.Field(variantsKey), fault.Index(i), fault.Index(1)})
		conv, err := w.convert(pair[1].(node), payloadAt)
		if err != nil {
			return converter{}, err
		}
		cases[i].payload = &conv
	}
	return enumConverter(cases, nil), nil
}

// decodedLiterals returns the values that the typed literals of a literal
// shape n decode to. The shape package has decoded each.
func decodedLiterals(n node) []any {
	items := n[valuesKey].([]any)
	out := make([]any, len(items))
	for i, item := range items {
		raw, _ := json.Marshal(item)
		out[i], _ = literal.Decode(raw)
	}
	return out
}

// literalOf returns the converter of a literal shape n: to the value of
// its typed literal, with a record as a map[string]any and a variant as a
// [Variant].
func literalOf(n node) converter {
	decoded := decodedLiterals(n)
	values := make([]reflect.Value, len(decoded))
	for i, v := range decoded {
		values[i] = reflect.ValueOf(plainOf(v))
	}
	return literalConverter(values, decoded)
}

// plainOf returns v, a value that a typed literal decodes to, with each
// record inside it as a map[string]any and each variant as a [Variant].
func plainOf(v any) any {
	switch x := v.(type) {
	case literal.Record:
		out := make(map[string]any, len(x.Fields))
		for _, f := range x.Fields {
			out[f.Name] = plainOf(f.Value)
		}
		return out
	case literal.Variant:
		return Variant{Name: x.Name, Payload: plainOf(x.Payload), HasPayload: x.HasPayload}
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = plainOf(item)
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(x))
		for key, value := range x {
			out[key] = plainOf(value)
		}
		return out
	}
	return v
}

// ref returns the converter of the definition that a ref n names, which the
// shape package has checked. The path of a definition is its name in the
// definitions.
func (w *walker) ref(n node) (converter, error) {
	name := n[nameKey].(string)
	if cell, ok := w.defined[name]; ok {
		return deferred(cell), nil
	}
	cell := &converter{}
	w.defined[name] = cell
	conv, err := w.convert(w.definitions[name].(node), fault.Path{fault.Field(definitionsKey), fault.Key(name)})
	if err != nil {
		return converter{}, err
	}
	*cell = conv
	return deferred(cell), nil
}
