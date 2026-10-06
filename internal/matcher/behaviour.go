// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

// HonoursCancellation calls fn with a context that is already
// cancelled, and reports when fn does not come back with a
// cancellation error.
//
// [context.Canceled] and [context.DeadlineExceeded] both count, and
// the error may be wrapped. A subject that ignores its context returns
// success or an unrelated error, and both fail.
//
// The cancellation is in place before fn starts, so the assertion
// checks whether fn reads its context at all, not how quickly it
// notices.
//
// # Allocation contract
//
// A passing call allocates twice besides what fn allocates: the context
// that it cancels.
func HonoursCancellation(seat Seat, mode Mode, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := fn(ctx)
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		Fail(seat, mode, "honours-cancellation", msg, map[string]any{"got": err})
		return
	}
	Pass(seat, mode, "honours-cancellation", msg)
}

// HonoursDeadline calls fn with a context whose deadline has already
// passed, and reports when fn does not come back with a deadline
// error.
//
// [context.DeadlineExceeded] and [context.Canceled] both count, and
// the error may be wrapped. The assertion differs from
// [HonoursCancellation] in the failure that it hands fn, because a
// subject may treat a caller who gave up apart from one who ran out of
// time.
//
// The deadline is read from the runtime clock, not from the seat's. A
// context decides expiry against the runtime clock alone, so a seat
// clock ahead of it would hand fn a deadline that has not passed, and
// a subject that honours it would fail.
//
// # Allocation contract
//
// A passing call allocates twice besides what fn allocates: the context
// whose deadline has passed.
func HonoursDeadline(seat Seat, mode Mode, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	ctx, cancel := context.WithDeadline(
		context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := fn(ctx)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		Fail(seat, mode, "honours-deadline", msg, map[string]any{"got": err})
		return
	}
	Pass(seat, mode, "honours-deadline", msg)
}

// CompletesWithin times fn and reports when it took longer than within.
//
// The verdict is the measured time. An error back from fn passes,
// because failing quickly is still finishing. Other assertions state
// which failures are acceptable, and [HonoursDeadline] checks whether a
// subject honours a deadline that it was handed.
//
// fn runs on a goroutine of its own. It receives a context whose
// deadline is within from now, so a subject that watches the context can
// return before it runs long. The deadline is read from the runtime
// clock, because [context] does not accept another clock. The verdict on
// a subject that returns is measured on the seat's clock, and a test that
// drives a controlled clock sets the time that the subject took. Under
// the default clock the two clocks agree.
//
// A subject that has not returned when the deadline passes fails then,
// with got the time waited on the runtime clock, and the assertion
// returns without it. A goroutine cannot be stopped from outside, so fn
// runs on, and a leak check after this call reports it. A subject that
// ends its goroutine through runtime.Goexit never returns either, so it
// fails when the deadline passes as well.
//
// Exactly one of fn and the deadline ends the wait. A panic in fn that
// ends it panics again on the calling goroutine. A panic in fn after the
// deadline panics on fn's own goroutine.
//
// The assertion spends real time, up to within.
//
// # Allocation contract
//
// A passing call allocates 14 times besides what fn allocates: the context
// with its deadline, and the goroutine of fn with its state.
func CompletesWithin(seat Seat, mode Mode, within time.Duration, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	// The readings precede the context, so a subject that returns after
	// the deadline has taken more than within on the runtime clock.
	clock := ClockOf(seat)
	started, waited := clock.Now(), time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()

	s := newSubject()
	stop := context.AfterFunc(ctx, s.expire)
	defer stop()
	go func() {
		defer s.end()
		_ = fn(ctx)
		s.returned = true
	}()

	switch raised := (<-s.outcome).(type) {
	case expired, exited:
		<-ctx.Done()
		Fail(seat, mode, "completes-within", msg, map[string]any{
			"want": within,
			"got":  time.Since(waited).Round(time.Millisecond),
		})
	case nil:
		if elapsed := clock.Now().Sub(started); elapsed > within {
			Fail(seat, mode, "completes-within", msg, map[string]any{
				"want": within,
				"got":  elapsed.Round(time.Millisecond),
			})
			return
		}
		Pass(seat, mode, "completes-within", msg)
	default:
		panic(raised)
	}
}

