// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"
	"strings"

	"go.dokimi.dev/assert/internal/equality"
)

// Contains reports when haystack does not contain needle.
//
// What containing means depends on the haystack. Text contains text as
// a substring. A slice or array contains an element that equals needle
// as [Equal] compares under opts, so an int does not match a float. A
// map contains a key that equals needle as [Equal] compares under opts,
// so a pointer key matches a needle of an equal target, and a NaN key
// matches a NaN needle under [EquateNaNs] alone. A haystack of any other
// type fails the assertion.
//
// # Allocation contract
//
// A passing call on text allocates nothing. A passing call on a slice of
// three ints whose last element is needle allocates once: the interface of
// the slice.
func Contains(seat Seat, mode Mode, haystack, needle any, msg string, opts ...Option) {
	seat.Helper()

	// contained reports false for a haystack of a type without containment.
	found, _ := contained(haystack, needle, rulesOf(opts))
	if !found {
		Fail(seat, mode, "contains", msg, map[string]any{"haystack": haystack, "needle": needle})
		return
	}
	Pass(seat, mode, "contains", msg)
}

// NotContains reports when haystack contains needle. See [Contains] for
// what containing means.
//
// # Allocation contract
//
// A passing call on text allocates nothing. A call on a slice or an array
// compares its elements as [Contains] does.
func NotContains(seat Seat, mode Mode, haystack, needle any, msg string, opts ...Option) {
	seat.Helper()

	found, supported := contained(haystack, needle, rulesOf(opts))
	if !supported {
		Fail(seat, mode, "not-contains", msg, map[string]any{"haystack": haystack, "needle": needle})
		return
	}
	if found {
		Fail(seat, mode, "not-contains", msg, map[string]any{"haystack": haystack, "needle": needle})
		return
	}
	Pass(seat, mode, "not-contains", msg)
}

// ContainsInOrder reports when haystack does not contain every needle
// in the given order, each one after the previous one's match ends.
//
// Use it where [Contains] is too weak. Asserting that fields appear in
// a stated order catches a formatter that reorders them, which
// checking for each field separately does not.
//
// The failure states the first needle not found and its index in
// needles. An empty needle list passes.
//
// # Allocation contract
//
// A passing call on a string allocates nothing. A call on a []byte
// allocates twice: the bytes in the interface of haystack, and their text.
func ContainsInOrder(seat Seat, mode Mode, haystack any, needles []string, msg string) {
	seat.Helper()

	text, ok := textOf(haystack)
	if !ok {
		Fail(seat, mode, "contains-in-order", msg,
			map[string]any{"haystack": haystack, "needle": "", "index": 0})
		return
	}

	cursor := 0
	for i, needle := range needles {
		at := strings.Index(text[cursor:], needle)
		if at < 0 {
			Fail(seat, mode, "contains-in-order", msg,
				map[string]any{"haystack": text, "needle": needle, "index": i})
			return
		}
		cursor += at + len(needle)
	}
	Pass(seat, mode, "contains-in-order", msg)
}

// Permutation reports when got and want do not contain the same elements,
// each as often, in any order.
//
// Elements compare as [Equal] compares them under opts. Each element of
// want is matched with an unmatched equal element of got, which is at most
// len(got)·len(want) comparisons. Two empty slices match when both are nil
// or neither is, so a nil slice does not match an empty one unless opts
// equate empty values. The identity of the slices themselves never
// matters, under [ByIdentity] too.
//
// # Allocation contract
//
// A passing call on two slices of three ints allocates twice: the interface
// of each slice, through which it compares their elements.
func Permutation[T any](seat Seat, mode Mode, got, want []T, msg string, opts ...Option) {
	seat.Helper()

	if !permuted(got, want, rulesOf(opts)) {
		Fail(seat, mode, "permutation", msg, map[string]any{"want": want, "got": got})
		return
	}
	Pass(seat, mode, "permutation", msg)
}

// permuted reports whether got and want contain the same elements, each as
// often, under r. Equal is an equivalence on the values that equal
// themselves, so matching each element of want with the first unmatched
// equal element of got never takes a match that another element needs.
func permuted[T any](got, want []T, r equality.Rules) bool {
	if len(got) != len(want) {
		return false
	}
	if len(got) == 0 {
		return (got == nil) == (want == nil) || r.EquateEmpty
	}

	g, w := reflect.ValueOf(got), reflect.ValueOf(want)
	matched := make([]bool, len(got))
	for j := range want {
		at := -1
		for i := range got {
			if !matched[i] && equality.Equal(g.Index(i), w.Index(j), r) {
				at = i
				break
			}
		}
		if at < 0 {
			return false
		}
		matched[at] = true
	}
	return true
}

// contained reports whether haystack contains needle under r, and whether
// the question applies to haystack's type at all.
//
// Text contains text as a substring. A slice or array contains an
// element that is equal under r. A map contains a key that is equal under
// r. A nil haystack has the kind [reflect.Invalid], and the question does
// not apply to it.
func contained(haystack, needle any, r equality.Rules) (found, supported bool) {
	if text, ok := textOf(haystack); ok {
		sub, ok := textOf(needle)
		if !ok {
			return false, false
		}
		return strings.Contains(text, sub), true
	}

	rv, key := reflect.ValueOf(haystack), reflect.ValueOf(needle)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := range rv.Len() {
			if equality.Equal(rv.Index(i), key, r) {
				return true, true
			}
		}
		return false, true
	case reflect.Map:
		return equality.HasKey(rv, key, r), true
	default:
		return false, false
	}
}
