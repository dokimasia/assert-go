// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"context"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestBehaviour(t *testing.T) {
	t.Parallel()

	t.Run("HonoursCancellation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursCancellation(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.HonoursCancellation(s, matcher.Fatal, fn, msg)
		})
	})

	t.Run("HonoursDeadline", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursDeadline(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.HonoursDeadline(s, matcher.Fatal, fn, msg)
		})
	})

	t.Run("CompletesWithin", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCompletesWithin(t, func(s *matchertest.Seat, within time.Duration,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.CompletesWithin(s, matcher.Fatal, within, fn, msg)
		})
	})

	t.Run("Pure", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPure(t, func(s *matchertest.Seat, observe func() []int,
			fn func(), msg string,
		) {
			matcher.Pure(s, matcher.Fatal, observe, fn, msg)
		})
	})

	t.Run("NilContextSafe", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNilContextSafe(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.NilContextSafe(s, matcher.Fatal, fn, msg)
		})
	})
}
