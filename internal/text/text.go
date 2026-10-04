// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package text

import (
	"cmp"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// maxParts is the most values that the walk of one argument visits. It
// bounds the text of a large value.
const maxParts = 65536

// The marks of the structural text.
const (
	// cycleMark stands for a map or a slice inside itself.
	cycleMark = "<cycle>"
	// cutMark stands for the values past maxParts.
	cutMark = "…"
	// nilMark is the text of nil, as fmt writes it.
	nilMark = "<nil>"
)

// Sprintf returns what fmt.Sprintf returns for format and args, for every
// argument whose walk ends within 65,536 values without meeting a map or a
// slice inside itself.
//
// fmt writes each other argument as its structural text, whatever its verb:
// the layout of fmt's %+v verb, with <cycle> where the walk meets a map or a
// slice inside itself, and … in place of the values past the 65,536th. The
// structural text calls no method of the value.
//
// # Allocation contract
//
// Sprintf allocates what fmt.Sprintf allocates for an argument whose walk
// ends. The walk allocates nothing for a value whose maps and slices hold
// no map, slice or interface.
func Sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, walled(args)...)
}

// Fprintf writes to b what [Sprintf] returns for format and args.
//
// # Allocation contract
//
// Fprintf allocates what fmt.Fprintf allocates for an argument whose walk
// ends, and what Sprintf allocates for any other argument.
func Fprintf(b *strings.Builder, format string, args ...any) {
	_, _ = fmt.Fprintf(b, format, walled(args)...)
}

// walled returns args when every argument's walk ends within maxParts
// values without meeting a map or a slice inside itself, and otherwise a
// copy in which each other argument is its structural text.
func walled(args []any) []any {
	written, cloned := args, false
	for i, arg := range args {
		if bounded(arg) {
			continue
		}
		if !cloned {
			written, cloned = slices.Clone(args), true
		}
		written[i] = structural(arg)
	}
	return written
}

// structured is the structural text of an argument, which fmt writes as it
// is whatever the verb.
type structured string

// Format writes the text.
func (s structured) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, string(s))
}

// bounded reports whether the walk of v ends within maxParts values without
// meeting a map or a slice inside itself. fmt walks no more of v than the
// walk does, because it stops where a value has a method that writes it.
func bounded(v any) bool {
	switch v.(type) {
	case nil, bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64, complex64, complex128:
		return true
	}
	var w walk
	w.visit(reflect.ValueOf(v), 0)
	return !w.cycled && w.parts <= maxParts
}

// structural returns the structural text of v.
func structural(v any) structured {
	var b strings.Builder
	w := walk{out: &b}
	w.visit(reflect.ValueOf(v), 0)
	return structured(b.String())
}

// walk is one walk of a value's parts, in the order fmt visits them.
type walk struct {
	// parts counts the values visited.
	parts int
	// inside are the maps and slices that contain the value being visited.
	inside []reference
	// cycled reports that the walk met a map or a slice inside itself.
	cycled bool
	// out receives the structural text, and is nil for a walk that checks
	// the value alone and stops at its first cycle or past maxParts.
	out *strings.Builder
}

// reference is one map or slice: its type, where its data is, and for a
// slice its length.
type reference struct {
	typ  reflect.Type
	data uintptr
	len  int
}

// halted reports whether the walk visits nothing more: it passed maxParts,
// or it checks the value alone and met a cycle.
func (w *walk) halted() bool {
	return w.parts > maxParts || (w.out == nil && w.cycled)
}

// write writes s to the text of a walk that writes.
func (w *walk) write(s string) {
	if w.out != nil {
		w.out.WriteString(s)
	}
}

// visit walks v at depth, the number of values that contain it, as fmt
// walks it. v is valid, because a walk starts at an argument other than nil.
func (w *walk) visit(v reflect.Value, depth int) {
	if w.halted() {
		return
	}
	w.parts++
	if w.parts > maxParts {
		w.write(cutMark)
		return
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			w.write(nilMark)
			return
		}
		w.visit(v.Elem(), depth+1)
	case reflect.Pointer:
		w.pointer(v, depth)
	case reflect.Struct:
		w.structure(v, depth)
	case reflect.Map:
		w.mapping(v, depth)
	case reflect.Array, reflect.Slice:
		w.list(v, depth)
	default:
		w.scalar(v)
	}
}

// pointer walks the target of a pointer at the top level to an array, a
// slice, a struct or a map, after an ampersand. It writes any other pointer
// as its address, and a nil pointer as nil.
func (w *walk) pointer(v reflect.Value, depth int) {
	if v.IsNil() {
		w.write(nilMark)
		return
	}
	if depth == 0 {
		switch v.Elem().Kind() {
		case reflect.Array, reflect.Slice, reflect.Struct, reflect.Map:
			w.write("&")
			w.visit(v.Elem(), depth+1)
			return
		}
	}
	w.write("0x" + strconv.FormatUint(uint64(v.Pointer()), 16))
}

