// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// The typed-literal type tags.
const (
	typeNull   = "null"
	typeBool   = "bool"
	typeInt    = "int"
	typeFloat  = "float"
	typeString = "string"
	typeList   = "list"
	typeMap    = "map"
)

// The float literals JSON has no number for.
const (
	floatNaN    = "NaN"
	floatInf    = "Inf"
	floatNegInf = "-Inf"
)

// ErrUnknownType reports a typed literal this decoder does not
// implement.
var ErrUnknownType = errors.New("conformance: unknown typed-literal type")

// literal is one encoded value from a corpus case.
type literal struct {
	Type  string          `json:"type"`
	Of    string          `json:"of"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// Decode turns one typed literal into a native value.
//
// An empty list decodes to a non-nil slice, and an empty map to a
// non-nil map. That is what lets a case tell a collection that is
// absent from one that is present and empty, which is the rule the
// whole encoding exists to pin.
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
		return scalar[int](lit.Value)
	case typeFloat:
		return decodeFloat(lit.Value)
	case typeString:
		return scalar[string](lit.Value)
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

// decodeList materializes a list of the element type named by Of.
func decodeList(lit literal) (any, error) {
	switch lit.Of {
	case typeBool:
		return typedList[bool](lit.Value)
	case typeInt:
		return typedList[int](lit.Value)
	case typeFloat:
		return typedList[float64](lit.Value)
	case typeString:
		return typedList[string](lit.Value)
	default:
		return nil, fmt.Errorf("%w: list of %q", ErrUnknownType, lit.Of)
	}
}

// typedList decodes into a non-nil slice of T.
func typedList[T any](raw json.RawMessage) (any, error) {
	out := []T{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("conformance: decode list: %w", err)
	}
	return out, nil
}

// decodeMap materializes a string-keyed map of the type named by Of.
func decodeMap(lit literal) (any, error) {
	if lit.Key != typeString {
		return nil, fmt.Errorf("%w: map keyed by %q", ErrUnknownType, lit.Key)
	}

	switch lit.Of {
	case typeBool:
		return typedMap[bool](lit.Value)
	case typeInt:
		return typedMap[int](lit.Value)
	case typeFloat:
		return typedMap[float64](lit.Value)
	case typeString:
		return typedMap[string](lit.Value)
	default:
		return nil, fmt.Errorf("%w: map of %q", ErrUnknownType, lit.Of)
	}
}

// typedMap decodes into a non-nil map of T.
func typedMap[T any](raw json.RawMessage) (any, error) {
	out := map[string]T{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("conformance: decode map: %w", err)
	}
	return out, nil
}
