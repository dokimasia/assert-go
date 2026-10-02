// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// canonical returns the one text of v under the definition's equality. Two
// values of the same type and the same value have the same text, and any
// other two values have different texts, under these rules:
//
//   - Every integer type is the type int, and every float type is the type
//     float, as in a typed literal.
//   - A float compares by its bits. -0 differs from +0, and every NaN is
//     one value.
//   - A map's text lists its entries in the order of their keys' texts. Two
//     maps of equal entries have one text.
//   - A pointer and an interface have the text of their target. nil has the
//     text null.
//   - A value of another kind, such as a channel, has the empty text. No
//     generator of the vocabulary decodes one.
func canonical(v any) string {
	var b strings.Builder
	writeCanonical(&b, reflect.ValueOf(v))
	return b.String()
}

// writeCanonical writes the canonical text of v to b.
func writeCanonical(b *strings.Builder, v reflect.Value) {
	switch v.Kind() {
	case reflect.Invalid:
		b.WriteString(typeNull)
	case reflect.Bool:
		fmt.Fprintf(b, "%s:%t", typeBool, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(b, "%s:%d", typeInt, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		fmt.Fprintf(b, "%s:%d", typeInt, v.Uint())
	case reflect.Float32, reflect.Float64:
		fmt.Fprintf(b, "%s:%s", typeFloat, strconv.FormatFloat(v.Float(), 'g', -1, 64))
	case reflect.String:
		fmt.Fprintf(b, "%s:%q", typeString, v.String())
	case reflect.Slice:
		writeSlice(b, v)
	case reflect.Map:
		writeMap(b, v)
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			b.WriteString(typeNull)
			return
		}
		writeCanonical(b, v.Elem())
	}
}

// writeSlice writes a byte string as bytes in hexadecimal, and any other
// slice as the list of its elements' texts.
func writeSlice(b *strings.Builder, v reflect.Value) {
	if v.Type().Elem().Kind() == reflect.Uint8 {
		fmt.Fprintf(b, "%s:%s", typeBytes, hex.EncodeToString(v.Bytes()))
		return
	}
	b.WriteString(typeList + ":[")
	for i := range v.Len() {
		if i > 0 {
			b.WriteByte(',')
		}
		writeCanonical(b, v.Index(i))
	}
	b.WriteByte(']')
}

// writeMap writes the entries of a map, sorted by the text of each key.
func writeMap(b *strings.Builder, v reflect.Value) {
	entries := make([]string, 0, v.Len())
	for key, value := range v.Seq2() {
		entries = append(entries, canonical(key.Interface())+"="+canonical(value.Interface()))
	}
	slices.Sort(entries)
	fmt.Fprintf(b, "%s:{%s}", typeMap, strings.Join(entries, ","))
}
