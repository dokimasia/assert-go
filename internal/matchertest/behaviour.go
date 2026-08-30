// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// The subjects the behavioural suites drive. Naming them once keeps
// every surface asking the same questions of the same shapes.
var (
	// RespectsCtx returns whatever its context says, which is what a
	// subject that checks cancellation does.
	RespectsCtx = func(ctx context.Context) error { return ctx.Err() }
	// IgnoresCtx succeeds however its context stands.
	IgnoresCtx = func(context.Context) error { return nil }
	// WrapsCtx returns its context's error wrapped, so a suite proves
	// the chain is walked rather than the error compared directly.
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
	name     string
	fn       func(ctx context.Context) error
	fails    bool
	contains []string
}{
	{name: "a subject that checks its context passes", fn: RespectsCtx},
	{name: "a wrapped context error passes", fn: WrapsCtx},
	{
		name:     "a subject that ignores its context reports",
		fn:       IgnoresCtx,
		fails:    true,
		contains: []string{"no error"},
	},
	{
		name:     "an unrelated error reports",
		fn:       FailsOtherwise,
		fails:    true,
		contains: []string{"sample"},
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
			checkOutcome(t, seat, tc.fails, tc.contains)
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
		checkOutcome(t, seat, false, nil)
	})

	t.Run("a subject that fails quickly still passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, time.Second, FailsOtherwise, contractMsg)
		checkOutcome(t, seat, false, nil)
	})

	t.Run("a subject that runs out of time reports", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}, contractMsg)
		checkOutcome(t, seat, true, []string{"within"})
	})
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
		checkOutcome(t, seat, false, nil)
	})

	t.Run("a changed projection reports", func(t *testing.T) {
		t.Parallel()

		state := []int{1, 2}
		seat := &Seat{}
		invoke(seat, func() []int { return append([]int(nil), state...) },
			func() { state = append(state, 3) }, contractMsg)
		checkOutcome(t, seat, true, []string{"changed"})
	})

	t.Run("a change outside the projection passes", func(t *testing.T) {
		t.Parallel()

		state, hidden := []int{1, 2}, 0
		seat := &Seat{}
		invoke(seat, func() []int { return append([]int(nil), state...) },
			func() { hidden++ }, contractMsg)
		checkOutcome(t, seat, false, nil)

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
		checkOutcome(t, seat, false, nil)
	})

	t.Run("a subject that succeeds passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, IgnoresCtx, contractMsg)
		checkOutcome(t, seat, false, nil)
	})

	t.Run("a subject that crashes reports", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, RespectsCtx, contractMsg)
		checkOutcome(t, seat, true, []string{"panic"})
	})
}
