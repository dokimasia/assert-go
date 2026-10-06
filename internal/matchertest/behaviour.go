// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// The subjects that the behavioural suites drive. Every surface's suite
// drives the same subjects.
var (
	// RespectsCtx returns the error of its context, as a subject that
	// checks cancellation does.
	RespectsCtx = func(ctx context.Context) error { return ctx.Err() }
	// IgnoresCtx returns success whatever the state of its context.
	IgnoresCtx = func(context.Context) error { return nil }
	// WrapsCtx returns the error of its context wrapped, so a suite
	// proves that the assertion walks the chain of the error.
	WrapsCtx = func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("matchertest: %w", err)
		}
		return nil
	}
	// FailsOtherwise fails for a reason that is not cancellation.
	FailsOtherwise = func(context.Context) error { return ErrSample }
)

// CtxInvoke calls a surface's assertion over a context-taking subject.
type CtxInvoke func(seat *Seat, fn func(ctx context.Context) error, msg string)

// WithinInvoke calls a surface's deadline assertion.
type WithinInvoke func(seat *Seat, within time.Duration, fn func(ctx context.Context) error, msg string)

// PureInvoke calls a surface's purity assertion. The projection is
// fixed at a slice of ints: what varies between surfaces is the call.
type PureInvoke func(seat *Seat, observe func() []int, fn func(), msg string)

// ctxCases are the subjects a cancellation or deadline assertion must
// judge, and how.
var ctxCases = []struct {
	name      string
	fn        func(ctx context.Context) error
	fails     bool
	assertion string
	detail    map[string]any
}{
	{name: "a subject that checks its context passes", fn: RespectsCtx},
	{name: "a wrapped context error passes", fn: WrapsCtx},
	{
		name: "a subject that returns context.Canceled passes",
		fn:   func(context.Context) error { return context.Canceled },
	},
	{
		name: "a subject that returns context.DeadlineExceeded passes",
		fn:   func(context.Context) error { return context.DeadlineExceeded },
	},
	{
		name:   "a subject that ignores its context reports",
		fn:     IgnoresCtx,
		fails:  true,
		detail: map[string]any{"got": nil},
	},
	{
		name:  "an unrelated error reports",
		fn:    FailsOtherwise,
		fails: true,
	},
}

// RunHonoursCancellation drives invoke against every case a
// cancellation assertion must produce.
func RunHonoursCancellation(t *testing.T, invoke CtxInvoke) {
	t.Helper()
	runCtxCases(t, invoke)
}

// RunHonoursDeadline drives invoke against every case a deadline
// assertion must produce. A subject cannot tell an expired deadline
// from a cancellation without inspecting the error, so the cases match
// the cancellation ones.
func RunHonoursDeadline(t *testing.T, invoke CtxInvoke) {
	t.Helper()
	runCtxCases(t, invoke)
}

func runCtxCases(t *testing.T, invoke CtxInvoke) {
	t.Helper()

	for _, tc := range ctxCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seat := &Seat{}
			invoke(seat, tc.fn, contractMsg)
			checkOutcome(t, seat, Case{Fails: tc.fails, Assertion: tc.assertion, Detail: tc.detail})
		})
	}
}

// RunCompletesWithin drives invoke against every case a deadline
// assertion must produce.
func RunCompletesWithin(t *testing.T, invoke WithinInvoke) {
	t.Helper()

	t.Run("a fast subject passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, time.Second, IgnoresCtx, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a subject that fails quickly still passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, time.Second, FailsOtherwise, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a subject that reads its context before the deadline passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, time.Second, WrapsCtx, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a subject that runs out of time reports", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "completes-within"})
	})

	// The verdict is the time taken, not the subject's result. A subject
	// that ignores its context and returns success late has missed the
	// ceiling, and an assertion that reads the result instead of the
	// clock passes it.
	t.Run("a subject that overruns without watching reports", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, func(context.Context) error {
			time.Sleep(4 * ShortTimeout)
			return nil
		}, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "completes-within"})
	})

	// The subject waits on a channel that is closed only after the
	// assertion returns, so the assertion returns at the deadline or the
	// test times out.
	t.Run("a subject that never returns reports at the deadline", func(t *testing.T) {
		t.Parallel()

		release := make(chan struct{})
		defer close(release)

		seat := &Seat{}
		invoke(seat, ShortTimeout, func(context.Context) error {
			<-release
			return nil
		}, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "completes-within"})
	})

	t.Run("a subject that ends its goroutine reports at the deadline", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, func(context.Context) error {
			runtime.Goexit()
			return nil
		}, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "completes-within"})
	})

	t.Run("a panic of the subject panics again on the caller", func(t *testing.T) {
		t.Parallel()

		raised := Raised(func() {
			invoke(&Seat{}, PatientTimeout, func(context.Context) error { panic(ErrSample) }, contractMsg)
		})
		if err, _ := raised.(error); !errors.Is(err, ErrSample) {
			t.Fatalf("recovered %v, want the subject's own panic value", raised)
		}
	})
}

// Raised calls fn and returns the value that fn panicked with, or nil
// when fn returned.
func Raised(fn func()) (raised any) {
	defer func() { raised = recover() }()
	fn()
	return nil
}

// RunPure drives invoke against every case a purity assertion must
// produce.
func RunPure(t *testing.T, invoke PureInvoke) {
	t.Helper()

	t.Run("an unchanged projection passes", func(t *testing.T) {
		t.Parallel()

		state := []int{1, 2}
		seat := &Seat{}
		invoke(seat, func() []int { return append([]int(nil), state...) },
			func() {}, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a changed projection reports", func(t *testing.T) {
		t.Parallel()

		state := []int{1, 2}
		seat := &Seat{}
		invoke(seat, func() []int { return append([]int(nil), state...) },
			func() { state = append(state, 3) }, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "pure"})
	})

	t.Run("a change outside the projection passes", func(t *testing.T) {
		t.Parallel()

		state, hidden := []int{1, 2}, 0
		seat := &Seat{}
		invoke(seat, func() []int { return append([]int(nil), state...) },
			func() { hidden++ }, contractMsg)
		checkOutcome(t, seat, Case{})

		if hidden != 1 {
			t.Fatalf("the call did not run: hidden = %d, want 1", hidden)
		}
	})
}

// RunNilContextSafe drives invoke against every case an absent-context
// assertion must produce.
func RunNilContextSafe(t *testing.T, invoke CtxInvoke) {
	t.Helper()

	t.Run("a subject that returns an error passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, func(context.Context) error {
			return errors.New("matchertest: refused")
		}, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a subject that succeeds passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, IgnoresCtx, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a subject that crashes reports", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, RespectsCtx, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "nil-context-safe"})
	})
}
