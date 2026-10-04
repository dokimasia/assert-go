// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"

	"go.dokimi.dev/assert/internal/equality"
)

// Equal compares got against want and reports when they differ. It
// reports nothing and returns when they are equal.
//
// The failure is msg, then a diff labelled -want +got. opts relax the
// comparison for this call alone. Two values are equal as the definition's
// equal states: the same type and the same value, every field taking part,
// and no method of either running.
//
//	matcher.Equal(seat, matcher.Fatal, store.Get(id), item,
//	    "Get returns the stored item")
//
// The type parameter refuses a mismatch at compile time, so got and
// want are always the same type by the time the comparison runs.
//
// # Allocation contract
//
// A passing call on two ints below 256 allocates nothing.
func Equal[T any](seat Seat, mode Mode, got, want T, msg string, opts ...Option) {
	seat.Helper()
	if !equal(got, want, rulesOf(opts)) {
		Fail(seat, mode, "equal", msg, map[string]any{"want": want, "got": got})
		return
	}
	Pass(seat, mode, "equal", msg)
}

// equal reports whether x and y are equal as [Equal] compares them under r.
func equal(x, y any, r equality.Rules) bool {
	return equality.Equal(reflect.ValueOf(x), reflect.ValueOf(y), r)
}

// NotEqual compares got against want and reports when they are equal.
// It reports nothing and returns when they differ.
//
// The failure is msg, then got. There is no diff to show: the two
// values matched, so printing one of them gives everything the reader
// needs. opts relax the comparison for this call alone.
//
//	matcher.NotEqual(seat, matcher.Fatal, token, previous,
//	    "Refresh issues a new token")
//
// # Allocation contract
//
// A passing call on two ints below 256 allocates nothing.
func NotEqual[T any](seat Seat, mode Mode, got, want T, msg string, opts ...Option) {
	seat.Helper()
	if equal(got, want, rulesOf(opts)) {
		Fail(seat, mode, "not-equal", msg, map[string]any{"got": got})
		return
	}
	Pass(seat, mode, "not-equal", msg)
}
