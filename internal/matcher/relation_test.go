// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"strconv"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
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

// TestRelationAllocs checks the allocation ceiling of a passing call of
// each relation assertion.
func TestRelationAllocs(t *testing.T) {
	checkAllocs(t, relationCases())
}

// BenchmarkRelation measures a passing call of each relation assertion.
func BenchmarkRelation(b *testing.B) {
	benchAllocs(b, relationCases())
}

// relationCases returns a passing call of each relation assertion over
// ints, with its allocation ceiling, measured.
func relationCases() []allocCase {
	var state int
	set := func(x int) error { state = x; return nil }   //nolint:unparam // the assertion's signature
	increment := func(int) error { state++; return nil } //nolint:unparam // the assertion's signature
	read := func() int { return state }
	advance := func() error { state++; return nil } //nolint:unparam // the assertion's signature
	double := func(x int) (int, error) { return 2 * x, nil }
	add := func(a, b int) int { return a + b }
	format := func(x int) (string, error) { return strconv.Itoa(x), nil }
	items := []int{1, 2, 3}
	listed := func() ([]int, error) { return items, nil } //nolint:unparam // the assertion's signature
	accepts := func(int) error { return nil }
	closes := func() error { return nil }
	refuses := func() error { return errClosed }
	return []allocCase{
		{name: "Idempotent", call: func(seat matcher.Seat) {
			matcher.Idempotent(seat, matcher.Fatal, set, 1, read, allocContract)
		}},
		{name: "Accumulates", call: func(seat matcher.Seat) {
			matcher.Accumulates(seat, matcher.Fatal, increment, 1, read, allocContract)
		}},
		{name: "Deterministic", call: func(seat matcher.Seat) {
			matcher.Deterministic(seat, matcher.Fatal, double, 3, allocContract)
		}},
		{name: "Commutative", call: func(seat matcher.Seat) {
			matcher.Commutative(seat, matcher.Fatal, add, 2, 3, allocContract)
		}},
		{name: "Associative", call: func(seat matcher.Seat) {
			matcher.Associative(seat, matcher.Fatal, add, 1, 2, 3, allocContract)
		}},
		{name: "RoundTrip", call: func(seat matcher.Seat) {
			matcher.RoundTrip(seat, matcher.Fatal, format, strconv.Atoi, 42, allocContract)
		}},
		{name: "StableOrder", allocs: 62, call: func(seat matcher.Seat) {
			matcher.StableOrder(seat, matcher.Fatal, listed, allocContract)
		}},
		{name: "NoDuplicates", allocs: 1, call: func(seat matcher.Seat) {
			matcher.NoDuplicates(seat, matcher.Fatal, listed, allocContract)
		}},
		{name: "Monotonic", call: func(seat matcher.Seat) {
			matcher.Monotonic(seat, matcher.Fatal, read, advance, 3, allocContract)
		}},
		{name: "Total", call: func(seat matcher.Seat) {
			matcher.Total(seat, matcher.Fatal, accepts, items, allocContract)
		}},
		{name: "NotPure", allocs: 2, call: func(seat matcher.Seat) {
			matcher.NotPure(seat, matcher.Fatal, read, func() { state++ }, allocContract)
		}},
		{name: "FailsAfterClose", call: func(seat matcher.Seat) {
			matcher.FailsAfterClose(seat, matcher.Fatal, closes, refuses, errClosed, allocContract)
		}},
		{name: "Poisoned", call: func(seat matcher.Seat) {
			matcher.Poisoned(seat, matcher.Fatal, func() {}, refuses, allocContract)
		}},
	}
}
