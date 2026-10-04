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
// starts, so the assertion checks whether fn reads its context at all,
// not how quickly it notices.
//
// # Allocation contract
//
// A passing call allocates twice besides what fn allocates: the context
// that it cancels.
func HonoursCancellation(tb TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.HonoursCancellation(tb, matcher.Fatal, fn, msg)
}

// HonoursDeadline calls fn with a context whose deadline has already
// passed, and stops the test when fn does not come back with a
// deadline error.
//
// The assertion differs from [HonoursCancellation] in the context that
// it hands fn, because a subject may treat a caller who gave up apart
// from one who ran out of time.
//
// # Allocation contract
//
// A passing call allocates twice besides what fn allocates: the context
// whose deadline has passed.
func HonoursDeadline(tb TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.HonoursDeadline(tb, matcher.Fatal, fn, msg)
}

// CompletesWithin calls fn with a context whose deadline is within from
// now, and stops the test when fn takes longer than within.
//
// The verdict is the time that fn took, on the seat's clock. An error
// back from fn passes, because failing quickly is still finishing. Other
// assertions state which failures are acceptable.
//
// fn runs on a goroutine of its own. A subject that is still running when
// the deadline passes fails then and runs on, because a goroutine cannot
// be stopped from outside. A panic in fn panics again on the calling
// goroutine.
//
// The assertion spends real time, up to within.
//
// # Allocation contract
//
// A passing call allocates 14 times besides what fn allocates: the context
// with its deadline, and the goroutine of fn with its state.
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
// observe returns a projection of the state, and fn passes when the
// projection is the same before and after it. fn may change whatever the
// projection leaves out. Return a copy from observe. A projection that
// shares memory with the subject reads the same value twice and passes
// whatever fn did. Leave out anything that moves on its own, such as a
// clock reading or a generated identifier.
//
// # Allocation contract
//
// A passing call with readings of one int below 256 allocates nothing.
func Pure[S any](tb TB, observe func() S, fn func(), msg string, opts ...Option) {
	tb.Helper()
	matcher.Pure(tb, matcher.Fatal, observe, fn, msg, opts...)
}

// NilContextSafe calls fn with a nil context and stops the test when
// fn panics.
//
// An error back from fn passes. The assertion checks only that a subject
// handed no context does not crash, because a caller passes a nil context
// by accident.
//
// # Allocation contract
//
// A passing call allocates nothing besides what fn allocates.
func NilContextSafe(tb TB, fn func(ctx context.Context) error, msg string) {
	tb.Helper()
	matcher.NilContextSafe(tb, matcher.Fatal, fn, msg)
}
