// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// seat records what a matcher reported, so a test reads the failure
// instead of suffering it. It stands in for the whole package's tests,
// which is why it lives beside the interface it implements.
type seat struct {
	fatals []string
	errs   []string
}

func (*seat) Helper() {}

func (s *seat) Fatalf(format string, args ...any) {
	s.fatals = append(s.fatals, fmt.Sprintf(format, args...))
}

func (s *seat) Errorf(format string, args ...any) {
	s.errs = append(s.errs, fmt.Sprintf(format, args...))
}

func TestSeat(t *testing.T) {
	t.Parallel()

	t.Run("Report", func(t *testing.T) {
		t.Parallel()

		t.Run("Fatal reports through Fatalf", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.Report(s, matcher.Fatal, "boom %d", 1)

			if got, want := len(s.fatals), 1; got != want {
				t.Fatalf("len(fatals) = %d, want %d", got, want)
			}
			if len(s.errs) != 0 {
				t.Fatalf("errs = %v, want none", s.errs)
			}
		})

		t.Run("Soft reports through Errorf", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.Report(s, matcher.Soft, "boom")

			if got, want := len(s.errs), 1; got != want {
				t.Fatalf("len(errs) = %d, want %d", got, want)
			}
			if len(s.fatals) != 0 {
				t.Fatalf("fatals = %v, want none", s.fatals)
			}
		})

		t.Run("formats its arguments", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.Report(s, matcher.Fatal, "%s has %d", "list", 2)

			if got, want := s.fatals[0], "list has 2"; got != want {
				t.Fatalf("fatals[0] = %q, want %q", got, want)
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
