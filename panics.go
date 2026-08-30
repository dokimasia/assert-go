// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Panics runs fn and stops the test when it returns without
// panicking. It returns whatever fn panicked with, so a caller can
// assert on the reason as well as the fact.
//
//	reason := assert.Panics(t, func() { store.MustGet("") },
//	    "MustGet rejects an empty key")
//	assert.Contains(t, reason, "empty key", "and says which argument was wrong")
//
// A panic carrying nil is still a panic and passes. Since Go 1.21 the
// runtime replaces the nil with a [runtime.PanicNilError], so the
// returned value is that error rather than nil. Read the seat, not the
// return, to learn whether fn panicked.
func Panics(tb TB, fn func(), msg string) any {
	tb.Helper()
	return matcher.Panics(tb, matcher.Fatal, fn, msg)
}

// NotPanics runs fn and stops the test when it panics, naming what it
// panicked with. The panic is recovered rather than allowed to unwind,
// so one misbehaving subject fails its test instead of taking the run
// down.
//
// This is the assertion for a call that may legitimately fail:
// returning an error is fine, crashing is not.
//
//	assert.NotPanics(t, func() { _ = store.Put(ctx, Item{}) },
//	    "Put survives a zero item")
func NotPanics(tb TB, fn func(), msg string) {
	tb.Helper()
	matcher.NotPanics(tb, matcher.Fatal, fn, msg)
}
