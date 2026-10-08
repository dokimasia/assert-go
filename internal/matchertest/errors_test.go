// Copyright Dokimasia B.V. 2026
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

	// Each table of an assertion over any value is the table of the
	// function, and failing cases of a value that is no error.
	tests := []struct {
		name      string
		give      []matchertest.Case
		giveBase  []matchertest.Case
		giveArity int
	}{
		{"NoErrorOfAnyCases", matchertest.NoErrorOfAnyCases(), matchertest.NoErrorCases(), 1},
		{"HasErrorOfAnyCases", matchertest.HasErrorOfAnyCases(), matchertest.HasErrorCases(), 1},
		{"ErrorIsOfAnyCases", matchertest.ErrorIsOfAnyCases(), matchertest.ErrorIsCases(), 2},
		{"ErrorIsNotOfAnyCases", matchertest.ErrorIsNotOfAnyCases(), matchertest.ErrorIsNotCases(), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			checkTable(t, tt.name, tt.give, tt.giveArity)

			for i, base := range tt.giveBase {
				if got := tt.give[i].Name; got != base.Name {
					t.Fatalf("%s states %q as its case %d, want %q", tt.name, got, i, base.Name)
				}
			}
			added := tt.give[len(tt.giveBase):]
			if len(added) == 0 {
				t.Fatalf("%s adds no case to the %d of the function", tt.name, len(tt.giveBase))
			}
			for _, tc := range added {
				if _, isError := tc.Args[0].(error); isError || tc.Args[0] == nil || !tc.Fails {
					t.Fatalf("%s adds %+v, want a failing case of a value that is no error", tt.name, tc)
				}
			}
		})
	}

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

// TestErrorsTwins runs TestErrorsTwinsProcess in a child process, and
// requires the failures of RunErrorAs for twins that return nil and a zero
// error.
func TestErrorsTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestErrorsTwinsProcess",
		"returned matchertest: typed error, want the error from the chain",
		"returned nil for an error already of the target type",
		"returned matchertest: typed error on failure, want the zero value")
}

// TestErrorsTwinsProcess runs only in the child process of TestErrorsTwins.
func TestErrorsTwinsProcess(t *testing.T) {
	inChild(t)

	t.Run("RunErrorAs of a twin that returns nil", func(t *testing.T) {
		matchertest.RunErrorAs(t, func(*matchertest.Seat, error, string) *matchertest.TypedError { return nil })
	})

	t.Run("RunErrorAs of a twin that returns a zero error", func(t *testing.T) {
		matchertest.RunErrorAs(t, func(*matchertest.Seat, error, string) *matchertest.TypedError {
			return &matchertest.TypedError{}
		})
	})
}
