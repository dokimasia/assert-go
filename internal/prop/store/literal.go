// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"bytes"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"strconv"
	"unicode/utf8"
)

// The types of the definition's typed literals.
const (
	// typeNull is null.
	typeNull = "null"
	// typeBool is a bool.
	typeBool = "bool"
	// typeInt is an integer.
	typeInt = "int"
	// typeFloat is a float.
	typeFloat = "float"
	// typeString is a string.
	typeString = "string"
	// typeBytes is a byte string, as lowercase hexadecimal.
	typeBytes = "bytes"
	// typeList is a list.
	typeList = "list"
	// typeMap is a map.
	typeMap = "map"
)

// The names of the floats that JSON has no number for.
const (
	// nameNaN is every NaN.
	nameNaN = "NaN"
	// nameInf is positive infinity.
	nameInf = "Inf"
	// nameNegInf is negative infinity.
	nameNegInf = "-Inf"
)

// The bounds of a literal.
const (
	// safeInteger is 2^53 - 1, the largest integer magnitude that a JSON
	// number states exactly in every target language. A larger one is a
	// decimal string.
	safeInteger = 9007199254740991
	// valueDepth is the most levels of objects and arrays that a draw's
	// value nests: the root object, the counterexample and the draw take
	// three of a file's 64.
	valueDepth = 61
	// maxParts is the most parts that the walk of one value visits, each
	// value one part and each byte of a string or a byte string one more.
	// It bounds the literal's size and ends the walk of a value that
	// contains itself.
	maxParts = 65536
)

// textMarshaler is the interface of a value that states itself as text.
var textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()

// nullLiteral is the literal of null.
type nullLiteral struct {
	// Type is typeNull.
	Type string `json:"type"`
}

// scalarLiteral is the literal of a bool, an integer, a float, a string or
// a byte string.
type scalarLiteral struct {
	// Type is the literal's type.
	Type string `json:"type"`
	// Value is the JSON value of the scalar.
	Value any `json:"value"`
}

// scalarList is the literal of a non-empty list whose elements are scalars
// of one type.
type scalarList struct {
	// Type is typeList.
	Type string `json:"type"`
	// Of is the type of every element.
	Of string `json:"of"`
	// Value are the JSON values of the elements.
	Value []any `json:"value"`
}

// itemList is the literal of any other list.
type itemList struct {
	// Type is typeList.
	Type string `json:"type"`
	// Items are the literals of the elements.
	Items []any `json:"items"`
}

// entryMap is the literal of a map, as pairs of a key's and a value's
// literal.
type entryMap struct {
	// Type is typeMap.
	Type string `json:"type"`
	// Entries are the pairs.
	Entries [][2]any `json:"entries"`
}

// literal is the typed literal of one value.
type literal struct {
	// form is the literal, ready for encoding/json.
	form any
	// scalar is the type of a bool, an integer, a float or a string, and
	// empty for any other literal.
	scalar string
	// plain is the JSON value of a scalar.
	plain any
	// depth is the levels of objects and arrays that the literal nests.
	depth int
}

// walk is the walk of one value's parts.
type walk struct {
	// parts counts the parts visited so far.
	parts int
}

// spend counts n more parts, and reports whether the walk is still within
// maxParts.
func (w *walk) spend(n int) bool {
	w.parts += n
	return w.parts <= maxParts
}

// literalOf returns the literal of v, and false when v has none or the
// walk passes maxParts.
func (w *walk) literalOf(v reflect.Value) (literal, bool) {
	if !w.spend(1) {
		return literal{}, false
	}
	kind := v.Kind()
	if !v.IsValid() || (kind == reflect.Pointer || kind == reflect.Interface) && v.IsNil() {
		return literal{form: nullLiteral{Type: typeNull}, depth: 1}, true
	}
	if v.Type().Implements(textMarshaler) {
		text, err := v.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil || !utf8.Valid(text) || !w.spend(len(text)) {
			return literal{}, false
		}
		return scalar(typeString, string(text)), true
	}
	if kind == reflect.Pointer || kind == reflect.Interface {
		return w.literalOf(v.Elem())
	}
	if kind == reflect.Struct {
		return w.structOf(v)
	}
	if kind == reflect.Map {
		return w.mapOf(v)
	}
	if kind == reflect.Slice || kind == reflect.Array {
		return w.listOf(v)
	}
	return w.scalarOf(v)
}

// scalarOf returns the literal of a bool, an integer, a float or a string,
// and false for a value of any other kind.
func (w *walk) scalarOf(v reflect.Value) (literal, bool) {
	if v.Kind() == reflect.Bool {
		return scalar(typeBool, v.Bool()), true
	}
	if v.CanInt() {
		n := v.Int()
		return integer(strconv.FormatInt(n, 10), n >= -safeInteger && n <= safeInteger), true
	}
	if v.CanUint() {
		n := v.Uint()
		return integer(strconv.FormatUint(n, 10), n <= safeInteger), true
	}
	if v.CanFloat() {
		return float(v.Float()), true
	}
	if v.Kind() == reflect.String && utf8.ValidString(v.String()) && w.spend(v.Len()) {
		return scalar(typeString, v.String()), true
	}
	return literal{}, false
}

