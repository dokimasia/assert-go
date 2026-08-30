// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The public seat satisfies this one by method set alone, which is
// what lets neither package import the other for its interface.
var _ matcher.Seat = assert.TB(nil)

func TestSeat(t *testing.T) {
	t.Parallel()

	t.Run("Report", func(t *testing.T) {
		t.Parallel()

		t.Run("Fatal reports through Fatalf", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			matcher.Report(s, matcher.Fatal, "boom %d", 1)

			if got, want := len(s.Fatals()), 1; got != want {
				t.Fatalf("len(Fatals()) = %d, want %d", got, want)
			}
			if len(s.Errs()) != 0 {
				t.Fatalf("Errs() = %v, want none", s.Errs())
			}
		})

		t.Run("Soft reports through Errorf", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			matcher.Report(s, matcher.Soft, "boom")

			if got, want := len(s.Errs()), 1; got != want {
				t.Fatalf("len(Errs()) = %d, want %d", got, want)
			}
			if len(s.Fatals()) != 0 {
				t.Fatalf("Fatals() = %v, want none", s.Fatals())
			}
		})

		t.Run("formats its arguments", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			matcher.Report(s, matcher.Fatal, "%s has %d", "list", 2)

			if got, want := s.First(), "list has 2"; got != want {
				t.Fatalf("First() = %q, want %q", got, want)
			}
		})

		t.Run("marks the calling frame as a helper", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			matcher.Report(s, matcher.Fatal, "boom")

			if s.HelperCalls() == 0 {
				t.Fatal("Report did not mark its frame as a helper")
			}
		})
	})

	t.Run("Mode", func(t *testing.T) {
		t.Parallel()

		t.Run("the zero value is Fatal", func(t *testing.T) {
			t.Parallel()

			var zero matcher.Mode
			if zero != matcher.Fatal {
				t.Fatalf("the zero Mode is %v, want Fatal", zero)
			}
		})
	})
}
