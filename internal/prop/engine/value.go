// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"slices"
	"strconv"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// canonicalKey returns a key that is equal for two values exactly when the
// definition counts them equal: the same type and the same value. Floats
// compare by their bits, so -0 differs from +0 and every NaN is one value.
// A nil slice equals an empty one, a map compares by its entries, and a
// pointer by what it points at.
func canonicalKey(v any) string {
	return string(appendCanonical(nil, reflect.ValueOf(v)))
}

// appendCanonical appends the canonical encoding of v to b: its type, then
// its value.
func appendCanonical(b []byte, v reflect.Value) []byte {
	if !v.IsValid() {
		return append(b, 0)
	}
	b = appendText(b, v.Type().PkgPath()+"."+v.Type().String())
	if v.Kind() == reflect.Bool {
		return strconv.AppendBool(b, v.Bool())
	}
	if v.CanInt() {
		return binary.AppendVarint(b, v.Int())
	}
	if v.CanUint() {
		return binary.AppendUvarint(b, v.Uint())
	}
	if v.CanFloat() {
		return binary.LittleEndian.AppendUint64(b, floatBits(v.Float()))
	}
	if v.CanComplex() {
		b = binary.LittleEndian.AppendUint64(b, floatBits(real(v.Complex())))
		return binary.LittleEndian.AppendUint64(b, floatBits(imag(v.Complex())))
	}
	return appendComposite(b, v)
}

// appendComposite appends the canonical value of a string, a slice, an
// array, a map, a struct, a pointer, an interface, a channel, a function
// or an unsafe pointer.
func appendComposite(b []byte, v reflect.Value) []byte {
	if v.Kind() == reflect.String {
		return appendText(b, v.String())
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		b = binary.AppendUvarint(b, uint64(v.Len()))
		for i := range v.Len() {
			b = appendCanonical(b, v.Index(i))
		}
		return b
	}
	if v.Kind() == reflect.Map {
		entries := make([][]byte, 0, v.Len())
		for it := v.MapRange(); it.Next(); {
			entries = append(entries, appendCanonical(appendCanonical(nil, it.Key()), it.Value()))
		}
		slices.SortFunc(entries, bytes.Compare)
		b = binary.AppendUvarint(b, uint64(len(entries)))
		for _, e := range entries {
			b = appendText(b, string(e))
		}
		return b
	}
	if v.Kind() == reflect.Struct {
		for _, field := range v.Fields() {
			b = appendCanonical(b, field)
		}
		return b
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return append(b, 0)
		}
		return appendCanonical(append(b, 1), v.Elem())
	}
	return binary.AppendUvarint(b, uint64(v.Pointer()))
}

// appendText appends s with its length before it, so no two sequences of
// texts encode alike.
func appendText(b []byte, s string) []byte {
	return append(binary.AppendUvarint(b, uint64(len(s))), s...)
}

// floatBits returns the bits of x, with one value for every NaN.
func floatBits(x float64) uint64 {
	if math.IsNaN(x) {
		return choice.NaNBits
	}
	return math.Float64bits(x)
}
