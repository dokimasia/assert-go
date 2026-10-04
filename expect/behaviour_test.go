// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"context"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestBehaviour runs the shared cases of the behaviour assertions.
func TestBehaviour(t *testing.T) {
	t.Parallel()

	t.Run("HonoursCancellation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursCancellation(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			expect.HonoursCancellation(s, fn, msg)
		})
	})

	t.Run("HonoursDeadline", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursDeadline(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			expect.HonoursDeadline(s, fn, msg)
		})
	})

	t.Run("CompletesWithin", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCompletesWithin(t, func(s *matchertest.Seat, within time.Duration,
			fn func(ctx context.Context) error, msg string,
		) {
			expect.CompletesWithin(s, within, fn, msg)
		})
	})

	t.Run("Pure", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPure(t, func(s *matchertest.Seat, observe func() []int,
			fn func(), msg string,
		) {
			expect.Pure(s, observe, fn, msg)
		})
	})

	t.Run("NilContextSafe", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNilContextSafe(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			expect.NilContextSafe(s, fn, msg)
		})
	})
}

// TestBehaviourAllocs checks the allocation ceiling of a passing call of
// each behaviour assertion.
func TestBehaviourAllocs(t *testing.T) {
	alloctest.Check(t, behaviourCases())
}

// BenchmarkBehaviour measures a passing call of each behaviour assertion.
func BenchmarkBehaviour(b *testing.B) {
	for _, c := range behaviourCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// behaviourCases returns a passing call of each behaviour assertion, with
// its allocation ceiling, measured. Each subject returns at once.
func behaviourCases() []alloctest.Case {
	honours := func(ctx context.Context) error { return ctx.Err() }
	quick := func(context.Context) error { return nil }
	observe := func() int { return 1 }
	return []alloctest.Case{
		{Name: "HonoursCancellation", Allocs: 2, Call: func(tb assert.TB) {
			expect.HonoursCancellation(tb, honours, allocContract)
		}},
		{Name: "HonoursDeadline", Allocs: 2, Call: func(tb assert.TB) {
			expect.HonoursDeadline(tb, honours, allocContract)
		}},
		{Name: "CompletesWithin", Allocs: 14, Call: func(tb assert.TB) {
			expect.CompletesWithin(tb, time.Minute, quick, allocContract)
		}},
		{Name: "Pure", Call: func(tb assert.TB) { expect.Pure(tb, observe, func() {}, allocContract) }},
		{Name: "NilContextSafe", Call: func(tb assert.TB) { expect.NilContextSafe(tb, quick, allocContract) }},
	}
}
