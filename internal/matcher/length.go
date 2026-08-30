// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "reflect"

// Length reports when got does not hold want items.
//
// It answers for an array, slice, map, channel or string. Anything
// else has no length, and passing one is itself the failure rather
// than a panic.
func Length(seat Seat, mode Mode, got any, want int, msg string) {
	seat.Helper()

	n, ok := lengthOf(got)
	if !ok {
		Report(seat, mode, "%s: length not supported for %T", msg, got)
		return
	}
	if n != want {
		Report(seat, mode, "%s: expected length %d, got %d", msg, want, n)
	}
}

// Empty reports when got holds anything. See [Length] for the types
// that answer.
func Empty(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()

	n, ok := lengthOf(got)
	if !ok {
		Report(seat, mode, "%s: emptiness not supported for %T", msg, got)
		return
	}
	if n != 0 {
		Report(seat, mode, "%s: expected empty, got length %d", msg, n)
	}
}

// NotEmpty reports when got holds nothing. See [Length] for the types
// that answer.
func NotEmpty(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()

	n, ok := lengthOf(got)
	if !ok {
		Report(seat, mode, "%s: emptiness not supported for %T", msg, got)
		return
	}
	if n == 0 {
		Report(seat, mode, "%s: expected non-empty, got length 0", msg)
	}
}

// lengthOf reads the length of anything that has one: an array, slice,
// map, channel or string.
func lengthOf(v any) (int, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0, false
	}

	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.String:
		return rv.Len(), true
	default:
		return 0, false
	}
}
