// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// repetitions is how many times the definition calls the subject of
// deterministic, iterates the subject of stable-order and reads the
// subject of poisoned.
const repetitions = 32

// raisedValue is what every callable of the relation suites panics with.
const raisedValue = "matchertest: the callable panicked"

// errOther is a failure that no sentinel of the suites matches.
var errOther = errors.New("matchertest: another failure")

// raise panics with raisedValue.
func raise() { panic(raisedValue) }

// relationCase is one case of a relation suite. drive builds the case's
// callables and calls the assertion, and want is the outcome it must
// produce.
type relationCase struct {
	name  string
	drive func(seat *Seat)
	want  Case
}

// runRelation drives every case, and checks each outcome with outcome.
func runRelation(t *testing.T, cases []relationCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seat := &Seat{}
			tc.drive(seat)
			outcome(t, seat, tc.want)
		})
	}
}

// outcome fails t when the seat's outcome is not what want requires, or
// when the first record does not contain exactly the fields that want
// states. Every failing case of a relation states every field of its
// record, nil ones included.
func outcome(t *testing.T, seat *Seat, want Case) {
	t.Helper()

	checkOutcome(t, seat, want)
	if !want.Fails {
		return
	}
	fields := slices.Sorted(maps.Keys(seat.Records()[0].Detail))
	stated := slices.Sorted(maps.Keys(want.Detail))
	if !slices.Equal(fields, stated) {
		t.Fatalf("the record contains the fields %q, want %q", fields, stated)
	}
}

// failure returns the outcome of a failing case of assertion with detail.
func failure(assertion string, detail map[string]any) Case {
	return Case{Fails: true, Assertion: assertion, Detail: detail}
}

// goexits drives an assertion on a goroutine of its own with a callable
// that calls [runtime.Goexit], and fails t unless the goroutine ends
// inside the assertion and the assertion reports nothing.
func goexits(t *testing.T, drive func(seat *Seat, exit func())) {
	t.Helper()

	t.Run("a callable that ends its goroutine ends the assertion", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		returned := false
		ended := make(chan struct{})
		go func() {
			defer close(ended)
			drive(seat, runtime.Goexit)
			returned = true
		}()
		<-ended

		if returned {
			t.Fatal("the assertion returned after a callable ended its goroutine")
		}
		if seat.Failed() {
			t.Fatalf("reported %q for a callable that ended its goroutine", seat.First())
		}
	})
}

// counted returns a function that returns how many times it has been
// called, starting at 1.
func counted() func() int {
	calls := 0
	return func() int {
		calls++
		return calls
	}
}

// IdempotentInvoke calls a surface's idempotent assertion. The state is a
// slice of ints, whose nil and empty values a relaxation equates.
type IdempotentInvoke func(seat *Seat, call func(int) error, input int, observe func() []int, msg string,
	opts ...matcher.Option)

