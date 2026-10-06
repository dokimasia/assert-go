// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal

import (
	"bytes"
	"context"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/text"
)

// The bounds of a literal that [Encode] writes.
const (
	// valueDepth is the most levels of objects and arrays that a value's
	// literal nests. A store entry's root object, its counterexample and the
	// draw take three of a file's 64.
	valueDepth = 61
	// maxParts is the most parts that the walk of one value visits, each
	// value one part and each byte of a string, a byte string or a field's
	// name one more. It bounds the literal's size and ends the walk of a
	// value that contains itself.
	maxParts = 65536
)

// The types that [Encode] writes in a form of their own.
var (
	// bigPointer and bigValue are the types of an integer of any size.
	bigPointer = reflect.TypeFor[*big.Int]()
	bigValue   = reflect.TypeFor[big.Int]()
	// safeBig is 2^53 - 1 as an integer of any size.
	safeBig = big.NewInt(safeInteger)
	// textMarshaler is the interface of a value that states itself as text.
	textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()
	// recordType is the type of a record.
	recordType = reflect.TypeFor[Record]()
	// pairsType is the type of a map that keeps its entries' order.
	pairsType = reflect.TypeFor[Pairs]()
	// variantType is the type of a variant.
	variantType = reflect.TypeFor[Variant]()
)

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

// absentMap is the literal of an absent map from strings to a scalar type:
// a map with key and of, and a null value.
type absentMap struct {
	// Type is typeMap.
	Type string `json:"type"`
	// Key is typeString.
	Key string `json:"key"`
	// Of is the type of every value.
	Of string `json:"of"`
	// Value is nil, which encodes as null.
	Value map[string]any `json:"value"`
}

// entryMap is the literal of a map, as pairs of a key's and a value's
// literal.
type entryMap struct {
	// Type is typeMap.
	Type string `json:"type"`
	// Entries are the pairs.
	Entries [][2]any `json:"entries"`
}

// recordLiteral is the literal of a record, as pairs of a field's name and
// its value's literal.
type recordLiteral struct {
	// Type is typeRecord.
	Type string `json:"type"`
	// Fields are the pairs, in declaration order.
	Fields [][2]any `json:"fields"`
}

// bareVariant is the literal of a variant without a payload.
type bareVariant struct {
	// Type is typeVariant.
	Type string `json:"type"`
	// Name is the variant's name.
	Name string `json:"name"`
}

// payloadVariant is the literal of a variant with a payload.
type payloadVariant struct {
	// Type is typeVariant.
	Type string `json:"type"`
	// Name is the variant's name.
	Name string `json:"name"`
	// Payload is the literal of the payload.
	Payload any `json:"payload"`
}

// opaqueLiteral is the literal of a value that no other literal states.
type opaqueLiteral struct {
	// Type is typeOpaque.
	Type string `json:"type"`
	// Text is the value's text, as the sentence of a failure prints it.
	Text string `json:"text"`
}

// written is the typed literal of one value, as the walk builds it.
type written struct {
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
	// level is the levels of objects and arrays that the literals around
	// the value being walked nest.
	level int
}

// spend counts n more parts, and reports whether the walk is still within
// maxParts.
func (w *walk) spend(n int) bool {
	w.parts += n
	return w.parts <= maxParts
}

// fits reports whether n more parts can be within maxParts, so that a
// collection whose every element is at least one part is refused before
// its storage is allocated.
func (w *walk) fits(n int) bool {
	return w.parts+n <= maxParts
}

