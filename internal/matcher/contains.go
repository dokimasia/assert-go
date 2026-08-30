// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// Contains reports when haystack does not hold needle.
//
// What holding means depends on the haystack: text holds text as a
// substring, a slice or array holds an element comparing equal under
// opts, and a map holds a key. Anything else cannot be asked, and
// asking is itself the failure.
func Contains(seat Seat, mode Mode, haystack, needle any, msg string, opts ...Option) {
	seat.Helper()

	found, supported := holds(haystack, needle, opts...)
	if !supported {
		Report(seat, mode, "%s: containment not supported for %T", msg, haystack)
		return
	}
	if !found {
		Report(seat, mode, "%s: %+v does not contain %+v", msg, haystack, needle)
	}
}

// NotContains reports when haystack holds needle. See [Contains] for
// what holding means.
func NotContains(seat Seat, mode Mode, haystack, needle any, msg string, opts ...Option) {
	seat.Helper()

	found, supported := holds(haystack, needle, opts...)
	if !supported {
		Report(seat, mode, "%s: containment not supported for %T", msg, haystack)
		return
	}
	if found {
		Report(seat, mode, "%s: %+v contains %+v, want it absent", msg, haystack, needle)
	}
}

// ContainsInOrder reports when haystack does not hold every needle in
// the given order, each one after the previous one's match ends.
//
// Use it where [Contains] is too weak. Asserting that fields appear in
// a stated order catches a formatter that reorders them, which
// checking for each field separately does not.
//
// The failure names the first needle not found and the position the
// search had reached, so a reader sees which one broke the order
// rather than only that the order broke. An empty needle list passes.
func ContainsInOrder(seat Seat, mode Mode, haystack any, needles []string, msg string) {
	seat.Helper()

	text, ok := textOf(haystack)
	if !ok {
		Report(seat, mode, "%s: ordered containment requires text, got %T", msg, haystack)
		return
	}

	cursor := 0
	for i, needle := range needles {
		at := strings.Index(text[cursor:], needle)
		if at < 0 {
			Report(seat, mode, "%s: needle %d (%q) not found after position %d in %q",
				msg, i, needle, cursor, text)
			return
		}
		cursor += at + len(needle)
	}
}

// holds reports whether haystack contains needle, and whether the
// question applies to haystack's type at all.
//
// Text contains text as a substring. A slice or array contains an
// element comparing equal under opts. A map contains a key.
func holds(haystack, needle any, opts ...Option) (found, supported bool) {
	if text, ok := textOf(haystack); ok {
		sub, ok := textOf(needle)
		if !ok {
			return false, false
		}
		return strings.Contains(text, sub), true
	}

	rv := reflect.ValueOf(haystack)
	if !rv.IsValid() {
		return false, false
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := range rv.Len() {
			if cmp.Equal(rv.Index(i).Interface(), needle, Options(opts...)...) {
				return true, true
			}
		}
		return false, true
	case reflect.Map:
		key := reflect.ValueOf(needle)
		if !key.IsValid() || !key.Type().AssignableTo(rv.Type().Key()) {
			return false, true
		}
		return rv.MapIndex(key).IsValid(), true
	default:
		return false, false
	}
}
