// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"context"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
)

// HonoursCancellation calls fn with a context that is already
// cancelled, and stops the test when fn does not come back with a
// cancellation error.
//
//	assert.HonoursCancellation(t, func(ctx context.Context) error {
//	    _, err := store.Get(ctx, id)
//	    return err
//	}, "Get reports a cancelled caller")
//
// [context.Canceled] and [context.DeadlineExceeded] both count, and
// the error may be wrapped. The cancellation is in place before fn
// starts, so this asks whether fn checks at all rather than how
// quickly it notices.
func HonoursCancellation(tb TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.HonoursCancellation(tb, matcher.Fatal, fn, msg)
}

// HonoursDeadline calls fn with a context whose deadline has already
// passed, and stops the test when fn does not come back with a
// deadline error.
//
// This differs from [HonoursCancellation] in which failure it asks
// for: a subject may distinguish a caller who gave up from one who ran
// out of time.
func HonoursDeadline(tb TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.HonoursDeadline(tb, matcher.Fatal, fn, msg)
}

// CompletesWithin calls fn with a context carrying the given deadline
// and stops the test when fn does not finish in time.
//
// A [context.DeadlineExceeded] back from fn is the failure: fn had the
// deadline and did not meet it. Any other error passes, because
// failing quickly is still finishing, and which failures are
// acceptable is a question for another assertion.
//
// This spends real time, up to within.
func CompletesWithin(tb TB, within time.Duration, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.CompletesWithin(tb, matcher.Fatal, within, fn, msg)
}

// Pure reads observable state with observe, calls fn, reads it again,
// and stops the test when the two readings differ.
//
//	assert.Pure(t,
//	    func() []Item { return store.List(ctx) },
//	    func() { _, _ = store.Get(ctx, id) },
//	    "Get does not disturb the store")
//
// The projection observe returns defines what changing nothing means:
// whatever it leaves out, fn is free to change. Return a copy. A
// projection sharing memory with the subject reads the same value
// twice and passes whatever fn did. Leave out anything that moves on
// its own, such as a clock reading or a generated identifier.
func Pure[S any](tb TB, observe func() S, fn func(), msg string, opts ...Option) {
	tb.Helper()
	matcher.Pure(tb, matcher.Fatal, observe, fn, msg, opts...)
}

// NilContextSafe calls fn with a nil context and stops the test when
// fn panics.
//
// An error back is fine and expected. The question is only whether a
// subject handed no context crashes, which a caller does by accident
// and a middlebox does by omission.
func NilContextSafe(tb TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.NilContextSafe(tb, matcher.Fatal, fn, msg)
}