// literalOf returns the literal of v, and false when v has none, the walk
// passes maxParts, or v is below more levels than any literal nests. The
// last bound ends the walk of a value that contains itself within a few
// dozen levels.
func (w *walk) literalOf(v reflect.Value) (written, bool) {
	if !w.spend(1) || w.level > valueDepth {
		return written{}, false
	}
	kind := v.Kind()
	if !v.IsValid() || (kind == reflect.Pointer || kind == reflect.Interface) && v.IsNil() {
		return written{form: nullLiteral{Type: typeNull}, depth: 1}, true
	}
	if n, ok := bigOf(v); ok {
		return integer(n.String(), n.CmpAbs(safeBig) <= 0), true
	}
	if v.Type().Implements(textMarshaler) {
		text, err := v.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil || !utf8.Valid(text) || !w.spend(len(text)) {
			return written{}, false
		}
		return scalar(typeString, string(text)), true
	}
	if kind == reflect.Pointer || kind == reflect.Interface {
		return w.literalOf(v.Elem())
	}
	switch v.Type() {
	case recordType:
		return w.recordOf(v.Interface().(Record))
	case pairsType:
		return w.pairsOf(v.Interface().(Pairs))
	case variantType:
		return w.variantOf(v.Interface().(Variant))
	}
	if kind == reflect.Struct {
		return w.structOf(v)
	}
	if (kind == reflect.Map || kind == reflect.Slice) && v.IsNil() {
		return absentOf(v.Type()), true
	}
	if kind == reflect.Map {
		return w.mapOf(v)
	}
	if kind == reflect.Slice || kind == reflect.Array {
		return w.listOf(v)
	}
	return w.scalarOf(v)
}

// absentOf returns the literal of a nil slice or map of type t: the absent
// list of a scalar type, the absent map from strings to a scalar type, and
// null for any other, such as nil bytes, which no absent literal states.
func absentOf(t reflect.Type) written {
	of := scalarOf(t.Elem())
	if t.Kind() == reflect.Slice && of != "" && t.Elem().Kind() != reflect.Uint8 {
		return written{form: scalarList{Type: typeList, Of: of}, depth: 1}
	}
	if t.Kind() == reflect.Map && of != "" && scalarOf(t.Key()) == typeString {
		return written{form: absentMap{Type: typeMap, Key: typeString, Of: of}, depth: 1}
	}
	return written{form: nullLiteral{Type: typeNull}, depth: 1}
}

// scalarOf returns the scalar type whose literal every value of t states,
// and empty for a type whose values state another literal, or none. A nil
// pointer and a nil interface state null, so neither kind has one.
func scalarOf(t reflect.Type) string {
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface {
		return ""
	}
	if t == bigValue {
		return typeInt
	}
	if t.Implements(textMarshaler) {
		return typeString
	}
	switch t.Kind() {
	case reflect.Bool:
		return typeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return typeInt
	case reflect.Float32, reflect.Float64:
		return typeFloat
	case reflect.String:
		return typeString
	}
	return ""
}

// bigOf returns the integer of a *big.Int or a big.Int, and false for a
// value of any other type.
func bigOf(v reflect.Value) (*big.Int, bool) {
	switch v.Type() {
	case bigPointer:
		return v.Interface().(*big.Int), true
	case bigValue:
		n := v.Interface().(big.Int)
		return &n, true
	}
	return nil, false
}

// scalarOf returns the literal of a bool, an integer, a float or a string,
// and false for a value of any other kind.
func (w *walk) scalarOf(v reflect.Value) (written, bool) {
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
	return written{}, false
}

// listOf returns the literal of a slice or an array: a byte string for
// bytes, and a list otherwise. A list of scalars of one type states their
// values, and any other list the literal of each element.
func (w *walk) listOf(v reflect.Value) (written, bool) {
	if v.Type().Elem().Kind() == reflect.Uint8 {
		if !w.spend(v.Len()) {
			return written{}, false
		}
		b := make([]byte, v.Len())
		for i := range b {
			b[i] = byte(v.Index(i).Uint())
		}
		return written{form: scalarLiteral{Type: typeBytes, Value: hex.EncodeToString(b)}, depth: 1}, true
	}
	if !w.fits(v.Len()) {
		return written{}, false
	}
	items := make([]written, v.Len())
	for i := range items {
		item, ok := w.nested(v.Index(i), 2)
		if !ok {
			return written{}, false
		}
		items[i] = item
	}
	if of := commonScalar(items); of != "" {
		values := make([]any, len(items))
		for i, item := range items {
			values[i] = item.plain
		}
		return written{form: scalarList{Type: typeList, Of: of, Value: values}, depth: 2}, true
	}
	forms := make([]any, len(items))
	deepest := 0
	for i, item := range items {
		forms[i], deepest = item.form, max(deepest, item.depth)
	}
	return written{form: itemList{Type: typeList, Items: forms}, depth: 2 + deepest}, true
}