// RunIdempotent drives invoke against every case an idempotent assertion
// must produce.
func RunIdempotent(t *testing.T, invoke IdempotentInvoke) {
	t.Helper()

	// appends returns a call that appends its input to the state and fails
	// as fail states for each call, and an observe that reads a copy of the
	// state and panics as raises states for each reading.
	appends := func(fail func(call int) error, raises func(reading int) bool) (func(int) error, func() []int) {
		var state []int
		call, reading := counted(), counted()
		return func(v int) error {
				if err := fail(call()); err != nil {
					return err
				}
				state = append(state, v)
				return nil
			}, func() []int {
				if raises(reading()) {
					raise()
				}
				return slices.Clone(state)
			}
	}
	never := func(int) error { return nil }
	calm := func(int) bool { return false }

	// nilThenEmpty returns a call whose first call leaves the state nil and
	// whose second makes it empty, and an observe that reads it.
	nilThenEmpty := func() (func(int) error, func() []int) {
		var state []int
		calls := counted()
		return func(int) error {
				if calls() == 2 {
					state = []int{}
				}
				return nil
			}, func() []int {
				return state
			}
	}

	runRelation(t, []relationCase{
		{
			name: "a call that sets the state to its input passes",
			drive: func(seat *Seat) {
				var state []int
				invoke(seat, func(v int) error {
					state = []int{v}
					return nil
				}, 7, func() []int { return slices.Clone(state) }, contractMsg)
			},
		},
		{
			name: "a call that appends its input reports both readings",
			drive: func(seat *Seat) {
				call, observe := appends(func(int) error { return nil }, calm)
				invoke(seat, call, 7, observe, contractMsg)
			},
			want: failure("idempotent", map[string]any{"first": []int{7}, "second": []int{7, 7}}),
		},
		{
			name: "a nil reading against an empty one reports",
			drive: func(seat *Seat) {
				call, observe := nilThenEmpty()
				invoke(seat, call, 7, observe, contractMsg)
			},
			want: failure("idempotent", map[string]any{"first": []int(nil), "second": []int{}}),
		},
		{
			name: "a nil reading equals an empty one under EquateEmpty",
			drive: func(seat *Seat) {
				call, observe := nilThenEmpty()
				invoke(seat, call, 7, observe, contractMsg, matcher.EquateEmpty())
			},
		},
		{
			name: "an error of the first call reports it as first",
			drive: func(seat *Seat) {
				call, observe := appends(func(int) error { return ErrSample }, calm)
				invoke(seat, call, 7, observe, contractMsg)
			},
			want: failure("idempotent", map[string]any{"first": ErrSample, "second": nil}),
		},
		{
			name: "an error of the second call reports it as second",
			drive: func(seat *Seat) {
				call, observe := appends(func(call int) error {
					if call == 2 {
						return ErrSample
					}
					return nil
				}, calm)
				invoke(seat, call, 7, observe, contractMsg)
			},
			want: failure("idempotent", map[string]any{"first": nil, "second": ErrSample}),
		},
		{
			name: "a panic of the first reading reports it as first",
			drive: func(seat *Seat) {
				call, observe := appends(never, func(reading int) bool { return reading == 1 })
				invoke(seat, call, 7, observe, contractMsg)
			},
			want: failure("idempotent", map[string]any{"first": raisedValue, "second": nil}),
		},
		{
			name: "a panic of the second reading reports it as second",
			drive: func(seat *Seat) {
				call, observe := appends(never, func(reading int) bool { return reading == 2 })
				invoke(seat, call, 7, observe, contractMsg)
			},
			want: failure("idempotent", map[string]any{"first": nil, "second": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, never, 7, func() []int {
			exit()
			return nil
		}, contractMsg)
	})
}

// AccumulatesInvoke calls a surface's accumulating assertion.
type AccumulatesInvoke func(seat *Seat, call func(int) error, input int, observe func() int, msg string)

// RunAccumulates drives invoke against every case an accumulating
// assertion must produce.
func RunAccumulates(t *testing.T, invoke AccumulatesInvoke) {
	t.Helper()

	// reads returns an observe that reads each of readings in turn and
	// panics at a reading that raises states, and a call that fails as fail
	// states for each call.
	reads := func(readings []int, fail func(call int) error, raises func(reading int) bool) (func(int) error,
		func() int,
	) {
		call, reading := counted(), counted()
		return func(int) error {
				return fail(call())
			}, func() int {
				n := reading()
				if raises(n) {
					raise()
				}
				return readings[n-1]
			}
	}
	never := func(int) error { return nil }
	calm := func(int) bool { return false }

	// scripted drives invoke with an observe that reads readings in turn.
	scripted := func(readings ...int) func(seat *Seat) {
		return func(seat *Seat) {
			call, observe := reads(readings, never, calm)
			invoke(seat, call, 1, observe, contractMsg)
		}
	}
	errorAt := func(at int) func(int) error {
		return func(call int) error {
			if call == at {
				return ErrSample
			}
			return nil
		}
	}
	raiseAt := func(at int) func(int) bool {
		return func(reading int) bool { return reading == at }
	}

	runRelation(t, []relationCase{
		{
			name: "a count that each call raises by its input passes",
			drive: func(seat *Seat) {
				count := 0
				invoke(seat, func(v int) error {
					count += v
					return nil
				}, 3, func() int { return count }, contractMsg)
			},
		},
		{name: "a count that each call lowers by the same amount passes", drive: scripted(10, 7, 4)},
		{
			name:  "a first call that changes nothing reports both changes",
			drive: scripted(5, 5, 5),
			want:  failure("accumulates", map[string]any{"first": 0, "second": 0}),
		},
		{
			name:  "a call that sets a value reports both changes",
			drive: scripted(0, 7, 7),
			want:  failure("accumulates", map[string]any{"first": 7, "second": 0}),
		},
		{
			name:  "changes of different amounts report both",
			drive: scripted(0, 1, 3),
			want:  failure("accumulates", map[string]any{"first": 1, "second": 2}),
		},
		{
			name:  "a first change beyond the range of int reports it exactly",
			drive: scripted(math.MinInt, 5, 15),
			want: failure("accumulates", map[string]any{
				"first": new(big.Int).Sub(big.NewInt(5), big.NewInt(math.MinInt)), "second": 10,
			}),
		},
		{
			name:  "a second change beyond the range of int reports it exactly",
			drive: scripted(0, math.MaxInt, math.MinInt),
			want: failure("accumulates", map[string]any{
				"first": math.MaxInt, "second": new(big.Int).Sub(big.NewInt(math.MinInt), big.NewInt(math.MaxInt)),
			}),
		},
		{
			name: "a panic of the reading before the first call reports it as first",
			drive: func(seat *Seat) {
				call, observe := reads([]int{0, 1, 2}, never, raiseAt(1))
				invoke(seat, call, 1, observe, contractMsg)
			},
			want: failure("accumulates", map[string]any{"first": raisedValue, "second": nil}),
		},
		{
			name: "an error of the first call reports it as first",
			drive: func(seat *Seat) {
				call, observe := reads([]int{0, 1, 2}, errorAt(1), calm)
				invoke(seat, call, 1, observe, contractMsg)
			},
			want: failure("accumulates", map[string]any{"first": ErrSample, "second": nil}),
		},
		{
			name: "a panic of the reading after the first call reports it as first",
			drive: func(seat *Seat) {
				call, observe := reads([]int{0, 1, 2}, never, raiseAt(2))
				invoke(seat, call, 1, observe, contractMsg)
			},
			want: failure("accumulates", map[string]any{"first": raisedValue, "second": nil}),
		},
		{
			name: "an error of the second call reports it as second",
			drive: func(seat *Seat) {
				call, observe := reads([]int{0, 1, 2}, errorAt(2), calm)
				invoke(seat, call, 1, observe, contractMsg)
			},
			want: failure("accumulates", map[string]any{"first": nil, "second": ErrSample}),
		},
		{
			name: "a panic of the reading after the second call reports it as second",
			drive: func(seat *Seat) {
				call, observe := reads([]int{0, 1, 2}, never, raiseAt(3))
				invoke(seat, call, 1, observe, contractMsg)
			},
			want: failure("accumulates", map[string]any{"first": nil, "second": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func(int) error {
			exit()
			return nil
		}, 1, func() int { return 0 }, contractMsg)
	})
}

// DeterministicInvoke calls a surface's deterministic assertion. The
// result is a float64, whose NaN a relaxation equates with itself.
type DeterministicInvoke func(seat *Seat, call func(int) (float64, error), input int, msg string,
	opts ...matcher.Option)

// RunDeterministic drives invoke against every case a deterministic
// assertion must produce.
func RunDeterministic(t *testing.T, invoke DeterministicInvoke) {
	t.Helper()

	// changesAt returns a call that returns 0 before its call at and 1 from
	// then on.
	changesAt := func(at int) func(int) (float64, error) {
		calls := counted()
		return func(int) (float64, error) {
			if calls() < at {
				return 0, nil
			}
			return 1, nil
		}
	}
	// failsAt returns a call that returns its input, and fails as fail
	// states at its call at.
	failsAt := func(at int, fail func() error) func(int) (float64, error) {
		calls := counted()
		return func(v int) (float64, error) {
			if calls() == at {
				return 0, fail()
			}
			return float64(v), nil
		}
	}
	sample := func() error { return ErrSample }
	panics := func() error {
		raise()
		return nil
	}
	nan := func(int) (float64, error) { return math.NaN(), nil }

	runRelation(t, []relationCase{
		{
			name: "a call that returns its input passes",
			drive: func(seat *Seat) {
				invoke(seat, func(v int) (float64, error) { return float64(v), nil }, 7, contractMsg)
			},
		},
		{
			name: "a call that multiplies its input by its call count reports the first two results",
			drive: func(seat *Seat) {
				calls := counted()
				invoke(seat, func(v int) (float64, error) { return float64(v * calls()), nil }, 7, contractMsg)
			},
			want: failure("deterministic", map[string]any{"first": 7.0, "second": 14.0}),
		},
		{
			name:  "a result that changes at the 32nd call reports it",
			drive: func(seat *Seat) { invoke(seat, changesAt(repetitions), 7, contractMsg) },
			want:  failure("deterministic", map[string]any{"first": 0.0, "second": 1.0}),
		},
		{
			name:  "a result that changes at the 33rd call passes",
			drive: func(seat *Seat) { invoke(seat, changesAt(repetitions+1), 7, contractMsg) },
		},
		{
			name:  "NaN results report",
			drive: func(seat *Seat) { invoke(seat, nan, 7, contractMsg) },
			want:  failure("deterministic", map[string]any{"first": math.NaN(), "second": math.NaN()}),
		},
		{
			name:  "NaN results pass under EquateNaNs",
			drive: func(seat *Seat) { invoke(seat, nan, 7, contractMsg, matcher.EquateNaNs()) },
		},
		{
			name:  "an error of the first call reports it as first",
			drive: func(seat *Seat) { invoke(seat, failsAt(1, sample), 7, contractMsg) },
			want:  failure("deterministic", map[string]any{"first": ErrSample, "second": nil}),
		},
		{
			name:  "an error of a later call reports it as second",
			drive: func(seat *Seat) { invoke(seat, failsAt(5, sample), 7, contractMsg) },
			want:  failure("deterministic", map[string]any{"first": nil, "second": ErrSample}),
		},
		{
			name:  "a panic of the first call reports it as first",
			drive: func(seat *Seat) { invoke(seat, failsAt(1, panics), 7, contractMsg) },
			want:  failure("deterministic", map[string]any{"first": raisedValue, "second": nil}),
		},
		{
			name:  "a panic of a later call reports it as second",
			drive: func(seat *Seat) { invoke(seat, failsAt(5, panics), 7, contractMsg) },
			want:  failure("deterministic", map[string]any{"first": nil, "second": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func(int) (float64, error) {
			exit()
			return 0, nil
		}, 7, contractMsg)
	})
}

// CombineInvoke calls a surface's commutative assertion over float64s,
// whose NaN a relaxation equates with itself.
type CombineInvoke func(seat *Seat, combine func(a, b float64) float64, a, b float64, msg string,
	opts ...matcher.Option)

// RunCommutative drives invoke against every case a commutative assertion
// must produce. The cases combine 2 and 3.
func RunCommutative(t *testing.T, invoke CombineInvoke) {
	t.Helper()

	add := func(a, b float64) float64 { return a + b }
	nan := func(float64, float64) float64 { return math.NaN() }
	// raisesOn returns a subtraction that panics when its first operand is
	// a.
	raisesOn := func(a float64) func(x, y float64) float64 {
		return func(x, y float64) float64 {
			if x == a {
				raise()
			}
			return x - y
		}
	}

	runRelation(t, []relationCase{
		{name: "addition passes", drive: func(seat *Seat) { invoke(seat, add, 2, 3, contractMsg) }},
		{
			name: "subtraction reports both orders",
			drive: func(seat *Seat) {
				invoke(seat, func(a, b float64) float64 { return a - b }, 2, 3, contractMsg)
			},
			want: failure("commutative", map[string]any{"first": -1.0, "second": 1.0}),
		},
		{
			name:  "a NaN result reports",
			drive: func(seat *Seat) { invoke(seat, nan, 2, 3, contractMsg) },
			want:  failure("commutative", map[string]any{"first": math.NaN(), "second": math.NaN()}),
		},
		{
			name:  "a NaN result passes under EquateNaNs",
			drive: func(seat *Seat) { invoke(seat, nan, 2, 3, contractMsg, matcher.EquateNaNs()) },
		},
		{
			name:  "a panic of the first order reports it as first",
			drive: func(seat *Seat) { invoke(seat, raisesOn(2), 2, 3, contractMsg) },
			want:  failure("commutative", map[string]any{"first": raisedValue, "second": nil}),
		},
		{
			name:  "a panic of the second order reports it as second",
			drive: func(seat *Seat) { invoke(seat, raisesOn(3), 2, 3, contractMsg) },
			want:  failure("commutative", map[string]any{"first": nil, "second": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func(a, b float64) float64 {
			exit()
			return a
		}, 2, 3, contractMsg)
	})
}

// AssociativeInvoke calls a surface's associative assertion over float64s,
// whose NaN a relaxation equates with itself.
type AssociativeInvoke func(seat *Seat, combine func(a, b float64) float64, a, b, c float64, msg string,
	opts ...matcher.Option)

// RunAssociative drives invoke against every case an associative
// assertion must produce. The cases combine 2, 3 and 5.
func RunAssociative(t *testing.T, invoke AssociativeInvoke) {
	t.Helper()

	add := func(a, b float64) float64 { return a + b }
	nan := func(float64, float64) float64 { return math.NaN() }
	// raisesOn returns a subtraction that panics when its operands are x
	// and y.
	raisesOn := func(x, y float64) func(a, b float64) float64 {
		return func(a, b float64) float64 {
			if a == x && b == y {
				raise()
			}
			return a - b
		}
	}

	runRelation(t, []relationCase{
		{name: "addition passes", drive: func(seat *Seat) { invoke(seat, add, 2, 3, 5, contractMsg) }},
		{
			name: "subtraction reports both groupings",
			drive: func(seat *Seat) {
				invoke(seat, func(a, b float64) float64 { return a - b }, 2, 3, 5, contractMsg)
			},
			want: failure("associative", map[string]any{"first": -6.0, "second": 4.0}),
		},
		{
			name:  "a NaN result reports",
			drive: func(seat *Seat) { invoke(seat, nan, 2, 3, 5, contractMsg) },
			want:  failure("associative", map[string]any{"first": math.NaN(), "second": math.NaN()}),
		},
		{
			name:  "a NaN result passes under EquateNaNs",
			drive: func(seat *Seat) { invoke(seat, nan, 2, 3, 5, contractMsg, matcher.EquateNaNs()) },
		},
		{
			name:  "a panic in the left grouping reports it as first",
			drive: func(seat *Seat) { invoke(seat, raisesOn(2, 3), 2, 3, 5, contractMsg) },
			want:  failure("associative", map[string]any{"first": raisedValue, "second": nil}),
		},
		{
			name:  "a panic in the right grouping reports it as second",
			drive: func(seat *Seat) { invoke(seat, raisesOn(3, 5), 2, 3, 5, contractMsg) },
			want:  failure("associative", map[string]any{"first": nil, "second": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func(a, b float64) float64 {
			exit()
			return a
		}, 2, 3, 5, contractMsg)
	})
}

// RoundTripInvoke calls a surface's round-trip assertion over a float64
// rendered as decimal text, whose NaN a relaxation equates with itself.
type RoundTripInvoke func(seat *Seat, forward func(float64) (string, error), inverse func(string) (float64, error),
	input float64, msg string, opts ...matcher.Option)

// RunRoundTrip drives invoke against every case a round-trip assertion
// must produce.
func RunRoundTrip(t *testing.T, invoke RoundTripInvoke) {
	t.Helper()

	render := func(v float64) (string, error) { return strconv.FormatFloat(v, 'g', -1, 64), nil }
	parse := func(text string) (float64, error) { return strconv.ParseFloat(text, 64) }
	dropsSign := func(v float64) (string, error) {
		text, err := render(v)
		return strings.TrimPrefix(text, "-"), err
	}

	runRelation(t, []relationCase{
		{
			name:  "a decimal rendering passes",
			drive: func(seat *Seat) { invoke(seat, render, parse, -42.5, contractMsg) },
		},
		{
			name:  "a rendering that drops the sign reports the input and what came back",
			drive: func(seat *Seat) { invoke(seat, dropsSign, parse, -42.5, contractMsg) },
			want:  failure("round-trip", map[string]any{"want": -42.5, "got": 42.5}),
		},
		{
			name:  "a NaN reports",
			drive: func(seat *Seat) { invoke(seat, render, parse, math.NaN(), contractMsg) },
			want:  failure("round-trip", map[string]any{"want": math.NaN(), "got": math.NaN()}),
		},
		{
			name:  "a NaN passes under EquateNaNs",
			drive: func(seat *Seat) { invoke(seat, render, parse, math.NaN(), contractMsg, matcher.EquateNaNs()) },
		},
		{
			name: "an error of forward reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, func(float64) (string, error) { return "", ErrSample }, parse, -42.5, contractMsg)
			},
			want: failure("round-trip", map[string]any{"want": nil, "got": ErrSample}),
		},
		{
			name: "an error of inverse reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, render, func(string) (float64, error) { return 0, ErrSample }, -42.5, contractMsg)
			},
			want: failure("round-trip", map[string]any{"want": nil, "got": ErrSample}),
		},
		{
			name: "a panic of forward reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, func(float64) (string, error) {
					raise()
					return "", nil
				}, parse, -42.5, contractMsg)
			},
			want: failure("round-trip", map[string]any{"want": nil, "got": raisedValue}),
		},
		{
			name: "a panic of inverse reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, render, func(string) (float64, error) {
					raise()
					return 0, nil
				}, -42.5, contractMsg)
			},
			want: failure("round-trip", map[string]any{"want": nil, "got": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func(float64) (string, error) {
			exit()
			return "", nil
		}, parse, -42.5, contractMsg)
	})
}

// StableOrderInvoke calls a surface's stable-order assertion. A sequence
// is a slice of ints, whose nil and empty values a relaxation equates.
type StableOrderInvoke func(seat *Seat, iterate func() ([]int, error), msg string, opts ...matcher.Option)

// RunStableOrder drives invoke against every case a stable-order assertion
// must produce.
func RunStableOrder(t *testing.T, invoke StableOrderInvoke) {
	t.Helper()

	fixed := func() ([]int, error) { return []int{1, 2, 3}, nil }
	// rotatesAt returns an iterate that yields 1, 2 and 3 before its
	// iteration at and 2, 3 and 1 from then on.
	rotatesAt := func(at int) func() ([]int, error) {
		iterations := counted()
		return func() ([]int, error) {
			if iterations() < at {
				return []int{1, 2, 3}, nil
			}
			return []int{2, 3, 1}, nil
		}
	}
	// failsAt returns an iterate that yields 1, 2 and 3, and fails as fail
	// states at its iteration at.
	failsAt := func(at int, fail func() error) func() ([]int, error) {
		iterations := counted()
		return func() ([]int, error) {
			if iterations() == at {
				return nil, fail()
			}
			return []int{1, 2, 3}, nil
		}
	}
	sample := func() error { return ErrSample }
	panics := func() error {
		raise()
		return nil
	}
	// nilThenEmpty returns an iterate that yields nil once and an empty
	// sequence from then on.
	nilThenEmpty := func() func() ([]int, error) {
		iterations := counted()
		return func() ([]int, error) {
			if iterations() == 1 {
				return nil, nil
			}
			return []int{}, nil
		}
	}

	runRelation(t, []relationCase{
		{name: "a fixed order passes", drive: func(seat *Seat) { invoke(seat, fixed, contractMsg) }},
		{
			name:  "an order that changes at the second iteration reports the first two orders",
			drive: func(seat *Seat) { invoke(seat, rotatesAt(2), contractMsg) },
			want:  failure("stable-order", map[string]any{"first": []int{1, 2, 3}, "second": []int{2, 3, 1}}),
		},
		{
			name:  "an order that changes at the 32nd iteration reports it",
			drive: func(seat *Seat) { invoke(seat, rotatesAt(repetitions), contractMsg) },
			want:  failure("stable-order", map[string]any{"first": []int{1, 2, 3}, "second": []int{2, 3, 1}}),
		},
		{
			name:  "an order that changes at the 33rd iteration passes",
			drive: func(seat *Seat) { invoke(seat, rotatesAt(repetitions+1), contractMsg) },
		},
		{
			name:  "a nil sequence against an empty one reports",
			drive: func(seat *Seat) { invoke(seat, nilThenEmpty(), contractMsg) },
			want:  failure("stable-order", map[string]any{"first": []int(nil), "second": []int{}}),
		},
		{
			name:  "a nil sequence equals an empty one under EquateEmpty",
			drive: func(seat *Seat) { invoke(seat, nilThenEmpty(), contractMsg, matcher.EquateEmpty()) },
		},
		{
			name:  "an error of the first iteration reports it as first",
			drive: func(seat *Seat) { invoke(seat, failsAt(1, sample), contractMsg) },
			want:  failure("stable-order", map[string]any{"first": ErrSample, "second": nil}),
		},
		{
			name:  "an error of a later iteration reports it as second",
			drive: func(seat *Seat) { invoke(seat, failsAt(5, sample), contractMsg) },
			want:  failure("stable-order", map[string]any{"first": nil, "second": ErrSample}),
		},
		{
			name:  "a panic of the first iteration reports it as first",
			drive: func(seat *Seat) { invoke(seat, failsAt(1, panics), contractMsg) },
			want:  failure("stable-order", map[string]any{"first": raisedValue, "second": nil}),
		},
		{
			name:  "a panic of a later iteration reports it as second",
			drive: func(seat *Seat) { invoke(seat, failsAt(5, panics), contractMsg) },
			want:  failure("stable-order", map[string]any{"first": nil, "second": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func() ([]int, error) {
			exit()
			return nil, nil
		}, contractMsg)
	})
}

// NoDuplicatesInvoke calls a surface's no-duplicates assertion. An element
// is a float64, whose NaN a relaxation equates with itself.
type NoDuplicatesInvoke func(seat *Seat, iterate func() ([]float64, error), msg string, opts ...matcher.Option)

// RunNoDuplicates drives invoke against every case a no-duplicates
// assertion must produce.
func RunNoDuplicates(t *testing.T, invoke NoDuplicatesInvoke) {
	t.Helper()

	// yields returns an iterate that yields items.
	yields := func(items ...float64) func() ([]float64, error) {
		return func() ([]float64, error) { return items, nil }
	}
	nans := yields(math.NaN(), math.NaN())

	runRelation(t, []relationCase{
		{name: "distinct elements pass", drive: func(seat *Seat) { invoke(seat, yields(1, 2, 3), contractMsg) }},
		{name: "no elements pass", drive: func(seat *Seat) { invoke(seat, yields(), contractMsg) }},
		{
			name:  "a repeated element reports it and its position",
			drive: func(seat *Seat) { invoke(seat, yields(1, 2, 2, 3), contractMsg) },
			want:  failure("no-duplicates", map[string]any{"got": 2.0, "index": 2}),
		},
		{
			name:  "a repeat of the first element reports where it repeats",
			drive: func(seat *Seat) { invoke(seat, yields(1, 2, 3, 1), contractMsg) },
			want:  failure("no-duplicates", map[string]any{"got": 1.0, "index": 3}),
		},
		{name: "two NaNs are distinct", drive: func(seat *Seat) { invoke(seat, nans, contractMsg) }},
		{
			name:  "two NaNs repeat under EquateNaNs",
			drive: func(seat *Seat) { invoke(seat, nans, contractMsg, matcher.EquateNaNs()) },
			want:  failure("no-duplicates", map[string]any{"got": math.NaN(), "index": 1}),
		},
		{
			name: "an error of iterate reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, func() ([]float64, error) { return nil, ErrSample }, contractMsg)
			},
			want: failure("no-duplicates", map[string]any{"got": ErrSample, "index": nil}),
		},
		{
			name: "a panic of iterate reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, func() ([]float64, error) {
					raise()
					return nil, nil
				}, contractMsg)
			},
			want: failure("no-duplicates", map[string]any{"got": raisedValue, "index": nil}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func() ([]float64, error) {
			exit()
			return nil, nil
		}, contractMsg)
	})
}

// MonotonicInvoke calls a surface's monotonic assertion over float64
// readings.
type MonotonicInvoke func(seat *Seat, observe func() float64, advance func() error, steps int, msg string)

// RunMonotonic drives invoke against every case a monotonic assertion must
// produce.
func RunMonotonic(t *testing.T, invoke MonotonicInvoke) {
	t.Helper()

	// walk returns an observe that reads readings in turn, one further on
	// for each advance, and an advance that fails as fail states for each
	// advance. observe panics at the reading that raises states.
	walk := func(readings []float64, fail func(advance int) error, raises func(reading int) bool) (func() float64,
		func() error,
	) {
		at, reading, advances := 0, counted(), counted()
		return func() float64 {
				if raises(reading()) {
					raise()
				}
				return readings[at]
			}, func() error {
				if err := fail(advances()); err != nil {
					return err
				}
				at++
				return nil
			}
	}
	never := func(int) error { return nil }
	calm := func(int) bool { return false }
	// scripted drives invoke over readings, advancing len(readings)-1 times.
	scripted := func(readings ...float64) func(seat *Seat) {
		return func(seat *Seat) {
			observe, advance := walk(readings, never, calm)
			invoke(seat, observe, advance, len(readings)-1, contractMsg)
		}
	}
	nan := math.NaN()

	runRelation(t, []relationCase{
		{name: "a rising reading passes", drive: scripted(0, 1, 2, 3, 4, 5)},
		{name: "a reading that does not change passes", drive: scripted(3, 3, 3)},
		{
			name:  "a reading that wraps around reports where it fell",
			drive: scripted(0, 1, 2, 3, 0, 1),
			want:  failure("monotonic", map[string]any{"index": 4, "first": 3.0, "second": 0.0}),
		},
		{
			name:  "a fall at the last step reports it",
			drive: scripted(0, 1, 2, 1),
			want:  failure("monotonic", map[string]any{"index": 3, "first": 2.0, "second": 1.0}),
		},
		{
			name: "no steps read the subject once",
			drive: func(seat *Seat) {
				observe, advance := walk([]float64{5}, never, calm)
				invoke(seat, observe, advance, 0, contractMsg)
			},
		},
		{
			name: "fewer than no steps read the subject once",
			drive: func(seat *Seat) {
				observe, advance := walk([]float64{5}, never, calm)
				invoke(seat, observe, advance, -1, contractMsg)
			},
		},
		{
			name:  "a NaN first reading reports at position 0",
			drive: scripted(nan, 1),
			want:  failure("monotonic", map[string]any{"index": 0, "first": nil, "second": nan}),
		},
		{
			name:  "a NaN later reading reports",
			drive: scripted(0, nan),
			want:  failure("monotonic", map[string]any{"index": 1, "first": 0.0, "second": nan}),
		},
		{
			name: "an error of advance reports it as second",
			drive: func(seat *Seat) {
				observe, advance := walk([]float64{0, 1, 2}, func(advance int) error {
					if advance == 2 {
						return ErrSample
					}
					return nil
				}, calm)
				invoke(seat, observe, advance, 2, contractMsg)
			},
			want: failure("monotonic", map[string]any{"index": nil, "first": nil, "second": ErrSample}),
		},
		{
			name: "a panic of the first reading reports it as second",
			drive: func(seat *Seat) {
				observe, advance := walk([]float64{0, 1}, never, func(reading int) bool { return reading == 1 })
				invoke(seat, observe, advance, 1, contractMsg)
			},
			want: failure("monotonic", map[string]any{"index": nil, "first": nil, "second": raisedValue}),
		},
		{
			name: "a panic of a later reading reports it as second",
			drive: func(seat *Seat) {
				observe, advance := walk([]float64{0, 1}, never, func(reading int) bool { return reading == 2 })
				invoke(seat, observe, advance, 1, contractMsg)
			},
			want: failure("monotonic", map[string]any{"index": nil, "first": nil, "second": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func() float64 { return 0 }, func() error {
			exit()
			return nil
		}, 1, contractMsg)
	})
}

// TotalInvoke calls a surface's total assertion over a domain of ints.
type TotalInvoke func(seat *Seat, call func(int) error, domain []int, msg string)

// RunTotal drives invoke against every case a total assertion must
// produce.
func RunTotal(t *testing.T, invoke TotalInvoke) {
	t.Helper()

	// failsFor returns a call that fails as fail states for the element v,
	// and succeeds for every other element.
	failsFor := func(v int, fail func() error) func(int) error {
		return func(element int) error {
			if element == v {
				return fail()
			}
			return nil
		}
	}
	succeeds := func(int) error { return nil }

	runRelation(t, []relationCase{
		{
			name:  "a call that succeeds for every element passes",
			drive: func(seat *Seat) { invoke(seat, succeeds, []int{1, 2, 3}, contractMsg) },
		},
		{name: "an empty domain passes", drive: func(seat *Seat) { invoke(seat, succeeds, []int{}, contractMsg) }},
		{
			name: "a failure for the first element reports its index and the error",
			drive: func(seat *Seat) {
				invoke(seat, failsFor(1, func() error { return ErrSample }), []int{1, 2, 3}, contractMsg)
			},
			want: failure("total", map[string]any{"index": 0, "got": ErrSample}),
		},
		{
			name: "a failure for a later element reports its index and the error",
			drive: func(seat *Seat) {
				invoke(seat, failsFor(2, func() error { return ErrSample }), []int{1, 2, 3}, contractMsg)
			},
			want: failure("total", map[string]any{"index": 1, "got": ErrSample}),
		},
		{
			name: "a panic for an element reports its index and the value",
			drive: func(seat *Seat) {
				invoke(seat, failsFor(3, func() error {
					raise()
					return nil
				}), []int{1, 2, 3}, contractMsg)
			},
			want: failure("total", map[string]any{"index": 2, "got": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func(int) error {
			exit()
			return nil
		}, []int{1}, contractMsg)
	})
}

// NotPureInvoke calls a surface's not-pure assertion. The projection is a
// slice of ints, whose nil and empty values a relaxation equates.
type NotPureInvoke func(seat *Seat, observe func() []int, fn func(), msg string, opts ...matcher.Option)

// RunNotPure drives invoke against every case a not-pure assertion must
// produce.
func RunNotPure(t *testing.T, invoke NotPureInvoke) {
	t.Helper()

	// state returns an observe that reads a copy of the state, and an fn
	// that sets the state to after.
	state := func(before, after []int) (func() []int, func()) {
		held := before
		return func() []int { return slices.Clone(held) }, func() { held = after }
	}

	runRelation(t, []relationCase{
		{
			name: "a changed projection passes",
			drive: func(seat *Seat) {
				observe, fn := state([]int{1, 2}, []int{1, 2, 3})
				invoke(seat, observe, fn, contractMsg)
			},
		},
		{
			name: "an unchanged projection reports it",
			drive: func(seat *Seat) {
				observe, fn := state([]int{1, 2}, []int{1, 2})
				invoke(seat, observe, fn, contractMsg)
			},
			want: failure("not-pure", map[string]any{"got": []int{1, 2}}),
		},
		{
			name: "a change from nil to empty passes",
			drive: func(seat *Seat) {
				observe, fn := state(nil, []int{})
				invoke(seat, observe, fn, contractMsg)
			},
		},
		{
			name: "a change from nil to empty reports under EquateEmpty",
			drive: func(seat *Seat) {
				observe, fn := state(nil, []int{})
				invoke(seat, observe, fn, contractMsg, matcher.EquateEmpty())
			},
			want: failure("not-pure", map[string]any{"got": []int{}}),
		},
		{
			name: "a panic of observe reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, func() []int {
					raise()
					return nil
				}, func() {}, contractMsg)
			},
			want: failure("not-pure", map[string]any{"got": raisedValue}),
		},
		{
			name: "a panic of fn reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, func() []int { return nil }, raise, contractMsg)
			},
			want: failure("not-pure", map[string]any{"got": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func() []int { return nil }, exit, contractMsg)
	})
}

// FailsAfterCloseInvoke calls a surface's after-close assertion.
type FailsAfterCloseInvoke func(seat *Seat, closer, call func() error, sentinel error, msg string)

// RunFailsAfterClose drives invoke against every case an after-close
// assertion must produce. ErrSample is the sentinel.
func RunFailsAfterClose(t *testing.T, invoke FailsAfterCloseInvoke) {
	t.Helper()

	succeeds := func() error { return nil }
	// resource returns a closer that closes the resource and fails as
	// closing states, and a call that succeeds before the resource closes
	// and returns what closed states after.
	resource := func(closing, closed func() error) (func() error, func() error) {
		current := succeeds
		return func() error {
				if err := closing(); err != nil {
					return err
				}
				current = closed
				return nil
			}, func() error {
				return current()
			}
	}
	sample := func() error { return ErrSample }
	panics := func() error {
		raise()
		return nil
	}
	// drive drives invoke over a resource that closing and closed state.
	drive := func(closing, closed func() error) func(seat *Seat) {
		return func(seat *Seat) {
			closer, call := resource(closing, closed)
			invoke(seat, closer, call, ErrSample, contractMsg)
		}
	}

	runRelation(t, []relationCase{
		{name: "a call that fails with the sentinel after close passes", drive: drive(succeeds, sample)},
		{
			name: "a call that fails with the sentinel wrapped passes",
			drive: drive(succeeds, func() error {
				return fmt.Errorf("matchertest: closed: %w", ErrSample)
			}),
		},
		{
			name:  "a call that succeeds after close reports what it returned",
			drive: drive(succeeds, succeeds),
			want:  failure("after-close", map[string]any{"want": ErrSample, "got": nil}),
		},
		{
			name:  "a call that fails otherwise after close reports its error",
			drive: drive(succeeds, func() error { return errOther }),
			want:  failure("after-close", map[string]any{"want": ErrSample, "got": errOther}),
		},
		{
			name:  "an error of closer reports it as got",
			drive: drive(func() error { return errOther }, sample),
			want:  failure("after-close", map[string]any{"want": nil, "got": errOther}),
		},
		{
			name:  "a panic of closer reports it as got",
			drive: drive(panics, sample),
			want:  failure("after-close", map[string]any{"want": nil, "got": raisedValue}),
		},
		{
			name:  "a panic of call reports it as got",
			drive: drive(succeeds, panics),
			want:  failure("after-close", map[string]any{"want": nil, "got": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, func() error {
			exit()
			return nil
		}, succeeds, ErrSample, contractMsg)
	})
}

// PoisonedInvoke calls a surface's poisoned assertion.
type PoisonedInvoke func(seat *Seat, induce func(), observe func() error, msg string)

// RunPoisoned drives invoke against every case a poisoned assertion must
// produce.
func RunPoisoned(t *testing.T, invoke PoisonedInvoke) {
	t.Helper()

	// settlesAt returns an induce that poisons the subject, and an observe
	// that fails while the subject is poisoned and before its reading at.
	settlesAt := func(at int) (func(), func() error) {
		poisoned, readings := false, counted()
		return func() { poisoned = true }, func() error {
			if poisoned && readings() < at {
				return ErrSample
			}
			return nil
		}
	}
	// drive drives invoke over a subject that settles at its reading at.
	drive := func(at int) func(seat *Seat) {
		return func(seat *Seat) {
			induce, observe := settlesAt(at)
			invoke(seat, induce, observe, contractMsg)
		}
	}
	// unpoisoned returns an observe of a subject that nothing poisons, for a
	// case whose induce does not return.
	unpoisoned := func() func() error {
		_, observe := settlesAt(repetitions + 1)
		return observe
	}

	runRelation(t, []relationCase{
		{name: "a subject that fails 32 readings after induce passes", drive: drive(repetitions + 1)},
		{
			name:  "a subject that succeeds at the 32nd reading reports it",
			drive: drive(repetitions),
			want:  failure("poisoned", map[string]any{"index": repetitions - 1, "got": nil}),
		},
		{
			name:  "a subject that succeeds at the third reading reports it",
			drive: drive(3),
			want:  failure("poisoned", map[string]any{"index": 2, "got": nil}),
		},
		{
			name:  "a panic of induce reports it as got",
			drive: func(seat *Seat) { invoke(seat, raise, unpoisoned(), contractMsg) },
			want:  failure("poisoned", map[string]any{"index": nil, "got": raisedValue}),
		},
		{
			name: "a panic of a reading reports it as got",
			drive: func(seat *Seat) {
				invoke(seat, func() {}, func() error {
					raise()
					return nil
				}, contractMsg)
			},
			want: failure("poisoned", map[string]any{"index": nil, "got": raisedValue}),
		},
	})

	goexits(t, func(seat *Seat, exit func()) {
		invoke(seat, exit, unpoisoned(), contractMsg)
	})
}