// structure walks the fields of a struct, each with its name.
func (w *walk) structure(v reflect.Value, depth int) {
	w.write("{")
	for i := range v.NumField() {
		if w.halted() {
			break
		}
		if i > 0 {
			w.write(" ")
		}
		w.write(v.Type().Field(i).Name + ":")
		w.visit(v.Field(i), depth+1)
	}
	w.write("}")
}

// mapping walks the entries of a map. A walk that checks counts the
// entries of a map whose keys and values hold no map, slice or interface
// without visiting them. A walk that writes sorts the entries by the text
// of their keys.
func (w *walk) mapping(v reflect.Value, depth int) {
	if w.out == nil {
		keys, keysFlat := flatParts(v.Type().Key())
		values, valuesFlat := flatParts(v.Type().Elem())
		if keysFlat && valuesFlat {
			w.parts += v.Len() * (keys + values)
			return
		}
	}
	if !w.enter(reference{typ: v.Type(), data: v.Pointer()}) {
		return
	}
	defer w.leave()
	if w.out == nil {
		for it := v.MapRange(); it.Next() && !w.halted(); {
			w.visit(it.Key(), depth+1)
			w.visit(it.Value(), depth+1)
		}
		return
	}
	type entry struct{ key, value string }
	var entries []entry
	out := w.out
	for it := v.MapRange(); it.Next() && !w.halted(); {
		var k, e strings.Builder
		w.out = &k
		w.visit(it.Key(), depth+1)
		w.out = &e
		w.visit(it.Value(), depth+1)
		entries = append(entries, entry{k.String(), e.String()})
	}
	w.out = out
	slices.SortFunc(entries, func(a, b entry) int { return cmp.Compare(a.key, b.key) })
	w.write("map[")
	for i, e := range entries {
		if i > 0 {
			w.write(" ")
		}
		w.write(e.key + ":" + e.value)
	}
	w.write("]")
}

// list walks the elements of an array or a slice. A walk that checks
// counts the elements of a list whose elements hold no map, slice or
// interface without visiting them.
func (w *walk) list(v reflect.Value, depth int) {
	if w.out == nil {
		if parts, flat := flatParts(v.Type().Elem()); flat {
			w.parts += v.Len() * parts
			return
		}
	}
	if v.Kind() == reflect.Slice {
		if !w.enter(reference{typ: v.Type(), data: v.Pointer(), len: v.Len()}) {
			return
		}
		defer w.leave()
	}
	w.write("[")
	for i := range v.Len() {
		if w.halted() {
			break
		}
		if i > 0 {
			w.write(" ")
		}
		w.visit(v.Index(i), depth+1)
	}
	w.write("]")
}

// scalar writes a value that fmt writes without walking it: a bool, a
// number and a string as fmt's %v writes them, and a channel, a function
// and an unsafe pointer as an address.
func (w *walk) scalar(v reflect.Value) {
	if w.out == nil {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		w.write(strconv.FormatBool(v.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		w.write(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		w.write(strconv.FormatUint(v.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		w.write(strconv.FormatFloat(v.Float(), 'g', -1, v.Type().Bits()))
	case reflect.Complex64, reflect.Complex128:
		w.write(strconv.FormatComplex(v.Complex(), 'g', -1, v.Type().Bits()))
	case reflect.String:
		w.write(v.String())
	default:
		if v.IsNil() {
			w.write(nilMark)
			return
		}
		w.write("0x" + strconv.FormatUint(uint64(v.Pointer()), 16))
	}
}

// enter adds a map or a slice to those that contain the walk, and reports
// whether it was not among them already. A walk that meets one inside
// itself marks the cycle and does not walk it again.
func (w *walk) enter(r reference) bool {
	if slices.Contains(w.inside, r) {
		w.cycled = true
		w.write(cycleMark)
		return false
	}
	w.inside = append(w.inside, r)
	return true
}

// leave removes the map or the slice that the walk entered last.
func (w *walk) leave() {
	w.inside = w.inside[:len(w.inside)-1]
}

// flatParts returns the parts that the walk visits in a value of type t
// below the top level, and false when a value of t can hold a map, a slice
// or an interface, whose parts its type does not fix. A pointer there is
// one part, its address.
func flatParts(t reflect.Type) (int, bool) {
	switch t.Kind() {
	case reflect.Map, reflect.Slice, reflect.Interface:
		return 0, false
	case reflect.Array:
		parts, flat := flatParts(t.Elem())
		return 1 + t.Len()*parts, flat
	case reflect.Struct:
		total := 1
		for i := range t.NumField() { //nolint:modernize // Type.Fields allocates on each call, and this loop does not
			parts, flat := flatParts(t.Field(i).Type)
			if !flat {
				return 0, false
			}
			total += parts
		}
		return total, true
	}
	return 1, true
}
