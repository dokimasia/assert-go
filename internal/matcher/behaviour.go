// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-cmp/cmp"
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
		Fail(seat, mode, "honours-cancellation", msg, map[string]any{"got": nil})
	case !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded):
		Fail(seat, mode, "honours-cancellation", msg, map[string]any{"got": err})
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
//
// The deadline is read from the runtime clock rather than the seat's.
// A context decides expiry against the runtime clock and takes no
// other, so a seat clock reading ahead of it would hand the subject a
// deadline that has not passed and fail a subject that was right.
func HonoursDeadline(seat Seat, mode Mode, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	ctx, cancel := context.WithDeadline(
		context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := fn(ctx)
	switch {
	case err == nil:
		Fail(seat, mode, "honours-deadline", msg, map[string]any{"got": nil})
	case !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled):
		Fail(seat, mode, "honours-deadline", msg, map[string]any{"got": err})
	}
}

// CompletesWithin times fn and reports when it took longer than within.
//
// The verdict is the measured time. An error back from fn passes,
// because failing quickly is still finishing, and which failure is
// acceptable is a question for another assertion. Whether a subject
// respects a deadline it was handed is [HonoursDeadline]; this asks
// only how long it took.
//
// fn receives a context carrying within as its deadline, so a subject
// that watches one can return rather than run long. That deadline is
// read from the runtime clock, because [context] offers no way to
// supply another. The verdict is not: it is measured on the seat's
// clock, so a test driving a controlled clock decides what the subject
// took. Under the default clock the two agree.
//
// This spends real time, up to however long fn takes.
func CompletesWithin(seat Seat, mode Mode, within time.Duration, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()

	clock := ClockOf(seat)
	started := clock.Now()
	_ = fn(ctx)
	elapsed := clock.Now().Sub(started)

	if elapsed > within {
		Fail(seat, mode, "completes-within", msg, map[string]any{
			"want": within,
			"got":  elapsed.Round(time.Millisecond),
		})
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

	if diff := cmp.Diff(before, after, Options(opts...)...); diff != "" {
		Fail(seat, mode, "pure", msg, map[string]any{"want": before, "got": after})
	}
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
			Fail(seat, mode, "nil-context-safe", msg, map[string]any{"got": r})
		}
	}()

	//nolint:staticcheck // passing nil is the subject of the assertion
	_ = fn(nil)
}
