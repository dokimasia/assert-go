// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"fmt"
	"testing"

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
		matchertest.RunOne(t, matchertest.NoErrorCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.NoError(s, matcher.Fatal, matchertest.AsError(got), msg)
			})
	})

	t.Run("HasError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.HasErrorCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.HasError(s, matcher.Fatal, matchertest.AsError(got), msg)
			})
	})

	t.Run("ErrorIs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				matcher.ErrorIs(s, matcher.Fatal, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorIsNot", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsNotCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				matcher.ErrorIsNot(s, matcher.Fatal, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorAs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunErrorAs(t, func(s *matchertest.Seat, err error, msg string) *matchertest.TypedError {
			return matcher.ErrorAs[*matchertest.TypedError](s, matcher.Fatal, err, msg)
		})
	})
}

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