// mapOf returns the literal of a map: its entries, sorted by the JSON of
// their keys' literals.
func (w *walk) mapOf(v reflect.Value) (written, bool) {
	type sorted struct {
		// key is the JSON of the key's literal.
		key []byte
		// pair are the key's and the value's literals.
		pair [2]written
	}
	if !w.fits(2 * v.Len()) {
		return written{}, false
	}
	entries := make([]sorted, 0, v.Len())
	for it := v.MapRange(); it.Next(); {
		key, ok := w.nested(it.Key(), 3)
		if !ok {
			return written{}, false
		}
		value, ok := w.nested(it.Value(), 3)
		if !ok {
			return written{}, false
		}
		text, _ := json.Marshal(key.form)
		entries = append(entries, sorted{key: text, pair: [2]written{key, value}})
	}
	slices.SortFunc(entries, func(a, b sorted) int { return bytes.Compare(a.key, b.key) })
	pairs := make([][2]written, len(entries))
	for i, e := range entries {
		pairs[i] = e.pair
	}
	return mapLiteral(pairs), true
}

// pairsOf returns the literal of a map that keeps its entries' order: its
// entries, in that order.
func (w *walk) pairsOf(p Pairs) (written, bool) {
	if !w.fits(2 * len(p.Entries)) {
		return written{}, false
	}
	pairs := make([][2]written, len(p.Entries))
	for i, e := range p.Entries {
		key, ok := w.nested(reflect.ValueOf(e.Key), 3)
		if !ok {
			return written{}, false
		}
		value, ok := w.nested(reflect.ValueOf(e.Value), 3)
		if !ok {
			return written{}, false
		}
		pairs[i] = [2]written{key, value}
	}
	return mapLiteral(pairs), true
}

// structOf returns a struct's literal, which is a map of its exported fields
// in declaration order with each field's name as its key.
func (w *walk) structOf(v reflect.Value) (written, bool) {
	var pairs [][2]written
	for field, value := range v.Fields() {
		if !field.IsExported() {
			continue
		}
		lit, ok := w.nested(value, 3)
		if !ok {
			return written{}, false
		}
		pairs = append(pairs, [2]written{scalar(typeString, field.Name), lit})
	}
	return mapLiteral(pairs), true
}

// recordOf returns a record's literal: each field's name and value, in
// order. The literal nests its object, the list of fields, and each field's
// list around its value's literal.
func (w *walk) recordOf(r Record) (written, bool) {
	if !w.fits(len(r.Fields)) {
		return written{}, false
	}
	forms := make([][2]any, len(r.Fields))
	depth := 2
	for i, f := range r.Fields {
		if !w.spend(len(f.Name)) {
			return written{}, false
		}
		value, ok := w.nested(reflect.ValueOf(f.Value), 3)
		if !ok {
			return written{}, false
		}
		forms[i] = [2]any{f.Name, value.form}
		depth = max(depth, 3+value.depth)
	}
	return written{form: recordLiteral{Type: typeRecord, Fields: forms}, depth: depth}, true
}

// variantOf returns a variant's literal: its name, and its payload's literal
// when it has one.
func (w *walk) variantOf(v Variant) (written, bool) {
	if !w.spend(len(v.Name)) {
		return written{}, false
	}
	if !v.HasPayload {
		return written{form: bareVariant{Type: typeVariant, Name: v.Name}, depth: 1}, true
	}
	payload, ok := w.nested(reflect.ValueOf(v.Payload), 1)
	if !ok {
		return written{}, false
	}
	form := payloadVariant{Type: typeVariant, Name: v.Name, Payload: payload.form}
	return written{form: form, depth: 1 + payload.depth}, true
}

// nested returns the literal of v, a part of a literal that nests v levels
// deeper than the literal itself.
func (w *walk) nested(v reflect.Value, levels int) (written, bool) {
	w.level += levels
	lit, ok := w.literalOf(v)
	w.level -= levels
	return lit, ok
}

