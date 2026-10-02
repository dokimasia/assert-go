// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// The typed-literal type tags.
const (
	typeNull   = "null"
	typeBool   = "bool"
	typeInt    = "int"
	typeFloat  = "float"
	typeString = "string"
	typeBytes  = "bytes"
	typeList   = "list"
	typeMap    = "map"
)

// The float literals JSON has no number for.
const (
	floatNaN    = "NaN"
	floatInf    = "Inf"
	floatNegInf = "-Inf"
)

// safeInteger is the largest magnitude of an integer that a typed literal
// states as a JSON number, 2^53 - 1. A larger one is a decimal string,
// because a JavaScript reader would round the number.
const safeInteger = 9007199254740991

// ErrUnknownType reports a typed literal this decoder does not
// implement.
var ErrUnknownType = errors.New("conformance: unknown typed-literal type")

// literal is one encoded value from a corpus case.
type literal struct {
	Type  string          `json:"type"`
	Of    string          `json:"of"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	// Items are the literals of a list whose elements share no scalar
	// type, and nil for a list of one.
	Items []json.RawMessage `json:"items"`
	// Entries are the key and value literals of a map in the order the
	// corpus states them, and nil for a map of string keys.
	Entries [][2]json.RawMessage `json:"entries"`
}

// Decode turns one typed literal into a native value.
//
// An empty list decodes to a non-nil slice, and an empty map to a
// non-nil map. A list or a map whose value is null decodes to a nil
// slice or a nil map of the stated type. That is what lets a case tell a
// collection that is absent from one that is present and empty, which is
// the rule the whole encoding exists to pin.
//
// An int is an int, and beyond 2^53 - 1 in magnitude a decimal string
// that decodes to an int64, or to a uint64 above the int64 range. A
// float is a number or one of the names NaN, Inf and -Inf. The elements
// of a list and the values of a map follow the same two rules, and an
// int element decodes to an int. Bytes are lowercase hexadecimal and
// decode to a []byte. A list of items decodes to a []any, and a map of
// entries to a map[any]any, whose keys must be comparable.
func Decode(raw json.RawMessage) (any, error) {
	var lit literal
	if err := json.Unmarshal(raw, &lit); err != nil {
		return nil, fmt.Errorf("conformance: parse literal: %w", err)
	}

	switch lit.Type {
	case typeNull:
		return nil, nil
	case typeBool:
		return scalar[bool](lit.Value)
	case typeInt:
		return decodeInt(lit.Value)
	case typeFloat:
		return decodeFloat(lit.Value)
	case typeString:
		return scalar[string](lit.Value)
	case typeBytes:
		return decodeBytes(lit.Value)
	case typeList:
		return decodeList(lit)
	case typeMap:
		return decodeMap(lit)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownType, lit.Type)
	}
}

// scalar decodes a JSON scalar into T.
func scalar[T any](raw json.RawMessage) (any, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("conformance: decode scalar: %w", err)
	}
	return v, nil
}

// decodeInt accepts a JSON integer, which decodes to an int, or a decimal
// string beyond 2^53 - 1 in magnitude.
func decodeInt(raw json.RawMessage) (any, error) {
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return scalar[int](raw)
	}
	return largeInt(text)
}

// largeInt decodes the decimal string of an integer beyond 2^53 - 1 in
// magnitude: to an int64, or to a uint64 above the int64 range. It refuses
// a string that is not the integer's canonical spelling, and an integer a
// JSON number states.
func largeInt(text string) (any, error) {
	if v, err := strconv.ParseInt(text, 10, 64); err == nil && strconv.FormatInt(v, 10) == text {
		if v < -safeInteger || v > safeInteger {
			return v, nil
		}
		return nil, fmt.Errorf("conformance: %s is within 2^53 - 1, which a JSON number states", text)
	}
	if v, err := strconv.ParseUint(text, 10, 64); err == nil && strconv.FormatUint(v, 10) == text {
		return v, nil
	}
	return nil, fmt.Errorf("conformance: %q is no canonical integer of 64 bits", text)
}

// decodeFloat accepts a JSON number, or one of the named literals JSON
// has no number for.
func decodeFloat(raw json.RawMessage) (any, error) {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		switch name {
		case floatNaN:
			return math.NaN(), nil
		case floatInf:
			return math.Inf(1), nil
		case floatNegInf:
			return math.Inf(-1), nil
		default:
			return nil, fmt.Errorf("conformance: unrecognized float literal %q", name)
		}
	}

	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("conformance: decode float: %w", err)
	}
	return f, nil
}

// decodeBytes accepts lowercase hexadecimal.
func decodeBytes(raw json.RawMessage) (any, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, fmt.Errorf("conformance: decode bytes: %w", err)
	}
	data, err := hex.DecodeString(text)
	if err != nil || text != strings.ToLower(text) {
		return nil, fmt.Errorf("conformance: %q is no lowercase hexadecimal", text)
	}
	return data, nil
}

// decodeList materializes a list of the literals of Items, or of the
// element type named by Of.
func decodeList(lit literal) (any, error) {
	if lit.Items != nil {
		out := make([]any, len(lit.Items))
		for i, item := range lit.Items {
			value, err := Decode(item)
			if err != nil {
				return nil, fmt.Errorf("conformance: list item %d: %w", i, err)
			}
			out[i] = value
		}
		return out, nil
	}
	switch lit.Of {
	case typeBool:
		return typedList(lit.Value, as[bool](scalar[bool]))
	case typeInt:
		return typedList(lit.Value, intElement)
	case typeFloat:
		return typedList(lit.Value, as[float64](decodeFloat))
	case typeString:
		return typedList(lit.Value, as[string](scalar[string]))
	default:
		return nil, fmt.Errorf("%w: list of %q", ErrUnknownType, lit.Of)
	}
}

// typedList decodes a JSON array into a non-nil slice of T, each element
// with decode, and JSON null into a nil slice of T.
func typedList[T any](raw json.RawMessage, decode func(json.RawMessage) (T, error)) (any, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("conformance: decode list: %w", err)
	}
	if items == nil {
		return []T(nil), nil
	}

	out := make([]T, len(items))
	for i, item := range items {
		value, err := decode(item)
		if err != nil {
			return nil, fmt.Errorf("conformance: list element %d: %w", i, err)
		}
		out[i] = value
	}
	return out, nil
}

// as adapts the decoder of one scalar type to the element type T that
// the decoder returns.
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

// intElement decodes an element of a list or a map of int. It refuses a
// decimal string whose integer is beyond the range of int.
func intElement(raw json.RawMessage) (int, error) {
	value, err := decodeInt(raw)
	if err != nil {
		return 0, err
	}
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		if n := int(v); int64(n) == v {
			return n, nil
		}
	}
	return 0, fmt.Errorf("conformance: %v is beyond the range of int", value)
}

// decodeMap materializes a map of the pairs of Entries, or a
// string-keyed map of the type named by Of.
func decodeMap(lit literal) (any, error) {
	if lit.Entries != nil {
		return decodeEntries(lit.Entries)
	}
	if lit.Key != typeString {
		return nil, fmt.Errorf("%w: map keyed by %q", ErrUnknownType, lit.Key)
	}

	switch lit.Of {
	case typeBool:
		return typedMap(lit.Value, as[bool](scalar[bool]))
	case typeInt:
		return typedMap(lit.Value, intElement)
	case typeFloat:
		return typedMap(lit.Value, as[float64](decodeFloat))
	case typeString:
		return typedMap(lit.Value, as[string](scalar[string]))
	default:
		return nil, fmt.Errorf("%w: map of %q", ErrUnknownType, lit.Of)
	}
}

// decodeEntries decodes the key and the value literal of each entry into
// a non-nil map[any]any. It refuses a key of a type that a Go map cannot
// hold.
func decodeEntries(entries [][2]json.RawMessage) (any, error) {
	out := make(map[any]any, len(entries))
	for i, entry := range entries {
		key, err := Decode(entry[0])
		if err != nil {
			return nil, fmt.Errorf("conformance: map key %d: %w", i, err)
		}
		if key != nil && !reflect.TypeOf(key).Comparable() {
			return nil, fmt.Errorf("conformance: map key %d of %T is no key of a Go map", i, key)
		}
		value, err := Decode(entry[1])
		if err != nil {
			return nil, fmt.Errorf("conformance: map value %d: %w", i, err)
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
		return nil, fmt.Errorf("conformance: decode map: %w", err)
	}
	if values == nil {
		return map[string]T(nil), nil
	}

	out := make(map[string]T, len(values))
	for key, item := range values {
		value, err := decode(item)
		if err != nil {
			return nil, fmt.Errorf("conformance: map value %q: %w", key, err)
		}
		out[key] = value
	}
	return out, nil
}
