// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Panics runs fn, and records a failure and lets the test continue when
// fn returns without panicking. It returns whatever fn panicked with,
// so a caller can assert on the reason as well as the fact.
//
//	reason := expect.Panics(t, func() { store.MustGet("") },
//	    "MustGet rejects an empty key")
//	expect.Contains(t, reason, "empty key", "and names the argument")
//
// A panic with a nil value is still a panic and passes. The runtime
// replaces the nil with a [runtime.PanicNilError], so the returned value
// is that error rather than nil. Read the seat, not the return, to learn
// whether fn panicked.
//
// # Allocation contract
//
// A passing call allocates nothing besides what fn and its panic allocate.
func Panics(tb assert.TB, fn func(), msg string) any {
	tb.Helper()
	return matcher.Panics(tb, matcher.Soft, fn, msg)
}

// NotPanics runs fn, and records a failure and lets the test continue
// when fn panics, naming what it panicked with. The panic is recovered,
// so a subject that panics fails its test instead of ending the test
// binary.
//
// Use it for a call that may return an error and must not panic.
//
//	expect.NotPanics(t, func() { _ = store.Put(ctx, Item{}) },
//	    "Put does not panic on a zero item")
//
// # Allocation contract
//
// A passing call allocates nothing besides what fn allocates.
func NotPanics(tb assert.TB, fn func(), msg string) {
	tb.Helper()
	matcher.NotPanics(tb, matcher.Soft, fn, msg)
}
