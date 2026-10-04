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

	"go.dokimi.dev/assert/internal/cycle"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// The tags that start the canonical encoding of each value.
const (
	// invalidTag starts the encoding of no value.
	invalidTag byte = iota
	// valueTag starts the encoding of a value: its type, then its value.
	valueTag
	// cycleTag starts the encoding of a pointer, a map or a slice inside
	// itself: the number of steps back to it.
	cycleTag
)

// canonicalKey returns a key that is equal for two values exactly when the
// definition counts them equal: the same type and the same value. Floats
// compare by their bits, so -0 differs from +0 and every NaN is one value.
// A nil slice equals an empty one, a map compares by its entries, and a
// pointer by what it points at. A pointer, a map or a slice inside itself
// is written as the number of steps back to it, so the key of a value that
// contains itself is finite, and two such values have one key when they
// have the same structure and the same cycles.
func canonicalKey(v any) string {
	var k keyWriter
	return string(k.append(nil, reflect.ValueOf(v)))
}

// keyWriter writes canonical keys. path are the pointers, maps and slices
// that contain the value being written.
type keyWriter struct {
	path cycle.Path
}

// append appends the canonical encoding of v to b.
func (k *keyWriter) append(b []byte, v reflect.Value) []byte {
	if !v.IsValid() {
		return append(b, invalidTag)
	}
	if !enclosing(v) {
		return k.appendValue(b, v)
	}
	back, entered := k.path.Enter(v)
	if !entered {
		return binary.AppendUvarint(append(b, cycleTag), uint64(back))
	}
	b = k.appendValue(b, v)
	k.path.Leave()
	return b
}

// enclosing reports whether v is a pointer, a map or a slice that is not
// nil, which a walk can meet again inside itself.
func enclosing(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		return !v.IsNil()
	}
	return false
}

// appendValue appends the canonical encoding of v, a valid value: its tag,
// its type, then its value.
func (k *keyWriter) appendValue(b []byte, v reflect.Value) []byte {
	b = appendText(append(b, valueTag), v.Type().PkgPath()+"."+v.Type().String())
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
	return k.appendComposite(b, v)
}

// appendComposite appends the canonical value of a string, a slice, an
// array, a map, a struct, a pointer, an interface, a channel, a function
// or an unsafe pointer.
func (k *keyWriter) appendComposite(b []byte, v reflect.Value) []byte {
	if v.Kind() == reflect.String {
		return appendText(b, v.String())
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		b = binary.AppendUvarint(b, uint64(v.Len()))
		for i := range v.Len() {
			b = k.append(b, v.Index(i))
		}
		return b
	}
	if v.Kind() == reflect.Map {
		entries := make([][]byte, 0, v.Len())
		for it := v.MapRange(); it.Next(); {
			entries = append(entries, k.append(k.append(nil, it.Key()), it.Value()))
		}
		slices.SortFunc(entries, bytes.Compare)
		b = binary.AppendUvarint(b, uint64(len(entries)))
		for _, e := range entries {
			b = appendText(b, string(e))
		}
		return b
	}
	if v.Kind() == reflect.Struct {
		for i := range v.NumField() { //nolint:modernize // Value.Fields allocates on each call, and this loop does not
			b = k.append(b, v.Field(i))
		}
		return b
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return append(b, 0)
		}
		return k.append(append(b, 1), v.Elem())
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
