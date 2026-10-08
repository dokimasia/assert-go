// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"errors"
	"math/big"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The twins in this file implement each relation independently of the
// package under test, and drive it through its suite. A suite that no
// implementation passes looks like coverage and checks nothing.

// guarded calls fn, and returns its value and the error that it returned
// or the value that it panicked with.
func guarded[T any](fn func() (T, error)) (value T, failure any) {
	defer func() {
		if raised := recover(); raised != nil {
			failure = raised
		}
	}()
	value, err := fn()
	if err != nil {
		return value, err
	}
	return value, nil
}

// compared reports the first of two steps that fails, or both values when
// they differ.
func compared[T any](s *matchertest.Seat, assertion, msg string, first, second func() (T, error),
	opts []matcher.Option,
) {
	a, failure := guarded(first)
	if failure != nil {
		report(s, assertion, msg, map[string]any{"first": failure, "second": nil})
		return
	}
	b, failure := guarded(second)
	if failure != nil {
		report(s, assertion, msg, map[string]any{"first": nil, "second": failure})
		return
	}
	if !same(a, b, opts) {
		report(s, assertion, msg, map[string]any{"first": a, "second": b})
	}
}

// repeated runs run 32 times, and reports the first run that fails or
// differs from the first.
func repeated[T any](s *matchertest.Seat, assertion, msg string, run func() (T, error), opts []matcher.Option) {
	first, failure := guarded(run)
	if failure != nil {
		report(s, assertion, msg, map[string]any{"first": failure, "second": nil})
		return
	}
	for range 31 {
		next, failure := guarded(run)
		if failure != nil {
			report(s, assertion, msg, map[string]any{"first": nil, "second": failure})
			return
		}
		if !same(first, next, opts) {
			report(s, assertion, msg, map[string]any{"first": first, "second": next})
			return
		}
	}
}

// delta returns to minus from: an int when it fits one, and a *big.Int
// otherwise.
func delta(from, to int) any {
	d := new(big.Int).Sub(big.NewInt(int64(to)), big.NewInt(int64(from)))
	if d.IsInt64() && int64(int(d.Int64())) == d.Int64() {
		return int(d.Int64())
	}
	return d
}