// listOf returns the literal of a slice or an array: a byte string for
// bytes, and a list otherwise. A list of scalars of one type states their
// values, and any other list the literal of each element.
func (w *walk) listOf(v reflect.Value) (literal, bool) {
	if v.Type().Elem().Kind() == reflect.Uint8 {
		if !w.spend(v.Len()) {
			return literal{}, false
		}
		b := make([]byte, v.Len())
		for i := range b {
			b[i] = byte(v.Index(i).Uint())
		}
		return literal{form: scalarLiteral{Type: typeBytes, Value: hex.EncodeToString(b)}, depth: 1}, true
	}
	items := make([]literal, v.Len())
	for i := range items {
		item, ok := w.literalOf(v.Index(i))
		if !ok {
			return literal{}, false
		}
		items[i] = item
	}
	if of := commonScalar(items); of != "" {
		values := make([]any, len(items))
		for i, item := range items {
			values[i] = item.plain
		}
		return literal{form: scalarList{Type: typeList, Of: of, Value: values}, depth: 2}, true
	}
	forms := make([]any, len(items))
	deepest := 0
	for i, item := range items {
		forms[i], deepest = item.form, max(deepest, item.depth)
	}
	return literal{form: itemList{Type: typeList, Items: forms}, depth: 2 + deepest}, true
}

// mapOf returns the literal of a map: its entries, sorted by the JSON of
// their keys' literals.
func (w *walk) mapOf(v reflect.Value) (literal, bool) {
	type sorted struct {
		// key is the JSON of the key's literal.
		key []byte
		// pair are the key's and the value's literals.
		pair [2]literal
	}
	entries := make([]sorted, 0, v.Len())
	for it := v.MapRange(); it.Next(); {
		key, ok := w.literalOf(it.Key())
		if !ok {
			return literal{}, false
		}
		value, ok := w.literalOf(it.Value())
		if !ok {
			return literal{}, false
		}
		text, _ := json.Marshal(key.form)
		entries = append(entries, sorted{key: text, pair: [2]literal{key, value}})
	}
	slices.SortFunc(entries, func(a, b sorted) int { return bytes.Compare(a.key, b.key) })
	pairs := make([][2]literal, len(entries))
	for i, e := range entries {
		pairs[i] = e.pair
	}
	return mapLiteral(pairs), true
}

// structOf returns a struct's literal, which is a map of its exported
// fields in declaration order with each field's name as its key.
func (w *walk) structOf(v reflect.Value) (literal, bool) {
	var pairs [][2]literal
	for field, value := range v.Fields() {
		if !field.IsExported() {
			continue
		}
		lit, ok := w.literalOf(value)
		if !ok {
			return literal{}, false
		}
		pairs = append(pairs, [2]literal{scalar(typeString, field.Name), lit})
	}
	return mapLiteral(pairs), true
}

// Literal returns the typed literal of v for an entry's counterexample, and
// false when no typed literal states v.
//
// A value maps to a literal by its kind:
//
//   - nil and a nil pointer are null.
//   - A value whose type implements [encoding.TextMarshaler] is the string
//     of its text.
//   - A pointer is its target, and an interface its dynamic value.
//   - A bool, an integer, a float and a string are scalars. An integer
//     beyond 2^53 - 1 in magnitude is a decimal string, and a NaN or an
//     infinity is its name.
//   - A slice or an array of bytes is a byte string, and any other slice or
//     array a list.
//   - A map is its entries, sorted by the JSON of their keys, so that one
//     map has one literal.
//   - A struct is a map of its exported fields in declaration order.
//
// A string that is not UTF-8, a complex number, a channel, a function and
// an unsafe pointer have no literal, and neither has a value that contains
// one. Nor has a value whose literal would nest more than the 61 levels
// that a draw's value may take, or that has more than 65,536 parts, each
// value one part and each byte of a string or a byte string one more. A
// value that contains itself has more.
func Literal(v any) (json.RawMessage, bool) {
	var w walk
	lit, ok := w.literalOf(reflect.ValueOf(v))
	if !ok || lit.depth > valueDepth {
		return nil, false
	}
	raw, err := json.Marshal(lit.form)
	return raw, err == nil
}

// commonScalar returns the scalar type that every one of items has, and
// empty when items is empty or holds a literal of another type.
func commonScalar(items []literal) string {
	if len(items) == 0 {
		return ""
	}
	for _, item := range items[1:] {
		if item.scalar != items[0].scalar {
			return ""
		}
	}
	return items[0].scalar
}

// mapLiteral returns the literal of a map with the pairs of a key's and a
// value's literal, in order. The literal nests its object and the list of
// pairs, and each pair's list around the two literals.
func mapLiteral(pairs [][2]literal) literal {
	forms := make([][2]any, len(pairs))
	depth := 2
	for i, pair := range pairs {
		forms[i] = [2]any{pair[0].form, pair[1].form}
		depth = max(depth, 3+pair[0].depth, 3+pair[1].depth)
	}
	return literal{form: entryMap{Type: typeMap, Entries: forms}, depth: depth}
}

// scalar returns the literal of a scalar of type typ with the JSON value
// plain.
func scalar(typ string, plain any) literal {
	return literal{form: scalarLiteral{Type: typ, Value: plain}, scalar: typ, plain: plain, depth: 1}
}

// integer returns the literal of an integer with the decimal text: a JSON
// number when safe, and the text as a string beyond 2^53 - 1.
func integer(text string, safe bool) literal {
	if safe {
		return scalar(typeInt, json.Number(text))
	}
	return scalar(typeInt, text)
}

// float returns the literal of x, with the name of a NaN or an infinity.
func float(x float64) literal {
	if math.IsNaN(x) {
		return scalar(typeFloat, nameNaN)
	}
	if math.IsInf(x, 1) {
		return scalar(typeFloat, nameInf)
	}
	if math.IsInf(x, -1) {
		return scalar(typeFloat, nameNegInf)
	}
	return scalar(typeFloat, x)
}
