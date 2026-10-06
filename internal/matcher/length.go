// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"
	"unicode/utf8"
)

// Length reports when got does not have want items.
//
// It counts the elements of an array, slice or channel, the entries of a
// map, and the Unicode scalar values of a string, so "é" has one. A
// []byte is a slice, so its length is its bytes. A byte of a string that
// is not UTF-8 counts as one. Anything else has no length, and Length
// fails for it with got nil.
//
// # Allocation contract
//
// A passing call on a slice allocates nothing.
func Length(seat Seat, mode Mode, got any, want int, msg string) {
	seat.Helper()

	n, ok := lengthOf(got)
	if !ok {
		Fail(seat, mode, "length", msg, map[string]any{"want": want, "got": nil})
		return
	}
	if n != want {
		Fail(seat, mode, "length", msg, map[string]any{"want": want, "got": n})
		return
	}
	Pass(seat, mode, "length", msg)
}

// Empty reports when got has any item. See [Length] for the types that
// have a length. A value without one, nil included, is no container, and
// fails with length nil.
//
// # Allocation contract
//
// A passing call on a slice allocates nothing.
func Empty(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()

	n, ok := lengthOf(got)
	if !ok {
		Fail(seat, mode, "empty", msg, map[string]any{"length": nil})
		return
	}
	if n != 0 {
		Fail(seat, mode, "empty", msg, map[string]any{"length": n})
		return
	}
	Pass(seat, mode, "empty", msg)
}

// NotEmpty reports when got has no item. See [Length] for the types that
// have a length. A value without one, nil included, is no container, and
// fails.
//
// # Allocation contract
//
// A passing call on a slice allocates nothing.
func NotEmpty(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()

	// A value without a length has no item.
	if n, _ := lengthOf(got); n == 0 {
		Fail(seat, mode, "not-empty", msg, nil)
		return
	}
	Pass(seat, mode, "not-empty", msg)
}

// lengthOf reads the length of anything that has one: the elements of an
// array, slice or channel, the entries of a map, and the Unicode scalar
// values of a string. A nil value has the kind [reflect.Invalid], and no
// length.
func lengthOf(v any) (int, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return utf8.RuneCountInString(rv.String()), true
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return rv.Len(), true
	default:
		return 0, false
	}
}
