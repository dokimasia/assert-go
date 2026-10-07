// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Canonical returns the one text of v under the definition's equality. Two
// values that one typed literal states have the same text, and any other
// two values have different texts, under these rules:
//
//   - Every integer type is the type int, a big.Int included, and every
//     float type is the type float, as in a typed literal.
//   - A float compares by its bits. -0 differs from +0, and every NaN is
//     one value.
//   - A map's text lists its entries in the order of their keys' texts, so
//     two maps of equal entries have one text. A [Pairs] has the text of a
//     map of its entries.
//   - A [Record] lists its fields in order, each with its name. A [Variant]
//     states its name, and its payload when it has one.
//   - A pointer and an interface have the text of their target. nil has the
//     text null.
//   - A value of another kind, such as a channel, or a struct other than
//     the three, has the empty text.
func Canonical(v any) string {
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
		entries := make([]string, 0, v.Len())
		for key, value := range v.Seq2() {
			entries = append(entries, Canonical(key.Interface())+"="+Canonical(value.Interface()))
		}
		writeEntries(b, entries)
	case reflect.Struct:
		writeStruct(b, v)
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
	if IsBytes(v.Type()) {
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

// writeEntries writes the texts of a map's entries, each a key's text and
// a value's text, sorted.
func writeEntries(b *strings.Builder, entries []string) {
	slices.Sort(entries)
	fmt.Fprintf(b, "%s:{%s}", typeMap, strings.Join(entries, ","))
}

// writeStruct writes the text of a big.Int, a [Record], a [Pairs] or a
// [Variant], and nothing for any other struct.
func writeStruct(b *strings.Builder, v reflect.Value) {
	switch v.Type() {
	case bigValue:
		n := v.Interface().(big.Int)
		fmt.Fprintf(b, "%s:%s", typeInt, n.String())
	case recordType:
		b.WriteString(typeRecord + ":[")
		for i, f := range v.Interface().(Record).Fields {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(b, "%q=%s", f.Name, Canonical(f.Value))
		}
		b.WriteByte(']')
	case pairsType:
		pairs := v.Interface().(Pairs).Entries
		entries := make([]string, len(pairs))
		for i, e := range pairs {
			entries[i] = Canonical(e.Key) + "=" + Canonical(e.Value)
		}
		writeEntries(b, entries)
	case variantType:
		variant := v.Interface().(Variant)
		fmt.Fprintf(b, "%s:%q", typeVariant, variant.Name)
		if variant.HasPayload {
			fmt.Fprintf(b, "(%s)", Canonical(variant.Payload))
		}
	}
}
