// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"fmt"
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// typed keeps the error that a call of ErrorAs returns.
var typed *matchertest.TypedError

// TestErrors runs the shared cases of the error assertions.
func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("NoError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NoErrorOfAnyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.NoError(s, matcher.Fatal, got, msg)
			})

		t.Run("writes the record of a passing call", func(t *testing.T) {
			t.Parallel()
			checkPassRecord(t, "err-absent", func(seat matcher.Seat) {
				matcher.NoError(seat, matcher.Fatal, nil, allocContract)
			})
		})
	})

	t.Run("HasError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.HasErrorOfAnyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.HasError(s, matcher.Fatal, got, msg)
			})

		t.Run("writes the record of a passing call", func(t *testing.T) {
			t.Parallel()
			checkPassRecord(t, "err-present", func(seat matcher.Seat) {
				matcher.HasError(seat, matcher.Fatal, matchertest.ErrSample, allocContract)
			})
		})
	})

	t.Run("ErrorIs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsOfAnyCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				matcher.ErrorIs(s, matcher.Fatal, got, matchertest.AsError(target), msg)
			})

		t.Run("writes the record of a passing call", func(t *testing.T) {
			t.Parallel()
			checkPassRecord(t, "err-is", func(seat matcher.Seat) {
				matcher.ErrorIs(seat, matcher.Fatal, matchertest.ErrSample, matchertest.ErrSample, allocContract)
			})
		})
	})

	t.Run("ErrorIsNot", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsNotOfAnyCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				matcher.ErrorIsNot(s, matcher.Fatal, got, matchertest.AsError(target), msg)
			})

		t.Run("writes the record of a passing call", func(t *testing.T) {
			t.Parallel()
			checkPassRecord(t, "err-is-not", func(seat matcher.Seat) {
				matcher.ErrorIsNot(seat, matcher.Fatal, matchertest.ErrSample, matchertest.ErrOther, allocContract)
			})
		})
	})

	t.Run("ErrorAs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunErrorAs(t, func(s *matchertest.Seat, err error, msg string) *matchertest.TypedError {
			return matcher.ErrorAs[*matchertest.TypedError](s, matcher.Fatal, err, msg)
		})

		t.Run("writes the record of a passing call", func(t *testing.T) {
			t.Parallel()
			wrapped := matchertest.WrappedTyped()
			checkPassRecord(t, "err-as", func(seat matcher.Seat) {
				_ = matcher.ErrorAs[*matchertest.TypedError](seat, matcher.Fatal, wrapped, allocContract)
			})
		})

		t.Run("returns an error of an interface type that does not embed error", func(t *testing.T) {
			t.Parallel()
			seat := &matchertest.Seat{}
			got := matcher.ErrorAs[interface{ Timeout() bool }](seat, matcher.Fatal,
				fmt.Errorf("outer: %w", timeoutError{}), allocContract)
			if seat.Failed() || len(seat.Faults()) != 0 {
				t.Fatalf(
					"reported %q and the faults %v for a chain that has the interface",
					seat.First(),
					seat.Faults(),
				)
			}
			if got == nil || !got.Timeout() {
				t.Fatalf("returned %v, want the error of the chain", got)
			}
		})

		t.Run("ends the call with a fault for a type that is no interface and does not implement error",
			func(t *testing.T) {
				t.Parallel()
				seat := &matchertest.Seat{}
				got := matcher.ErrorAs[int](seat, matcher.Fatal, matchertest.ErrSample, allocContract)
				faults := seat.Faults()
				if len(faults) != 1 || len(seat.Records()) != 0 {
					t.Fatalf("reported the faults %v and the records %v, want one fault", faults, seat.Records())
				}
				var f *fault.Error
				const reason = "the type int is no interface and does not implement error"
				if !errors.As(faults[0], &f) || f.Op != "" || f.Kind != nil || f.Reason != reason {
					t.Fatalf("reported %#v, want the fault %q", faults[0], reason)
				}
				if got != 0 {
					t.Fatalf("returned %d, want the zero value", got)
				}
			})
	})
}

// timeoutError is an error that reports a timeout, as a network error does.
type timeoutError struct{}

// Error returns the error's text.
func (timeoutError) Error() string { return "matcher_test: the call timed out" }

// Timeout reports that the error is a timeout.
func (timeoutError) Timeout() bool { return true }

// TestErrorsAllocs checks the allocation ceiling of a passing call of each
// error assertion.
func TestErrorsAllocs(t *testing.T) {
	checkAllocs(t, errorsCases())
}

// BenchmarkErrors measures a passing call of each error assertion.
func BenchmarkErrors(b *testing.B) {
	benchAllocs(b, errorsCases())
}

// errorsCases returns a passing call of each error assertion, with its
// allocation ceiling, measured: on a sentinel wrapped twice where the
// assertion reads a chain.
func errorsCases() []allocCase {
	wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", matchertest.ErrSample))
	wrappedTyped := matchertest.WrappedTyped()
	return []allocCase{
		{name: "NoError", call: func(seat matcher.Seat) { matcher.NoError(seat, matcher.Fatal, nil, allocContract) }},
		{name: "HasError", call: func(seat matcher.Seat) {
			matcher.HasError(seat, matcher.Fatal, wrapped, allocContract)
		}},
		{name: "ErrorIs", call: func(seat matcher.Seat) {
			matcher.ErrorIs(seat, matcher.Fatal, wrapped, matchertest.ErrSample, allocContract)
		}},
		{name: "ErrorIsNot", call: func(seat matcher.Seat) {
			matcher.ErrorIsNot(seat, matcher.Fatal, wrapped, matchertest.ErrOther, allocContract)
		}},
		{name: "ErrorAs", allocs: 1, call: func(seat matcher.Seat) {
			typed = matcher.ErrorAs[*matchertest.TypedError](seat, matcher.Fatal, wrappedTyped, allocContract)
		}},
	}
}
