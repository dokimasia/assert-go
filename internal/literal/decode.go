// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"go.dokimi.dev/assert/internal/fault"
)

// The typed-literal type tags.
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
	// typeRecord is a record.
	typeRecord = "record"
	// typeVariant is a variant of an enum.
	typeVariant = "variant"
	// typeOpaque is a value that no other literal states, which a call
	// record states and a corpus case never does. Decode refuses it.
	typeOpaque = "opaque"
)

// The members of a typed literal's JSON object, which name the first
// segment of the path of a fault in the literal.
const (
	// memberType is the type tag.
	memberType = "type"
	// memberOf is the element type of a list or the value type of a map.
	memberOf = "of"
	// memberKey is the key type of a map.
	memberKey = "key"
	// memberValue is a scalar's value, or the values of a list or a map.
	memberValue = "value"
	// memberItems is the literals of a list.
	memberItems = "items"
	// memberEntries is the key and value literals of a map.
	memberEntries = "entries"
	// memberFields is the name and value literals of a record.
	memberFields = "fields"
	// memberName is a variant's name.
	memberName = "name"
	// memberPayload is a variant's payload.
	memberPayload = "payload"
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

// safeInteger is 2^53 - 1, the largest integer magnitude that a JSON number
// states exactly in every target language. A larger one is a decimal string,
// because a JavaScript reader would round the number.
const safeInteger = 9007199254740991

// pair is the length of a record's field and of a map's entry: a name or a
// key, and a value.
const pair = 2

// ErrUnknownType reports a typed literal of a type that the encoding does
// not define.
var ErrUnknownType = errors.New("literal: unknown typed-literal type")

// typed is one typed literal, as its JSON states it.
type typed struct {
	// Type is the literal's type tag.
	Type string `json:"type"`
	// Of is the element type of a list, or the value type of a map of
	// string keys.
	Of string `json:"of"`
	// Key is the key type of a map that states its value as an object.
	Key string `json:"key"`
	// Value is a scalar's value, or the elements or the entries of a list or
	// a map of one scalar type.
	Value json.RawMessage `json:"value"`
	// Items are the literals of a list whose elements share no scalar type,
	// and nil for a list of one.
	Items []json.RawMessage `json:"items"`
	// Entries are the key and value literals of a map in the order the
	// literal states them, and nil for a map of string keys.
	Entries [][]json.RawMessage `json:"entries"`
	// Fields are the name and value literal of each field of a record.
	Fields []json.RawMessage `json:"fields"`
	// Name is a variant's name.
	Name string `json:"name"`
	// Payload is a variant's payload, and nil for a variant without one.
	Payload json.RawMessage `json:"payload"`
}

// Decode turns one typed literal into a Go value.
//
// An empty list decodes to a non-nil slice, and an empty map to a non-nil
// map. A list or a map whose value is null decodes to a nil slice or a nil
// map of the stated type. A case can therefore tell a collection that is
// absent from one that is present and empty, as the encoding requires.
//
// An int is an int, and beyond 2^53 - 1 in magnitude a decimal string that
// decodes to an int64, to a uint64 above the int64 range, and to a
// *big.Int beyond both ranges. A float is a number or one of the names NaN,
// Inf and -Inf. The elements of a list and the values of a map follow the
// same two rules. A list or a map of int decodes to a []int or a
// map[string]int when every value fits an int, and otherwise to a []any or
// a map[string]any of each value as an int literal decodes. Bytes are
// lowercase hexadecimal and decode to a []byte. A list of items decodes to
// a []any, and a map of entries to a map[any]any, whose keys must be
// comparable. A record decodes to a [Record], and a variant to a [Variant].
//
// # Errors
//
// Every error is a fault whose path states the members of the literal's
// JSON down to what the literal misstates, such as
// fields[2][1].items[0].value. A fault of a type that the encoding does not
// define has the kind [ErrUnknownType].
func Decode(raw json.RawMessage) (any, error) {
	var lit typed
	if err := json.Unmarshal(raw, &lit); err != nil {
		return nil, fault.New("the text is no typed literal").Because(err)
	}

	switch lit.Type {
	case typeNull:
		return nil, nil
	case typeBool:
		return atValue(decodeScalar[bool](lit.Value))
	case typeInt:
		return atValue(decodeInt(lit.Value))
	case typeFloat:
		return atValue(Float(lit.Value))
	case typeString:
		return atValue(decodeScalar[string](lit.Value))
	case typeBytes:
		return atValue(decodeBytes(lit.Value))
	case typeList:
		return decodeList(lit)
	case typeMap:
		return decodeMap(lit)
	case typeRecord:
		return decodeRecord(lit.Fields)
	case typeVariant:
		return decodeVariant(lit)
	default:
		return nil, fault.At(fault.Of(ErrUnknownType, "the type %q is no type of the encoding", lit.Type),
			fault.Field(memberType))
	}
}

// atValue returns v, or err at the literal's value.
func atValue[T any](v T, err error) (any, error) {
	if err != nil {
		return nil, fault.At(err, fault.Field(memberValue))
	}
	return v, nil
}

// decodeScalar decodes a JSON scalar into T.
func decodeScalar[T any](raw json.RawMessage) (any, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fault.New("the value is no %T", v).Because(err)
	}
	return v, nil
}