// The states of a subject that [CompletesWithin] runs. The subject and
// its deadline each try to move it out of running, and exactly one of
// them succeeds.
const (
	// running is a subject that has neither ended nor run out of time.
	running int32 = iota
	// finished is a subject that ended before its deadline. The caller
	// judges its outcome.
	finished
	// abandoned is a subject whose deadline passed first. The caller has
	// reported it, and nothing recovers a panic of the subject.
	abandoned
)

// subject is the state of a subject that [CompletesWithin] runs.
type subject struct {
	state atomic.Int32
	// outcome receives one value: the value that the subject panicked
	// with, nil for a subject that returned, exited, or expired.
	outcome chan any
	// returned reports that the subject returned. Only the subject's
	// goroutine reads and writes it.
	returned bool
}

// expired is the outcome of a subject whose deadline passed first.
type expired struct{}

// exited is the outcome of a subject that ended its goroutine through
// runtime.Goexit before its deadline, and so never returned.
type exited struct{}

// newSubject returns a subject that is running.
func newSubject() *subject {
	return &subject{outcome: make(chan any, 1)}
}

// end runs deferred on the subject's goroutine. When the subject ends
// first, it recovers the value that the subject panicked with, and hands
// the caller that value, nil for a subject that returned, or exited for a
// subject that neither panicked nor returned. When the deadline passed
// first, it recovers nothing, so a panic of the subject ends the program,
// as a panic on any goroutine does.
func (s *subject) end() {
	if !s.state.CompareAndSwap(running, finished) {
		return
	}
	raised := recover()
	if raised == nil && !s.returned {
		raised = exited{}
	}
	s.outcome <- raised
}

// expire ends the wait for a subject whose deadline passes first.
func (s *subject) expire() {
	if s.state.CompareAndSwap(running, abandoned) {
		s.outcome <- expired{}
	}
}

// Pure reads observable state with observe, calls fn, reads it again,
// and reports when the two readings differ.
//
// Use it to state that a read-only operation leaves the state that a
// caller can see unchanged. observe returns a projection of the state,
// and fn passes when the projection is the same before and after it. fn
// may change whatever the projection leaves out.
//
//	matcher.Pure(seat, matcher.Fatal,
//	    func() []Item { return store.List(ctx) },
//	    func() { _, _ = store.Get(ctx, id) },
//	    "Get does not disturb the store")
//
// Return a copy from observe. A projection that shares memory with the
// subject reads the same value twice and passes whatever fn did. Leave
// out anything that moves on its own, such as a clock reading or a
// generated identifier.
//
// # Allocation contract
//
// A passing call with readings of one int below 256 allocates nothing.
func Pure[S any](seat Seat, mode Mode, observe func() S, fn func(), msg string, opts ...Option) {
	seat.Helper()

	before := observe()
	fn()
	after := observe()

	if !equal(before, after, rulesOf(opts)) {
		Fail(seat, mode, "pure", msg, map[string]any{"want": before, "got": after})
		return
	}
	Pass(seat, mode, "pure", msg)
}

// NilContextSafe calls fn with a nil context and reports when fn
// panics.
//
// An error back from fn passes. The assertion checks only that a
// subject handed no context does not crash, because a caller passes a
// nil context by accident. A fn that ends its goroutine through
// runtime.Goexit neither panics nor returns, and the call states no
// verdict.
//
// # Allocation contract
//
// A passing call allocates nothing besides what fn allocates.
func NilContextSafe(seat Seat, mode Mode, fn func(ctx context.Context) error, msg string) {
	seat.Helper()

	returned := false
	defer func() {
		if r := recover(); r != nil {
			Fail(seat, mode, "nil-context-safe", msg, map[string]any{"got": r})
			return
		}
		if returned {
			Pass(seat, mode, "nil-context-safe", msg)
		}
	}()

	//nolint:staticcheck // passing nil is the subject of the assertion
	_ = fn(nil)
	returned = true
}
