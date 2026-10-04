// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"errors"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The package declares the seat without importing assert or expect. These
// assertions check at compile time that the seat satisfies matcher.Seat,
// assert.TB and matcher.FaultReporter by its method set alone.
var (
	_ matcher.Seat          = (*matchertest.Seat)(nil)
	_ assert.TB             = (*matchertest.Seat)(nil)
	_ matcher.FaultReporter = (*matchertest.Seat)(nil)
)

// concurrentCalls is the number of goroutines of the concurrency case:
// enough for -race to report an unguarded append, and few enough to keep
// the suite fast.
const concurrentCalls = 8

// TestSeat checks what the seat records through each of its methods.
func TestSeat(t *testing.T) {
	t.Parallel()

	t.Run("Fatalf", func(t *testing.T) {
		t.Parallel()

		t.Run("records the formatted message", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("value %d and %s", 7, "text")

			if got, want := s.Fatals(), "value 7 and text"; len(got) != 1 || got[0] != want {
				t.Fatalf("Fatals() = %v, want [%q]", got, want)
			}
		})

		t.Run("returns, so a test reads what the seat recorded", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("first")
			s.Fatalf("second")

			if got, want := len(s.Fatals()), 2; got != want {
				t.Fatalf("len(Fatals()) = %d, want %d: the seat returns from Fatalf", got, want)
			}
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		t.Run("records every message in order", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Errorf("one")
			s.Errorf("two")

			got := s.Errs()
			if len(got) != 2 || got[0] != "one" || got[1] != "two" {
				t.Fatalf("Errs() = %v, want [one two]", got)
			}
		})
	})

	t.Run("Report", func(t *testing.T) {
		t.Parallel()

		t.Run("records an aborting failure, and fails the seat through Fatalf", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			f := matcher.Failure{Assertion: "true", Contract: "the flag is set"}
			s.Report(f, true)

			if got := s.Records(); len(got) != 1 || got[0].Contract != f.Contract {
				t.Fatalf("Records() = %+v, want [%+v]", got, f)
			}
			if got, want := s.Fatals(), matcher.Render(f); len(got) != 1 || got[0] != want || len(s.Errs()) != 0 {
				t.Fatalf("Fatals() = %q and Errs() = %q, want [%q] and none", got, s.Errs(), want)
			}
		})

		t.Run("records a failure that does not abort, and fails the seat through Errorf", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			f := matcher.Failure{Assertion: "true", Contract: "the flag is set"}
			s.Report(f, false)

			if got := s.Records(); len(got) != 1 || got[0].Contract != f.Contract {
				t.Fatalf("Records() = %+v, want [%+v]", got, f)
			}
			if got, want := s.Errs(), matcher.Render(f); len(got) != 1 || got[0] != want || len(s.Fatals()) != 0 {
				t.Fatalf("Errs() = %q and Fatals() = %q, want [%q] and none", got, s.Fatals(), want)
			}
		})
	})

	t.Run("Records", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{Assertion: "true", Contract: "the flag is set"}, false)

			got := s.Records()
			got[0].Contract = "changed"

			if s.Records()[0].Contract != "the flag is set" {
				t.Fatal("changing the result changed the seat, so the copy is shared")
			}
		})
	})

	t.Run("ReportFault", func(t *testing.T) {
		t.Parallel()

		t.Run("records a fault that ends its call, and fails the seat through Fatalf", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			err := fault.New("the seed is no number")
			s.ReportFault(err, true)

			if got := s.Faults(); len(got) != 1 || !errors.Is(got[0], err) {
				t.Fatalf("Faults() = %v, want [%v]", got, err)
			}
			if got, want := s.Fatals(), matcher.RenderFault(err); len(got) != 1 || got[0] != want {
				t.Fatalf("Fatals() = %q, want [%q]", got, want)
			}
		})

		t.Run("records a noted fault, and fails nothing", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			err := fault.New("the stored case decodes to other values")
			s.ReportFault(err, false)

			if got := s.Faults(); len(got) != 1 || !errors.Is(got[0], err) {
				t.Fatalf("Faults() = %v, want [%v]", got, err)
			}
			if s.Failed() {
				t.Fatalf("reported %q for a noted fault", s.First())
			}
		})
	})

	t.Run("Faults", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			err := fault.New("the seed is no number")
			s.ReportFault(err, false)

			got := s.Faults()
			got[0] = nil

			if !errors.Is(s.Faults()[0], err) {
				t.Fatal("changing the result changed the seat, so the copy is shared")
			}
		})
	})

	t.Run("Failed", func(t *testing.T) {
		t.Parallel()

		t.Run("reports false for a fresh seat", func(t *testing.T) {
			t.Parallel()

			if (&matchertest.Seat{}).Failed() {
				t.Fatal("a fresh seat reports failed")
			}
		})

		t.Run("reports true after a failure through either method", func(t *testing.T) {
			t.Parallel()

			fatal, soft := &matchertest.Seat{}, &matchertest.Seat{}
			fatal.Fatalf("x")
			soft.Errorf("x")

			if !fatal.Failed() || !soft.Failed() {
				t.Fatalf("Failed() = %v and %v, want both true", fatal.Failed(), soft.Failed())
			}
		})
	})

	t.Run("First", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the empty string for a fresh seat", func(t *testing.T) {
			t.Parallel()

			if got := (&matchertest.Seat{}).First(); got != "" {
				t.Fatalf("First() = %q, want empty", got)
			}
		})

		t.Run("returns the first message of Fatalf before any of Errorf", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Errorf("soft")
			s.Fatalf("fatal")

			if got, want := s.First(), "fatal"; got != want {
				t.Fatalf("First() = %q, want %q", got, want)
			}
		})

		t.Run("returns the first message of Errorf when Fatalf received none", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Errorf("soft")

			if got, want := s.First(), "soft"; got != want {
				t.Fatalf("First() = %q, want %q", got, want)
			}
		})
	})

	t.Run("HelperCalls", func(t *testing.T) {
		t.Parallel()

		t.Run("counts each call", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Helper()
			s.Helper()

			if got, want := s.HelperCalls(), 2; got != want {
				t.Fatalf("HelperCalls() = %d, want %d", got, want)
			}
		})
	})

	t.Run("Fatals", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("original")

			got := s.Fatals()
			got[0] = "mutated"

			if s.Fatals()[0] != "original" {
				t.Fatal("changing the result changed the seat, so the copy is shared")
			}
		})
	})

	t.Run("concurrency", func(t *testing.T) {
		t.Parallel()

		t.Run("records the calls of concurrent goroutines", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}

			var wg sync.WaitGroup
			wg.Add(concurrentCalls)
			for i := range concurrentCalls {
				go func() {
					defer wg.Done()
					s.Helper()
					s.Errorf("from %d", i)
				}()
			}
			wg.Wait()

			if got, want := len(s.Errs()), concurrentCalls; got != want {
				t.Fatalf("len(Errs()) = %d, want %d", got, want)
			}
			if got, want := s.HelperCalls(), concurrentCalls; got != want {
				t.Fatalf("HelperCalls() = %d, want %d", got, want)
			}
		})
	})
}
