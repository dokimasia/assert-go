// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"context"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestBehaviour(t *testing.T) {
	t.Parallel()

	t.Run("HonoursCancellation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursCancellation(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			assert.HonoursCancellation(s, fn, msg)
		})
	})

	t.Run("HonoursDeadline", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursDeadline(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			assert.HonoursDeadline(s, fn, msg)
		})
	})

	t.Run("CompletesWithin", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCompletesWithin(t, func(s *matchertest.Seat, within time.Duration,
			fn func(ctx context.Context) error, msg string,
		) {
			assert.CompletesWithin(s, within, fn, msg)
		})
	})

	t.Run("Pure", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPure(t, func(s *matchertest.Seat, observe func() []int,
			fn func(), msg string,
		) {
			assert.Pure(s, observe, fn, msg)
		})
	})

	t.Run("NilContextSafe", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNilContextSafe(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			assert.NilContextSafe(s, fn, msg)
		})
	})
}
