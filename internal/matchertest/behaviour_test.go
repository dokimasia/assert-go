// Copyright ThesmOS B.V. 2026
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
		ctx, cancel := context.WithCancel(context.Background())
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
			ctx, cancel := context.WithTimeout(context.Background(), within)
			defer cancel()
			started := time.Now()
			_ = fn(ctx)
			if time.Since(started) > within {
				s.Report(matcher.Failure{Assertion: "completes-within", Contract: msg}, true)
			}
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
			//nolint:staticcheck // passing nil is the subject of the suite
			_ = fn(nil)
		})
	})
}