// Int returns the integer that the value of an int literal states: an int
// for a JSON integer, and for a decimal string beyond 2^53 - 1 in magnitude
// an int64, or a uint64 above the int64 range.
//
// # Errors
//
// It returns a fault for any other JSON, and for a decimal string that is
// not an integer's canonical spelling or that a JSON number states.
func Int(raw json.RawMessage) (any, error) {
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return decodeScalar[int](raw)
	}
	return largeInt(text)
}

// decodeInt returns the integer that the value of an int literal states, as
// [Int] does, and a *big.Int for the canonical decimal string of an integer
// beyond both 64-bit ranges.
func decodeInt(raw json.RawMessage) (any, error) {
	v, err := Int(raw)
	if err == nil {
		return v, nil
	}
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return nil, err
	}
	n, ok := new(big.Int).SetString(text, 10)
	if !ok || n.String() != text || n.IsInt64() || n.IsUint64() {
		return nil, err
	}
	return n, nil
}

// largeInt decodes the decimal string of an integer beyond 2^53 - 1 in
// magnitude: to an int64, or to a uint64 above the int64 range. It refuses a
// string that is not the integer's canonical spelling, and an integer that
// a JSON number states.
func largeInt(text string) (any, error) {
	if v, err := strconv.ParseInt(text, 10, 64); err == nil && strconv.FormatInt(v, 10) == text {
		if v < -safeInteger || v > safeInteger {
			return v, nil
		}
		return nil, fault.New("%s is within 2^53 - 1, which a JSON number states", text)
	}
	if v, err := strconv.ParseUint(text, 10, 64); err == nil && strconv.FormatUint(v, 10) == text {
		return v, nil
	}
	return nil, fault.New("%q is no canonical integer of 64 bits", text)
}

// Float returns the float that the value of a float literal states: a JSON
// number, or one of the names NaN, Inf and -Inf, which JSON has no number
// for.
//
// # Errors
//
// It returns a fault for any other JSON and any other name.
func Float(raw json.RawMessage) (float64, error) {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		switch name {
		case nameNaN:
			return math.NaN(), nil
		case nameInf:
			return math.Inf(1), nil
		case nameNegInf:
			return math.Inf(-1), nil
		default:
			return 0, fault.New("%q is none of the names NaN, Inf and -Inf", name)
		}
	}

	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0, fault.New("the value is no float").Because(err)
	}
	return f, nil
}

// decodeBytes accepts lowercase hexadecimal.
func decodeBytes(raw json.RawMessage) (any, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, fault.New("the value is no text").Because(err)
	}
	data, err := hex.DecodeString(text)
	if err != nil || text != strings.ToLower(text) {
		return nil, fault.New("%q is no lowercase hexadecimal", text)
	}
	return data, nil
}

