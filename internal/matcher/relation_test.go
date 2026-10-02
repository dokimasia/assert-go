// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestRelation(t *testing.T) {
	t.Parallel()

	t.Run("Idempotent", func(t *testing.T) {
		t.Parallel()
		matchertest.RunIdempotent(t, func(s *matchertest.Seat, call func(int) error, input int,
			observe func() []int, msg string, opts ...matcher.Option,
		) {
			matcher.Idempotent(s, matcher.Fatal, call, input, observe, msg, opts...)
		})
	})

	t.Run("Accumulates", func(t *testing.T) {
		t.Parallel()
		matchertest.RunAccumulates(t, func(s *matchertest.Seat, call func(int) error, input int,
			observe func() int, msg string,
		) {
			matcher.Accumulates(s, matcher.Fatal, call, input, observe, msg)
		})
	})

	t.Run("Deterministic", func(t *testing.T) {
		t.Parallel()
		matchertest.RunDeterministic(t, func(s *matchertest.Seat, call func(int) (float64, error), input int,
			msg string, opts ...matcher.Option,
		) {
			matcher.Deterministic(s, matcher.Fatal, call, input, msg, opts...)
		})
	})

	t.Run("Commutative", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCommutative(t, func(s *matchertest.Seat, combine func(a, b float64) float64, a, b float64,
			msg string, opts ...matcher.Option,
		) {
			matcher.Commutative(s, matcher.Fatal, combine, a, b, msg, opts...)
		})
	})

	t.Run("Associative", func(t *testing.T) {
		t.Parallel()
		matchertest.RunAssociative(t, func(s *matchertest.Seat, combine func(a, b float64) float64, a, b, c float64,
			msg string, opts ...matcher.Option,
		) {
			matcher.Associative(s, matcher.Fatal, combine, a, b, c, msg, opts...)
		})
	})

	t.Run("RoundTrip", func(t *testing.T) {
		t.Parallel()
		matchertest.RunRoundTrip(t, func(s *matchertest.Seat, forward func(float64) (string, error),
			inverse func(string) (float64, error), input float64, msg string, opts ...matcher.Option,
		) {
			matcher.RoundTrip(s, matcher.Fatal, forward, inverse, input, msg, opts...)
		})
	})

	t.Run("StableOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunStableOrder(t, func(s *matchertest.Seat, iterate func() ([]int, error), msg string,
			opts ...matcher.Option,
		) {
			matcher.StableOrder(s, matcher.Fatal, iterate, msg, opts...)
		})
	})

	t.Run("NoDuplicates", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNoDuplicates(t, func(s *matchertest.Seat, iterate func() ([]float64, error), msg string,
			opts ...matcher.Option,
		) {
			matcher.NoDuplicates(s, matcher.Fatal, iterate, msg, opts...)
		})
	})

	t.Run("Monotonic", func(t *testing.T) {
		t.Parallel()
		matchertest.RunMonotonic(t, func(s *matchertest.Seat, observe func() float64, advance func() error,
			steps int, msg string,
		) {
			matcher.Monotonic(s, matcher.Fatal, observe, advance, steps, msg)
		})
	})

	t.Run("Total", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTotal(t, func(s *matchertest.Seat, call func(int) error, domain []int, msg string) {
			matcher.Total(s, matcher.Fatal, call, domain, msg)
		})
	})

	t.Run("NotPure", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNotPure(t, func(s *matchertest.Seat, observe func() []int, fn func(), msg string,
			opts ...matcher.Option,
		) {
			matcher.NotPure(s, matcher.Fatal, observe, fn, msg, opts...)
		})
	})

	t.Run("FailsAfterClose", func(t *testing.T) {
		t.Parallel()
		matchertest.RunFailsAfterClose(t, func(s *matchertest.Seat, closer, call func() error, sentinel error,
			msg string,
		) {
			matcher.FailsAfterClose(s, matcher.Fatal, closer, call, sentinel, msg)
		})
	})

	t.Run("Poisoned", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPoisoned(t, func(s *matchertest.Seat, induce func(), observe func() error, msg string) {
			matcher.Poisoned(s, matcher.Fatal, induce, observe, msg)
		})
	})
}
