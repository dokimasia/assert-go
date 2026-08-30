// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"context"
	"errors"
	"time"
)

// HonoursCancellation calls fn with a context that is already
// cancelled, and reports when fn does not come back with a
// cancellation error.
//
// [context.Canceled] and [context.DeadlineExceeded] both count, and
// the error may be wrapped. A subject that ignores its context returns
// success or some unrelated error, and both fail here.
//
// The cancellation is in place before fn starts, so this asks whether
// fn checks at all rather than how quickly it notices.
func HonoursCancellation(seat Seat, mode Mode, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := fn(ctx)
	switch {
	case err == nil:
		Report(seat, mode, "%s: a cancelled context produced no error", msg)
	case !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded):
		Report(seat, mode, "%s: a cancelled context produced %v, want a cancellation error", msg, err)
	}
}

// HonoursDeadline calls fn with a context whose deadline has already
// passed, and reports when fn does not come back with a deadline
// error.
//
// [context.DeadlineExceeded] and [context.Canceled] both count, and
// the error may be wrapped. This differs from [HonoursCancellation] in
// which failure it asks for: a subject may distinguish a caller who
// gave up from one who ran out of time.
func HonoursDeadline(seat Seat, mode Mode, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := fn(ctx)
	switch {
	case err == nil:
		Report(seat, mode, "%s: an expired deadline produced no error", msg)
	case !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled):
		Report(seat, mode, "%s: an expired deadline produced %v, want a deadline error", msg, err)
	}
}

// CompletesWithin calls fn with a context carrying the given deadline
// and reports when fn does not finish in time.
//
// A [context.DeadlineExceeded] back from fn is the failure: fn was
// given the deadline and did not meet it. Any other error passes,
// because failing quickly is still finishing, and which failure is
// acceptable is a question for another assertion.
//
// This spends real time, up to the deadline.
func CompletesWithin(seat Seat, mode Mode, within time.Duration, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()

	started := time.Now()
	if err := fn(ctx); errors.Is(err, context.DeadlineExceeded) {
		Report(seat, mode, "%s: did not finish within %v (gave up after %v)",
			msg, within, time.Since(started).Round(time.Millisecond))
	}
}

// Pure reads observable state with observe, calls fn, reads it again,
// and reports when the two readings differ.
//
// Use it to state that a read-only operation changes nothing a caller
// can see. observe returns a projection, and that projection defines
// what "nothing" means here: whatever it leaves out, fn is free to
// change.
//
//	matcher.Pure(seat, matcher.Fatal,
//	    func() []Item { return store.List(ctx) },
//	    func() { _, _ = store.Get(ctx, id) },
//	    "Get does not disturb the store")
//
// Return a copy from observe. A projection that shares memory with the
// subject reads the same value twice and passes whatever fn did.
// Leave out anything that moves on its own, such as a clock reading or
// a generated identifier.
func Pure[S any](seat Seat, mode Mode, observe func() S, fn func(), msg string, opts ...Option) {
	seat.Helper()

	before := observe()
	fn()
	after := observe()

	Equal(seat, mode, after, before, msg+": observable state changed", opts...)
}

// NilContextSafe calls fn with a nil context and reports when fn
// panics.
//
// An error back is fine and expected. The question is only whether a
// subject handed no context crashes, which is what a caller does by
// accident and a middlebox does by omission.
func NilContextSafe(seat Seat, mode Mode, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	defer func() {
		if r := recover(); r != nil {
			Report(seat, mode, "%s: a nil context caused a panic: %v", msg, r)
		}
	}()

	//nolint:staticcheck // passing nil is the subject of the assertion
	_ = fn(nil)
}
