// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"errors"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// errClosed is the error of a subject's call after it closed.
var errClosed = errors.New("the subject is closed")

// TestRelation runs the shared cases of the relation assertions.
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

// TestRelationAllocs checks the allocation ceiling of a passing call of
// each relation assertion.
func TestRelationAllocs(t *testing.T) {
	alloctest.Check(t, relationCases())
}

// BenchmarkRelation measures a passing call of each relation assertion.
func BenchmarkRelation(b *testing.B) {
	for _, c := range relationCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// relationCases returns a passing call of each relation assertion over
// ints, with its allocation ceiling.
func relationCases() []alloctest.Case {
	var state int
	set := func(x int) error { state = x; return nil }
	increment := func(int) error { state++; return nil }
	read := func() int { return state }
	advance := func() error { state++; return nil }
	double := func(x int) (int, error) { return 2 * x, nil }
	add := func(a, b int) int { return a + b }
	format := func(x int) (string, error) { return strconv.Itoa(x), nil }
	items := []int{1, 2, 3}
	listed := func() ([]int, error) { return items, nil }
	accepts := func(int) error { return nil }
	closes := func() error { return nil }
	refuses := func() error { return errClosed }
	return []alloctest.Case{
		{Name: "Idempotent", Call: func(tb assert.TB) { assert.Idempotent(tb, set, 1, read, allocContract) }},
		{Name: "Accumulates", Call: func(tb assert.TB) { assert.Accumulates(tb, increment, 1, read, allocContract) }},
		{Name: "Deterministic", Call: func(tb assert.TB) { assert.Deterministic(tb, double, 3, allocContract) }},
		{Name: "Commutative", Call: func(tb assert.TB) { assert.Commutative(tb, add, 2, 3, allocContract) }},
		{Name: "Associative", Call: func(tb assert.TB) { assert.Associative(tb, add, 1, 2, 3, allocContract) }},
		{Name: "RoundTrip", Call: func(tb assert.TB) {
			assert.RoundTrip(tb, format, strconv.Atoi, 42, allocContract)
		}},
		{Name: "StableOrder", Call: func(tb assert.TB) { assert.StableOrder(tb, listed, allocContract) }, Allocs: 78},
		{Name: "NoDuplicates", Call: func(tb assert.TB) { assert.NoDuplicates(tb, listed, allocContract) }, Allocs: 2},
		{Name: "Monotonic", Call: func(tb assert.TB) { assert.Monotonic(tb, read, advance, 3, allocContract) }},
		{Name: "Total", Call: func(tb assert.TB) { assert.Total(tb, accepts, items, allocContract) }},
		{Name: "NotPure", Allocs: 3, Call: func(tb assert.TB) {
			assert.NotPure(tb, read, func() { state++ }, allocContract)
		}},
		{Name: "FailsAfterClose", Call: func(tb assert.TB) {
			assert.FailsAfterClose(tb, closes, refuses, errClosed, allocContract)
		}},
		{Name: "Poisoned", Call: func(tb assert.TB) { assert.Poisoned(tb, func() {}, refuses, allocContract) }},
	}
}
