// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "github.com/google/go-cmp/cmp"

// Equal compares got against want and reports when they differ. It
// reports nothing and returns when they are equal.
//
// The failure is msg, then a diff labelled -want +got. opts relax the
// comparison for this call alone; see [Options] for the rules that
// always apply.
//
//	matcher.Equal(seat, matcher.Fatal, store.Get(id), item,
//	    "Get returns the stored item")
//
// The type parameter refuses a mismatch at compile time, so got and
// want are always the same type by the time the comparison runs.
func Equal[T any](seat Seat, mode Mode, got, want T, msg string, opts ...Option) {
	seat.Helper()
	if diff := cmp.Diff(want, got, Options(opts...)...); diff != "" {
		Fail(seat, mode, "equal", msg, map[string]any{"want": want, "got": got})
	}
}

// NotEqual compares got against want and reports when they are equal.
// It reports nothing and returns when they differ.
//
// The failure is msg, then got. There is no diff to show: the two
// values matched, so printing one of them says everything the reader
// needs. opts relax the comparison for this call alone.
//
//	matcher.NotEqual(seat, matcher.Fatal, token, previous,
//	    "Refresh issues a new token")
func NotEqual[T any](seat Seat, mode Mode, got, want T, msg string, opts ...Option) {
	seat.Helper()
	if cmp.Equal(got, want, Options(opts...)...) {
		Fail(seat, mode, "not-equal", msg, map[string]any{"got": got})
	}
}
