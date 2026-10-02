// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"context"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// HonoursCancellation calls fn with a context that is already
// cancelled, and records a failure and lets the test continue when fn
// does not return a cancellation error.
//
//	expect.HonoursCancellation(t, func(ctx context.Context) error {
//	    _, err := store.Get(ctx, id)
//	    return err
//	}, "Get reports a cancelled caller")
//
// [context.Canceled] and [context.DeadlineExceeded] both count, and
// the error may be wrapped. The cancellation is in place before fn
// starts, so this asks whether fn checks at all rather than how
// quickly it notices.
func HonoursCancellation(tb assert.TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.HonoursCancellation(tb, matcher.Soft, fn, msg)
}

// HonoursDeadline calls fn with a context whose deadline has already
// passed, and records a failure and lets the test continue when fn does
// not return a deadline error.
//
// This differs from [HonoursCancellation] in which failure it asks
// for: a subject may distinguish a caller who gave up from one who ran
// out of time.
func HonoursDeadline(tb assert.TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.HonoursDeadline(tb, matcher.Soft, fn, msg)
}

// CompletesWithin calls fn with a context whose deadline is within from
// now, and records a failure when fn takes longer than within.
//
// The verdict is the time fn took, on the seat's clock. What fn returns
// does not count, because failing quickly is still finishing, and which
// failures are acceptable is a question for another assertion.
//
// fn runs on a goroutine of its own. A subject still running when the
// deadline passes fails then, and runs on, because no goroutine can be
// stopped from outside. A panic in fn panics again on the calling
// goroutine.
//
// This spends real time, up to within.
func CompletesWithin(tb assert.TB, within time.Duration, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.CompletesWithin(tb, matcher.Soft, within, fn, msg)
}

// Pure reads observable state with observe, calls fn, reads it again,
// and records a failure when the two readings differ.
//
//	expect.Pure(t,
//	    func() []Item { return store.List(ctx) },
//	    func() { _, _ = store.Get(ctx, id) },
//	    "Get does not disturb the store")
//
// The projection observe returns defines what changing nothing means:
// whatever it leaves out, fn is free to change. Return a copy. A
// projection sharing memory with the subject reads the same value
// twice and passes whatever fn did. Leave out anything that moves on
// its own, such as a clock reading or a generated identifier.
func Pure[S any](tb assert.TB, observe func() S, fn func(), msg string, opts ...Option) {
	tb.Helper()
	matcher.Pure(tb, matcher.Soft, observe, fn, msg, opts...)
}

// NilContextSafe calls fn with a nil context, and records a failure and
// lets the test continue when fn panics.
//
// An error back is fine and expected. The question is only whether a
// subject handed no context crashes, which a caller does by accident
// and a middlebox does by omission.
func NilContextSafe(tb assert.TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.NilContextSafe(tb, matcher.Soft, fn, msg)
}
