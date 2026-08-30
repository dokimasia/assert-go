// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The package's whole claim is that one seat serves every surface.
// These prove it at compile time: the seat is declared without
// importing either surface, and satisfies both by method set alone.
var (
	_ matcher.Seat = (*matchertest.Seat)(nil)
	_ assert.TB    = (*matchertest.Seat)(nil)
)

// concurrentCalls is enough goroutines to surface an unguarded append
// under -race, and few enough not to slow the suite.
const concurrentCalls = 8

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

		t.Run("returns so a test can read what was reported", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("first")
			s.Fatalf("second")

			if got, want := len(s.Fatals()), 2; got != want {
				t.Fatalf("len(Fatals()) = %d, want %d; this seat must not stop", got, want)
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

	t.Run("Failed", func(t *testing.T) {
		t.Parallel()

		t.Run("false on a fresh seat", func(t *testing.T) {
			t.Parallel()

			if (&matchertest.Seat{}).Failed() {
				t.Fatal("a fresh seat reports failed")
			}
		})

		t.Run("true after either path", func(t *testing.T) {
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

		t.Run("empty on a fresh seat", func(t *testing.T) {
			t.Parallel()

			if got := (&matchertest.Seat{}).First(); got != "" {
				t.Fatalf("First() = %q, want empty", got)
			}
		})

		t.Run("prefers the aborting path", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Errorf("soft")
			s.Fatalf("fatal")

			if got, want := s.First(), "fatal"; got != want {
				t.Fatalf("First() = %q, want %q", got, want)
			}
		})

		t.Run("falls back to the recording path", func(t *testing.T) {
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

		t.Run("returns a copy the caller owns", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("original")

			got := s.Fatals()
			got[0] = "mutated"

			if s.Fatals()[0] != "original" {
				t.Fatal("mutating the result changed the seat; the copy is shared")
			}
		})
	})

	t.Run("concurrency", func(t *testing.T) {
		t.Parallel()

		t.Run("survives calls from several goroutines", func(t *testing.T) {
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
