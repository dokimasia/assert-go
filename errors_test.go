// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
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
				assert.NoError(s, matchertest.AsError(got), msg)
			})
	})

	t.Run("HasError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.HasErrorCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.HasError(s, matchertest.AsError(got), msg)
			})
	})

	t.Run("ErrorIs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				assert.ErrorIs(s, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorIsNot", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsNotCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				assert.ErrorIsNot(s, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorAs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunErrorAs(t, func(s *matchertest.Seat, err error, msg string) *matchertest.TypedError {
			return assert.ErrorAs[*matchertest.TypedError](s, err, msg)
		})
	})
}

// TestErrorsAllocs checks the allocation ceiling of a passing call of each
// error assertion.
func TestErrorsAllocs(t *testing.T) {
	alloctest.Check(t, errorsCases())
}

// BenchmarkErrors measures a passing call of each error assertion.
func BenchmarkErrors(b *testing.B) {
	for _, c := range errorsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// errorsCases returns a passing call of each error assertion, with its
// allocation ceiling, measured: on a sentinel wrapped twice where the
// assertion reads a chain.
func errorsCases() []alloctest.Case {
	wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", matchertest.ErrSample))
	wrappedTyped := matchertest.WrappedTyped()
	return []alloctest.Case{
		{Name: "NoError", Call: func(tb assert.TB) { assert.NoError(tb, nil, allocContract) }},
		{Name: "HasError", Call: func(tb assert.TB) { assert.HasError(tb, wrapped, allocContract) }},
		{
			Name: "ErrorIs",
			Call: func(tb assert.TB) { assert.ErrorIs(tb, wrapped, matchertest.ErrSample, allocContract) },
		},
		{Name: "ErrorIsNot", Call: func(tb assert.TB) {
			assert.ErrorIsNot(tb, wrapped, matchertest.ErrOther, allocContract)
		}},
		{Name: "ErrorAs", Allocs: 1, Call: func(tb assert.TB) {
			typed = assert.ErrorAs[*matchertest.TypedError](tb, wrappedTyped, allocContract)
		}},
	}
}