// decodeList materializes a list of the literals of Items, or of the element
// type that Of names.
func decodeList(lit typed) (any, error) {
	if lit.Items != nil {
		out := make([]any, len(lit.Items))
		for i, item := range lit.Items {
			value, err := Decode(item)
			if err != nil {
				return nil, fault.At(err, fault.Field(memberItems), fault.Index(i))
			}
			out[i] = value
		}
		return out, nil
	}
	switch lit.Of {
	case typeBool:
		return atValue(typedList(lit.Value, as[bool](decodeScalar[bool])))
	case typeInt:
		return atValue(intList(lit.Value))
	case typeFloat:
		return atValue(typedList(lit.Value, Float))
	case typeString:
		return atValue(typedList(lit.Value, as[string](decodeScalar[string])))
	default:
		return nil, fault.At(fault.Of(ErrUnknownType, "the element type %q is no scalar type", lit.Of),
			fault.Field(memberOf))
	}
}

// typedList decodes a JSON array into a non-nil slice of T, each element with
// decode, and JSON null into a nil slice of T.
func typedList[T any](raw json.RawMessage, decode func(json.RawMessage) (T, error)) (any, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fault.New("the value is no list").Because(err)
	}
	if items == nil {
		return []T(nil), nil
	}

	out := make([]T, len(items))
	for i, item := range items {
		value, err := decode(item)
		if err != nil {
			return nil, fault.At(err, fault.Index(i))
		}
		out[i] = value
	}
	return out, nil
}

// as adapts the decoder of one scalar type to the element type T that the
// decoder returns.
func as[T any](decode func(json.RawMessage) (any, error)) func(json.RawMessage) (T, error) {
	return func(raw json.RawMessage) (T, error) {
		value, err := decode(raw)
		if err != nil {
			var zero T
			return zero, err
		}
		return value.(T), nil
	}
}

// intList decodes a list of int: a []int when every value fits an int, and
// otherwise a []any of each value as an int literal decodes. JSON null
// decodes to a nil []int.
func intList(raw json.RawMessage) (any, error) {
	decoded, err := typedList(raw, decodeInt)
	if err != nil {
		return nil, err
	}
	values := decoded.([]any)
	if values == nil {
		return []int(nil), nil
	}
	ints := make([]int, len(values))
	for i, v := range values {
		n, ok := intOf(v)
		if !ok {
			return values, nil
		}
		ints[i] = n
	}
	return ints, nil
}

// intMap decodes a map of string keys to int: a map[string]int when every
// value fits an int, and otherwise a map[string]any of each value as an int
// literal decodes. JSON null decodes to a nil map[string]int.
func intMap(raw json.RawMessage) (any, error) {
	decoded, err := typedMap(raw, decodeInt)
	if err != nil {
		return nil, err
	}
	values := decoded.(map[string]any)
	if values == nil {
		return map[string]int(nil), nil
	}
	ints := make(map[string]int, len(values))
	for key, v := range values {
		n, ok := intOf(v)
		if !ok {
			return values, nil
		}
		ints[key] = n
	}
	return ints, nil
}

// intOf returns v, an integer as an int literal decodes, as an int, and
// false for one that does not fit an int.
func intOf(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), int64(int(n)) == n
	}
	return 0, false
}

// decodeMap materializes a map of the pairs of Entries, or a string-keyed
// map of the type that Of names.
func decodeMap(lit typed) (any, error) {
	if lit.Entries != nil {
		return decodeEntries(lit.Entries)
	}
	if lit.Key != typeString {
		return nil, fault.At(fault.Of(ErrUnknownType, "the key type %q is not string", lit.Key),
			fault.Field(memberKey))
	}

	switch lit.Of {
	case typeBool:
		return atValue(typedMap(lit.Value, as[bool](decodeScalar[bool])))
	case typeInt:
		return atValue(intMap(lit.Value))
	case typeFloat:
		return atValue(typedMap(lit.Value, Float))
	case typeString:
		return atValue(typedMap(lit.Value, as[string](decodeScalar[string])))
	default:
		return nil, fault.At(fault.Of(ErrUnknownType, "the value type %q is no scalar type", lit.Of),
			fault.Field(memberOf))
	}
}

