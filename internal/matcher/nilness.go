// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "reflect"

// Nil reports when got is not nil.
//
// A typed nil counts as nil: a (*T)(nil) held in an interface is not
// equal to a plain nil under ==, but it is nil for every purpose a
// test cares about, and treating it otherwise is a trap rather than a
// distinction.
func Nil(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()
	if !isNil(got) {
		Fail(seat, mode, "nil", msg, map[string]any{"got": got})
	}
}

// NotNil reports when got is nil. A typed nil counts as nil; see
// [Nil].
func NotNil(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()
	if isNil(got) {
		Fail(seat, mode, "not-nil", msg, nil)
	}
}

// isNil reports whether v is nil, including a typed nil such as a
// (*T)(nil) held in an interface. A plain == nil misses those, which
// is the trap the nil assertions exist to avoid.
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
