// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package text

import (
	"cmp"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.dokimi.dev/assert/internal/cycle"
)

// maxParts is the most values that the walk of one argument visits. It
// bounds the text of a large value.
const maxParts = 65536

// The magnitudes of the whole numbers that %v writes in decimal: fmt writes
// a float of a million or more with an exponent, and a decimal of 10^21 or
// more runs to 22 digits.
const (
	decimalFrom  = 1e6
	decimalBelow = 1e21
)

// The marks of the structural text.
const (
	// cycleMark replaces a map or a slice inside itself.
	cycleMark = "<cycle>"
	// cutMark replaces the values past maxParts.
	cutMark = "…"
	// nilMark is the text of nil, as fmt writes it.
	nilMark = "<nil>"
)

// Sprintf returns what fmt.Sprintf returns for format and args, for every
// argument whose walk ends within 65,536 values without meeting a map or a
// slice inside itself, but one: %v and %+v write a float argument whose
// value is a whole number of at least a million and below 10^21 in
// magnitude in decimal, as 4194298 where fmt writes 4.194298e+06.
//
// fmt writes each other argument as its structural text, whatever its verb:
// the layout of fmt's %+v verb, with <cycle> where the walk meets a map or a
// slice inside itself, and … in place of the values past the 65,536th. The
// structural text calls no method of the value, and writes its floats as %v
// writes a float argument.
//
// An argument of type [reflect.Value] is the value inside it, as fmt takes
// it, so a value of an unexported field is written without its methods.
//
// # Allocation contract
//
// Sprintf allocates what fmt.Sprintf allocates for an argument whose walk
// ends. The walk allocates nothing for a value whose maps and slices
// contain no map, slice or interface. A float written in decimal allocates
// a copy of the arguments, the float as an argument, its directive and its
// text.
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
// values without meeting a map or a slice inside itself and no argument is
// a float that %v writes in decimal. Otherwise it returns a copy in which
// each such float writes itself, and each other argument is its structural
// text.
func walled(args []any) []any {
	written, cloned := args, false
	for i, arg := range args {
		var replaced any
		if w, ok := wholeOf(arg); ok {
			replaced = w
		} else if !bounded(arg) {
			replaced = structural(arg)
		} else {
			continue
		}
		if !cloned {
			written, cloned = slices.Clone(args), true
		}
		written[i] = replaced
	}
	return written
}

// whole is a float argument whose value %v writes in decimal.
type whole struct {
	// arg is the float32 or the float64 argument.
	arg any
	// value is its value, and bits its width.
	value float64
	bits  int
}

// wholeOf returns the whole of a float argument whose value %v writes in
// decimal, and false for any other argument.
func wholeOf(arg any) (whole, bool) {
	w := whole{arg: arg, bits: 64}
	switch f := arg.(type) {
	case float64:
		w.value = f
	case float32:
		w.value, w.bits = float64(f), 32
	default:
		return whole{}, false
	}
	return w, decimal(w.value)
}

// Format writes the value in decimal under %v and %+v, and as fmt writes
// the float under any other directive.
func (w whole) Format(f fmt.State, verb rune) {
	directive := fmt.FormatString(f, verb)
	if directive != "%v" && directive != "%+v" {
		_, _ = fmt.Fprintf(f, directive, w.arg)
		return
	}
	_, _ = io.WriteString(f, strconv.FormatFloat(w.value, 'f', -1, w.bits))
}

// decimal reports whether %v writes f in decimal: whether f is a whole
// number of at least decimalFrom and below decimalBelow in magnitude.
func decimal(f float64) bool {
	magnitude := math.Abs(f)
	return f == math.Trunc(f) && magnitude >= decimalFrom && magnitude < decimalBelow
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
	w.visit(valueOf(v), 0)
	return !w.cycled && w.parts <= maxParts
}

// structural returns the structural text of v.
func structural(v any) structured {
	var b strings.Builder
	w := walk{out: &b}
	w.visit(valueOf(v), 0)
	return structured(b.String())
}

// valueOf returns the value of an argument that fmt writes: the value inside
// a reflect.Value, and the argument itself for any other type.
func valueOf(arg any) reflect.Value {
	if v, ok := arg.(reflect.Value); ok {
		return v
	}
	return reflect.ValueOf(arg)
}

// walk is one walk of a value's parts, in the order fmt visits them.
type walk struct {
	// parts counts the values visited.
	parts int
	// path are the maps and slices that contain the value being visited.
	path cycle.Path
	// cycled reports that the walk met a map or a slice inside itself.
	cycled bool
	// out receives the structural text, and is nil for a walk that checks
	// the value alone and stops at its first cycle or past maxParts.
	out *strings.Builder
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
// walks it. A walk that writes starts at a valid value, because the walk
// that checks counts the invalid value of a reflect.Value as one part.
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
	if !w.enter(v) {
		return
	}
	defer w.path.Leave()
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
		if !w.enter(v) {
			return
		}
		defer w.path.Leave()
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
// number and a string as fmt's %v writes them, but a float that %v writes
// in decimal, and a channel, a function and an unsafe pointer as an
// address.
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
		format := byte('g')
		if decimal(v.Float()) {
			format = 'f'
		}
		w.write(strconv.FormatFloat(v.Float(), format, -1, v.Type().Bits()))
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

// enter adds the map or the slice v to the path of the walk, and reports
// whether the path did not contain it already. A walk that meets one inside
// itself marks the cycle and does not walk it again.
func (w *walk) enter(v reflect.Value) bool {
	if _, entered := w.path.Enter(v); !entered {
		w.cycled = true
		w.write(cycleMark)
		return false
	}
	return true
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