// decodeEntries decodes the key and the value literal of each entry into a
// non-nil map[any]any. It refuses an entry that is no pair, and a key of a
// type that is not comparable.
func decodeEntries(entries [][]json.RawMessage) (any, error) {
	out := make(map[any]any, len(entries))
	for i, entry := range entries {
		if len(entry) != pair {
			return nil, fault.At(fault.New("the entry states %d literals, not a key and a value", len(entry)),
				fault.Field(memberEntries), fault.Index(i))
		}
		key, err := Decode(entry[0])
		if err != nil {
			return nil, fault.At(err, fault.Field(memberEntries), fault.Index(i), fault.Index(0))
		}
		if key != nil && !reflect.TypeOf(key).Comparable() {
			return nil, fault.At(fault.New("a key of %T is no key of a Go map", key),
				fault.Field(memberEntries), fault.Index(i), fault.Index(0))
		}
		value, err := Decode(entry[1])
		if err != nil {
			return nil, fault.At(err, fault.Field(memberEntries), fault.Index(i), fault.Index(1))
		}
		out[key] = value
	}
	return out, nil
}

// typedMap decodes a JSON object into a non-nil map of T, each value with
// decode, and JSON null into a nil map of T.
func typedMap[T any](raw json.RawMessage, decode func(json.RawMessage) (T, error)) (any, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fault.New("the value is no object").Because(err)
	}
	if values == nil {
		return map[string]T(nil), nil
	}

	out := make(map[string]T, len(values))
	for key, item := range values {
		value, err := decode(item)
		if err != nil {
			return nil, fault.At(err, fault.Key(key))
		}
		out[key] = value
	}
	return out, nil
}

// decodeRecord decodes each field of a record, a name and a value literal,
// into a [Record]. It refuses a record that states no list of fields, a
// field that is no pair of a name and a literal, an empty name, and a name
// stated twice.
func decodeRecord(fields []json.RawMessage) (any, error) {
	if fields == nil {
		return nil, fault.At(fault.New("the record states no fields"), fault.Field(memberFields))
	}
	out := Record{Fields: make([]Field, len(fields))}
	seen := make(map[string]bool, len(fields))
	for i, raw := range fields {
		var parts []json.RawMessage
		var name string
		if json.Unmarshal(raw, &parts) != nil || len(parts) != pair || json.Unmarshal(parts[0], &name) != nil {
			return nil, fault.At(fault.New("the field is no pair of a name and a literal"),
				fault.Field(memberFields), fault.Index(i))
		}
		if name == "" || seen[name] {
			return nil, fault.At(fault.New("the name %q is empty or named twice", name),
				fault.Field(memberFields), fault.Index(i), fault.Index(0))
		}
		seen[name] = true
		value, err := Decode(parts[1])
		if err != nil {
			return nil, fault.At(err, fault.Field(memberFields), fault.Index(i), fault.Index(1))
		}
		out.Fields[i] = Field{Name: name, Value: value}
	}
	return out, nil
}

// decodeVariant decodes a variant's name, and its payload when it states
// one, into a [Variant]. It refuses an empty name.
func decodeVariant(lit typed) (any, error) {
	if lit.Name == "" {
		return nil, fault.At(fault.New("the variant states no name"), fault.Field(memberName))
	}
	if lit.Payload == nil {
		return Variant{Name: lit.Name}, nil
	}
	payload, err := Decode(lit.Payload)
	if err != nil {
		return nil, fault.At(err, fault.Field(memberPayload))
	}
	return Variant{Name: lit.Name, Payload: payload, HasPayload: true}, nil
}