// Encode returns the typed literal of v, and false when no typed literal
// states v.
//
// A value maps to a literal by its kind:
//
//   - nil and a nil pointer are null.
//   - A *big.Int and a big.Int are an int.
//   - A value whose type implements [encoding.TextMarshaler] is the string
//     of its text.
//   - A pointer is its target, and an interface its dynamic value.
//   - A [Record] is a record, a [Pairs] a map of its entries in their order,
//     and a [Variant] a variant.
//   - A bool, an integer, a float and a string are scalars. An integer
//     beyond 2^53 - 1 in magnitude is a decimal string, and a NaN or an
//     infinity is its name.
//   - A slice or an array of bytes is a byte string, and any other slice or
//     array a list.
//   - A map is its entries, sorted by the JSON of their keys, so that one
//     map has one literal.
//   - Any other struct is a map of its exported fields in declaration order.
//
// A string that is not UTF-8, a complex number, a channel, a function and an
// unsafe pointer have no literal, and neither has a value that contains one.
// Nor has a value whose literal would nest more than 61 levels, or that has
// more than 65,536 parts, each value one part and each byte of a string, a
// byte string or a field's name one more. A value that contains itself has
// more.
func Encode(v any) (json.RawMessage, bool) {
	var w walk
	lit, ok := w.literalOf(reflect.ValueOf(v))
	if !ok || lit.depth > valueDepth {
		return nil, false
	}
	raw, err := json.Marshal(lit.form)
	return raw, err == nil
}

// Plain returns the JSON value that the typed literal of v states for a
// bool, an integer, a float, a string and an integer of any size: a number,
// a decimal string for an integer beyond 2^53 - 1 in magnitude, and the
// name NaN, Inf or -Inf for a float that JSON has no number for. It reports
// false for a value of any other kind.
//
// # Allocation contract
//
// Plain allocates the literal that it builds the value from: four
// allocations for an int.
func Plain(v any) (any, bool) {
	var w walk
	lit, ok := w.literalOf(reflect.ValueOf(v))
	if !ok || lit.scalar == "" {
		return nil, false
	}
	return lit.plain, true
}

// Opaque returns the literal of a value that no other typed literal states,
// with the value's text. A call record states such a value, and a corpus
// case never does, so [Decode] refuses it.
//
// # Allocation contract
//
// Opaque allocates the JSON and encoding/json's buffer for it.
func Opaque(text string) json.RawMessage {
	// Marshal returns no error for a struct of two strings, and writes the
	// replacement character for each byte that is not UTF-8.
	raw, _ := json.Marshal(opaqueLiteral{Type: typeOpaque, Text: text})
	return raw
}

// Detail returns the literal of v as the detail of a call record states it:
// the literal that [Encode] returns, or the [Opaque] literal of the text
// that fmt's %+v verb writes for v, bounded as the package text bounds it.
// An error and a context.Context are opaque, as is a value that Encode
// states no literal of.
//
// # Allocation contract
//
// Detail allocates what Encode allocates for a value that it states, and
// the text and its literal for an opaque value.
func Detail(v any) json.RawMessage {
	switch v.(type) {
	case error, context.Context:
		return Opaque(text.Sprintf("%+v", v))
	}
	if raw, ok := Encode(v); ok {
		return raw
	}
	return Opaque(text.Sprintf("%+v", v))
}

// commonScalar returns the scalar type that every one of items has, and
// empty when items is empty or contains a literal of another type.
func commonScalar(items []written) string {
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
func mapLiteral(pairs [][2]written) written {
	forms := make([][2]any, len(pairs))
	depth := 2
	for i, pair := range pairs {
		forms[i] = [2]any{pair[0].form, pair[1].form}
		depth = max(depth, 3+pair[0].depth, 3+pair[1].depth)
	}
	return written{form: entryMap{Type: typeMap, Entries: forms}, depth: depth}
}

// scalar returns the literal of a scalar of type typ with the JSON value
// plain.
func scalar(typ string, plain any) written {
	return written{form: scalarLiteral{Type: typ, Value: plain}, scalar: typ, plain: plain, depth: 1}
}

// integer returns the literal of an integer with the decimal text: a JSON
// number when safe, and the text as a string beyond 2^53 - 1.
func integer(text string, safe bool) written {
	if safe {
		return scalar(typeInt, json.Number(text))
	}
	return scalar(typeInt, text)
}

// float returns the literal of x, with the name of a NaN or an infinity.
func float(x float64) written {
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
