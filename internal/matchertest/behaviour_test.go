// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The subjects these suites drive are declared in the package under
// test. Each twin here drives a correct implementation through the
// suite, which proves the suite is satisfiable: a suite nobody can
// pass is worse than none, because it looks like coverage.
func TestBehaviour(t *testing.T) {
	t.Parallel()

	// Both suites drive a subject with a context that is already done,
	// so one correct implementation satisfies each.
	honours := func(s *matchertest.Seat, fn func(ctx context.Context) error, msg string) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := fn(ctx)
		switch {
		case err == nil:
			s.Report(matcher.Failure{
				Assertion: "honours-cancellation", Contract: msg,
				Detail: map[string]any{"got": nil},
			}, true)
		case !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded):
			s.Report(matcher.Failure{
				Assertion: "honours-cancellation", Contract: msg,
				Detail: map[string]any{"got": err},
			}, true)
		}
	}

	t.Run("RunHonoursCancellation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursCancellation(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			honours(s, fn, msg)
		})
	})

	t.Run("RunHonoursDeadline", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursDeadline(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			honours(s, fn, msg)
		})
	})

	t.Run("RunCompletesWithin", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCompletesWithin(t, func(s *matchertest.Seat, within time.Duration,
			fn func(ctx context.Context) error, msg string,
		) {
			started := time.Now()
			ctx, cancel := context.WithTimeout(t.Context(), within)
			defer cancel()

			type outcome struct {
				raised   any
				returned bool
			}
			ended := make(chan outcome, 1)
			go func() {
				var o outcome
				defer func() {
					o.raised = recover()
					ended <- o
				}()
				_ = fn(ctx)
				o.returned = true
			}()
			select {
			case o := <-ended:
				if o.raised != nil {
					panic(o.raised)
				}
				if o.returned && time.Since(started) <= within {
					return
				}
				<-ctx.Done()
			case <-ctx.Done():
			}
			s.Report(matcher.Failure{Assertion: "completes-within", Contract: msg}, true)
		})
	})

	t.Run("RunNilContextSafe", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNilContextSafe(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			defer func() {
				if r := recover(); r != nil {
					s.Report(matcher.Failure{
						Assertion: "nil-context-safe", Contract: msg,
						Detail: map[string]any{"got": r},
					}, true)
				}
			}()
			_ = fn(nil)
		})
	})
}

// TestBehaviourTwins runs TestBehaviourTwinsProcess in a child process, and
// requires the failures of RunCompletesWithin and RunPure for their broken
// twins.
func TestBehaviourTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestBehaviourTwinsProcess",
		"recovered <nil>, want the subject's own panic value",
		"the call did not run: hidden = 0, want 1")
}

// TestBehaviourTwinsProcess runs only in the child process of
// TestBehaviourTwins.
func TestBehaviourTwinsProcess(t *testing.T) {
	inChild(t)

	t.Run("RunCompletesWithin of a twin that recovers the subject's panic", func(t *testing.T) {
		matchertest.RunCompletesWithin(t, func(seat *matchertest.Seat, within time.Duration,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.CompletesWithin(seat, matcher.Fatal, within, func(ctx context.Context) error {
				defer func() { _ = recover() }()
				return fn(ctx)
			}, msg)
		})
	})

	t.Run("RunPure of a twin that never calls the subject", func(t *testing.T) {
		matchertest.RunPure(t, func(_ *matchertest.Seat, observe func() []int, _ func(), _ string) {
			_ = observe()
		})
	})
}
