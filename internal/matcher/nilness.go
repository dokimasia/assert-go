// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "reflect"

// Nil reports when got is not nil.
//
// A typed nil counts as nil: Nil passes for a (*T)(nil) in an interface,
// which == reports as unequal to a plain nil.
//
// # Allocation contract
//
// A passing call on a nil pointer allocates nothing.
func Nil(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()
	if !isNil(got) {
		Fail(seat, mode, "nil", msg, map[string]any{gotField: got})
		return
	}
	Pass(seat, mode, "nil", msg)
}

// NotNil reports when got is nil. A typed nil counts as nil, as it does
// for [Nil].
//
// # Allocation contract
//
// A passing call on a pointer allocates nothing.
func NotNil(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()
	if isNil(got) {
		Fail(seat, mode, "not-nil", msg, nil)
		return
	}
	Pass(seat, mode, "not-nil", msg)
}

// isNil reports whether v is nil, a typed nil such as a (*T)(nil) in an
// interface included. A plain == nil reports false for a typed nil.
func isNil(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice,
		reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}
