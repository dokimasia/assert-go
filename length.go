// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Length stops the test when got does not have want items.
//
// It counts the elements of an array, slice or channel, the entries of a
// map, the bytes of a []byte and the Unicode scalar values of a string,
// so "é" has one. Anything else has no length, nil included, and fails
// with got nil. It does not panic.
//
// # Allocation contract
//
// A passing call on a slice allocates nothing.
func Length(tb TB, got any, want int, msg string) {
	tb.Helper()
	matcher.Length(tb, matcher.Fatal, got, want, msg)
}

// Empty stops the test when got has any item. A value without a length,
// nil included, fails. See [Length] for the values that have one. The
// failure states got, so it names the items.
//
// # Allocation contract
//
// A passing call on a slice allocates once: the interface of the slice,
// which a failure states as got.
func Empty(tb TB, got any, msg string) {
	tb.Helper()
	matcher.Empty(tb, matcher.Fatal, got, msg)
}

// NotEmpty stops the test when got has no item. A value without a
// length, nil included, fails. See [Length] for the values that have
// one. The failure states got.
//
// # Allocation contract
//
// A passing call on a slice allocates once: the interface of the slice,
// which a failure states as got.
func NotEmpty(tb TB, got any, msg string) {
	tb.Helper()
	matcher.NotEmpty(tb, matcher.Fatal, got, msg)
}
