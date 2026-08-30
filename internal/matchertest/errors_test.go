// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("NoErrorCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NoErrorCases", matchertest.NoErrorCases(), 1)
	})

	t.Run("HasErrorCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "HasErrorCases", matchertest.HasErrorCases(), 1)
	})

	t.Run("ErrorIsCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "ErrorIsCases", matchertest.ErrorIsCases(), 2)
	})

	t.Run("ErrorIsNotCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "ErrorIsNotCases", matchertest.ErrorIsNotCases(), 2)
	})

	t.Run("AsError", func(t *testing.T) {
		t.Parallel()

		t.Run("reads a nil argument as a nil error", func(t *testing.T) {
			t.Parallel()

			if got := matchertest.AsError(nil); got != nil {
				t.Fatalf("AsError(nil) = %v, want nil", got)
			}
		})

		t.Run("reads an error argument as itself", func(t *testing.T) {
			t.Parallel()

			if got := matchertest.AsError(matchertest.ErrSample); !errors.Is(got, matchertest.ErrSample) {
				t.Fatalf("AsError = %v, want the error it was given", got)
			}
		})
	})

	t.Run("WrappedTyped", func(t *testing.T) {
		t.Parallel()

		t.Run("hides the typed error behind two wraps", func(t *testing.T) {
			t.Parallel()

			var target *matchertest.TypedError
			if !errors.As(matchertest.WrappedTyped(), &target) {
				t.Fatal("the typed error is not reachable through the chain")
			}
			if target.Field != matchertest.TypedField {
				t.Fatalf("Field = %q, want %q", target.Field, matchertest.TypedField)
			}
		})
	})
}
