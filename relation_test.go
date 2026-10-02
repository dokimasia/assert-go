// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestRelation(t *testing.T) {
	t.Parallel()

	t.Run("Idempotent", func(t *testing.T) {
		t.Parallel()
		matchertest.RunIdempotent(t, func(s *matchertest.Seat, call func(int) error, input int,
			observe func() []int, msg string, opts ...assert.Option,
		) {
			assert.Idempotent(s, call, input, observe, msg, opts...)
		})
	})

	t.Run("Accumulates", func(t *testing.T) {
		t.Parallel()
		matchertest.RunAccumulates(t, func(s *matchertest.Seat, call func(int) error, input int,
			observe func() int, msg string,
		) {
			assert.Accumulates(s, call, input, observe, msg)
		})
	})

	t.Run("Deterministic", func(t *testing.T) {
		t.Parallel()
		matchertest.RunDeterministic(t, func(s *matchertest.Seat, call func(int) (float64, error), input int,
			msg string, opts ...assert.Option,
		) {
			assert.Deterministic(s, call, input, msg, opts...)
		})
	})

	t.Run("Commutative", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCommutative(t, func(s *matchertest.Seat, combine func(a, b float64) float64, a, b float64,
			msg string, opts ...assert.Option,
		) {
			assert.Commutative(s, combine, a, b, msg, opts...)
		})
	})

	t.Run("Associative", func(t *testing.T) {
		t.Parallel()
		matchertest.RunAssociative(t, func(s *matchertest.Seat, combine func(a, b float64) float64, a, b, c float64,
			msg string, opts ...assert.Option,
		) {
			assert.Associative(s, combine, a, b, c, msg, opts...)
		})
	})

	t.Run("RoundTrip", func(t *testing.T) {
		t.Parallel()
		matchertest.RunRoundTrip(t, func(s *matchertest.Seat, forward func(float64) (string, error),
			inverse func(string) (float64, error), input float64, msg string, opts ...assert.Option,
		) {
			assert.RoundTrip(s, forward, inverse, input, msg, opts...)
		})
	})

	t.Run("StableOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunStableOrder(t, func(s *matchertest.Seat, iterate func() ([]int, error), msg string,
			opts ...assert.Option,
		) {
			assert.StableOrder(s, iterate, msg, opts...)
		})
	})

	t.Run("NoDuplicates", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNoDuplicates(t, func(s *matchertest.Seat, iterate func() ([]float64, error), msg string,
			opts ...assert.Option,
		) {
			assert.NoDuplicates(s, iterate, msg, opts...)
		})
	})

	t.Run("Monotonic", func(t *testing.T) {
		t.Parallel()
		matchertest.RunMonotonic(t, func(s *matchertest.Seat, observe func() float64, advance func() error,
			steps int, msg string,
		) {
			assert.Monotonic(s, observe, advance, steps, msg)
		})
	})

	t.Run("Total", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTotal(t, func(s *matchertest.Seat, call func(int) error, domain []int, msg string) {
			assert.Total(s, call, domain, msg)
		})
	})

	t.Run("NotPure", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNotPure(t, func(s *matchertest.Seat, observe func() []int, fn func(), msg string,
			opts ...assert.Option,
		) {
			assert.NotPure(s, observe, fn, msg, opts...)
		})
	})

	t.Run("FailsAfterClose", func(t *testing.T) {
		t.Parallel()
		matchertest.RunFailsAfterClose(t, func(s *matchertest.Seat, closer, call func() error, sentinel error,
			msg string,
		) {
			assert.FailsAfterClose(s, closer, call, sentinel, msg)
		})
	})

	t.Run("Poisoned", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPoisoned(t, func(s *matchertest.Seat, induce func(), observe func() error, msg string) {
			assert.Poisoned(s, induce, observe, msg)
		})
	})
}