func TestRelation(t *testing.T) {
	t.Parallel()

	t.Run("RunIdempotent", func(t *testing.T) {
		t.Parallel()
		matchertest.RunIdempotent(t, func(s *matchertest.Seat, call func(int) error, input int,
			observe func() []int, msg string, opts ...matcher.Option,
		) {
			step := func() ([]int, error) {
				if err := call(input); err != nil {
					return nil, err
				}
				return observe(), nil
			}
			compared(s, "idempotent", msg, step, step, opts)
		})
	})

	t.Run("RunAccumulates", func(t *testing.T) {
		t.Parallel()
		matchertest.RunAccumulates(t, func(s *matchertest.Seat, call func(int) error, input int,
			observe func() int, msg string,
		) {
			var readings [3]int
			first, failure := guarded(func() (any, error) {
				readings[0] = observe()
				if err := call(input); err != nil {
					return nil, err
				}
				readings[1] = observe()
				return delta(readings[0], readings[1]), nil
			})
			if failure != nil {
				report(s, "accumulates", msg, map[string]any{"first": failure, "second": nil})
				return
			}
			second, failure := guarded(func() (any, error) {
				if err := call(input); err != nil {
					return nil, err
				}
				readings[2] = observe()
				return delta(readings[1], readings[2]), nil
			})
			if failure != nil {
				report(s, "accumulates", msg, map[string]any{"first": nil, "second": failure})
				return
			}
			equal := first == second
			if large, ok := first.(*big.Int); ok {
				other, _ := second.(*big.Int)
				equal = other != nil && large.Cmp(other) == 0
			}
			if first == 0 || !equal {
				report(s, "accumulates", msg, map[string]any{"first": first, "second": second})
			}
		})
	})

	t.Run("RunDeterministic", func(t *testing.T) {
		t.Parallel()
		matchertest.RunDeterministic(t, func(s *matchertest.Seat, call func(int) (float64, error), input int,
			msg string, opts ...matcher.Option,
		) {
			repeated(s, "deterministic", msg, func() (float64, error) { return call(input) }, opts)
		})
	})

	t.Run("RunCommutative", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCommutative(t, func(s *matchertest.Seat, combine func(a, b float64) float64, a, b float64,
			msg string, opts ...matcher.Option,
		) {
			compared(s, "commutative", msg,
				func() (float64, error) { return combine(a, b), nil },
				func() (float64, error) { return combine(b, a), nil }, opts)
		})
	})

	t.Run("RunAssociative", func(t *testing.T) {
		t.Parallel()
		matchertest.RunAssociative(t, func(s *matchertest.Seat, combine func(a, b float64) float64, a, b, c float64,
			msg string, opts ...matcher.Option,
		) {
			compared(s, "associative", msg,
				func() (float64, error) { return combine(combine(a, b), c), nil },
				func() (float64, error) { return combine(a, combine(b, c)), nil }, opts)
		})
	})

	t.Run("RunRoundTrip", func(t *testing.T) {
		t.Parallel()
		matchertest.RunRoundTrip(t, func(s *matchertest.Seat, forward func(float64) (string, error),
			inverse func(string) (float64, error), input float64, msg string, opts ...matcher.Option,
		) {
			got, failure := guarded(func() (float64, error) {
				text, err := forward(input)
				if err != nil {
					return 0, err
				}
				return inverse(text)
			})
			switch {
			case failure != nil:
				report(s, "round-trip", msg, map[string]any{"want": nil, "got": failure})
			case !same(input, got, opts):
				report(s, "round-trip", msg, map[string]any{"want": input, "got": got})
			}
		})
	})

	t.Run("RunStableOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunStableOrder(t, func(s *matchertest.Seat, iterate func() ([]int, error), msg string,
			opts ...matcher.Option,
		) {
			repeated(s, "stable-order", msg, iterate, opts)
		})
	})

	t.Run("RunNoDuplicates", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNoDuplicates(t, func(s *matchertest.Seat, iterate func() ([]float64, error), msg string,
			opts ...matcher.Option,
		) {
			items, failure := guarded(iterate)
			if failure != nil {
				report(s, "no-duplicates", msg, map[string]any{"got": failure, "index": nil})
				return
			}
			for i := 1; i < len(items); i++ {
				for j := range i {
					if same(items[i], items[j], opts) {
						report(s, "no-duplicates", msg, map[string]any{"got": items[i], "index": i})
						return
					}
				}
			}
		})
	})

	t.Run("RunMonotonic", func(t *testing.T) {
		t.Parallel()
		matchertest.RunMonotonic(t, func(s *matchertest.Seat, observe func() float64, advance func() error,
			steps int, msg string,
		) {
			broke := func(failure any) {
				report(s, "monotonic", msg, map[string]any{"index": nil, "first": nil, "second": failure})
			}
			readings := []float64{}
			for step := 0; step <= max(steps, 0); step++ {
				reading, failure := guarded(func() (float64, error) {
					if step > 0 {
						if err := advance(); err != nil {
							return 0, err
						}
					}
					return observe(), nil
				})
				if failure != nil {
					broke(failure)
					return
				}
				readings = append(readings, reading)
				var first any
				if step > 0 {
					first = readings[step-1]
				}
				if reading != reading || (step > 0 && reading < readings[step-1]) {
					report(s, "monotonic", msg, map[string]any{"index": step, "first": first, "second": reading})
					return
				}
			}
		})
	})

	t.Run("RunTotal", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTotal(t, func(s *matchertest.Seat, call func(int) error, domain []int, msg string) {
			for i, element := range domain {
				if _, failure := guarded(func() (bool, error) { return true, call(element) }); failure != nil {
					report(s, "total", msg, map[string]any{"index": i, "got": failure})
					return
				}
			}
		})
	})

	t.Run("RunNotPure", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNotPure(t, func(s *matchertest.Seat, observe func() []int, fn func(), msg string,
			opts ...matcher.Option,
		) {
			readings, failure := guarded(func() ([2][]int, error) {
				before := observe()
				fn()
				return [2][]int{before, observe()}, nil
			})
			switch {
			case failure != nil:
				report(s, "not-pure", msg, map[string]any{"got": failure})
			case same(readings[0], readings[1], opts):
				report(s, "not-pure", msg, map[string]any{"got": readings[1]})
			}
		})
	})

	t.Run("RunFailsAfterClose", func(t *testing.T) {
		t.Parallel()
		matchertest.RunFailsAfterClose(t, func(s *matchertest.Seat, closer, call func() error, sentinel error,
			msg string,
		) {
			got, failure := guarded(func() (error, error) {
				if err := closer(); err != nil {
					return nil, err
				}
				return call(), nil
			})
			switch {
			case failure != nil:
				report(s, "after-close", msg, map[string]any{"want": nil, "got": failure})
			case !errors.Is(got, sentinel):
				report(s, "after-close", msg, map[string]any{"want": sentinel, "got": got})
			}
		})
	})

	t.Run("RunPoisoned", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPoisoned(t, func(s *matchertest.Seat, induce func(), observe func() error, msg string) {
			if _, failure := guarded(func() (bool, error) {
				induce()
				return true, nil
			}); failure != nil {
				report(s, "poisoned", msg, map[string]any{"index": nil, "got": failure})
				return
			}
			for i := range 32 {
				err, failure := guarded(func() (error, error) { return observe(), nil })
				if failure != nil {
					report(s, "poisoned", msg, map[string]any{"index": nil, "got": failure})
					return
				}
				if err == nil {
					report(s, "poisoned", msg, map[string]any{"index": i, "got": nil})
					return
				}
			}
		})
	})
}

// TestRelationTwins runs TestRelationTwinsProcess in a child process, and
// requires the failures of RunPoisoned for twins that end the goroutine of
// induce in two ways.
func TestRelationTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestRelationTwinsProcess",
		"the assertion returned after a callable ended its goroutine",
		"for a callable that ended its goroutine")
}

// TestRelationTwinsProcess runs only in the child process of
// TestRelationTwins.
func TestRelationTwinsProcess(t *testing.T) {
	inChild(t)

	t.Run("RunPoisoned of a twin that induces on a goroutine of its own", func(t *testing.T) {
		matchertest.RunPoisoned(t, func(_ *matchertest.Seat, induce func(), _ func() error, _ string) {
			ended := make(chan struct{})
			go func() {
				defer close(ended)
				defer func() { _ = recover() }()
				induce()
			}()
			<-ended
		})
	})

	t.Run("RunPoisoned of a twin that reports as its goroutine ends", func(t *testing.T) {
		matchertest.RunPoisoned(t, func(s *matchertest.Seat, induce func(), _ func() error, msg string) {
			defer func() { _ = recover() }()
			defer report(s, "poisoned", msg, map[string]any{"index": nil, "got": nil})
			induce()
		})
	})
}
