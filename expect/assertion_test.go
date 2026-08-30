// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The chain is a third surface over the same comparisons, so it runs
// the same suites the functions do. A method that drifted from its
// function fails the shared case rather than passing a case of its
// own.
func TestAssertion(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.EqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				expect.That(s, got).Equal(want, msg)
			})
	})

	t.Run("NotEqual", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotEqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				expect.That(s, got).NotEqual(want, msg)
			})
	})

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.That(s, got).Nil(msg)
			})
	})

	t.Run("NotNil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotNilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.That(s, got).NotNil(msg)
			})
	})

	t.Run("Length", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.LengthCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				expect.That(s, got).Length(want.(int), msg)
			})
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.EmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.That(s, got).Empty(msg)
			})
	})

	t.Run("NotEmpty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotEmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.That(s, got).NotEmpty(msg)
			})
	})

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				expect.That(s, haystack).Contains(needle, msg)
			})
	})

	t.Run("NotContains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				expect.That(s, haystack).NotContains(needle, msg)
			})
	})

	t.Run("ContainsInOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsInOrderCases(),
			func(s *matchertest.Seat, got, needles any, msg string) {
				expect.That(s, got).ContainsInOrder(needles.([]string), msg)
			})
	})

	t.Run("HasPrefix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasPrefixCases(),
			func(s *matchertest.Seat, got, prefix any, msg string) {
				expect.That(s, got).HasPrefix(prefix.(string), msg)
			})
	})

	t.Run("HasSuffix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasSuffixCases(),
			func(s *matchertest.Seat, got, suffix any, msg string) {
				expect.That(s, got).HasSuffix(suffix.(string), msg)
			})
	})

	t.Run("Matches", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.MatchesCases(),
			func(s *matchertest.Seat, got, pattern any, msg string) {
				expect.That(s, got).Matches(pattern.(string), msg)
			})
	})

	t.Run("CloseTo", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.CloseToCases(),
			func(s *matchertest.Seat, got, want, tolerance any, msg string) {
				expect.That(s, got).CloseTo(want.(float64), tolerance.(float64), msg)
			})
	})

	t.Run("InRange", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.InRangeCases(),
			func(s *matchertest.Seat, got, low, high any, msg string) {
				expect.That(s, got).InRange(low.(float64), high.(float64), msg)
			})
	})

	// What follows belongs to the chain alone: the shared cases say
	// what one assertion reports, and these say how several compose.
	t.Run("That", func(t *testing.T) {
		t.Parallel()

		t.Run("marks the calling frame as a helper", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			expect.That(s, 1)

			if s.HelperCalls() == 0 {
				t.Fatal("That did not mark its frame as a helper")
			}
		})

		t.Run("a method returns the receiver so calls chain", func(t *testing.T) {
			t.Parallel()

			a := expect.That(&matchertest.Seat{}, 1)
			if a.Equal(1, "is one") != a {
				t.Fatal("Equal did not return the receiver")
			}
		})

		t.Run("the first failure is the one reported", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			expect.That(s, 1).Equal(2, "first failure").Equal(3, "second failure")

			if !strings.HasPrefix(s.First(), "first failure") {
				t.Fatalf("First() = %q, want it to lead with the first failure", s.First())
			}
		})

		t.Run("methods of different families mix in one chain", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			expect.That(s, "store: missing").
				HasPrefix("store: ", "carries the package").
				Contains("missing", "says what happened").
				NotEqual("", "is not empty")

			if s.Failed() {
				t.Fatalf("reported %q, want none", s.First())
			}
		})

		t.Run("an option reaches the method it is passed to", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			var nilSlice []int
			expect.That(s, nilSlice).Equal([]int{}, "opted in", expect.EquateEmpty())

			if s.Failed() {
				t.Fatalf("reported %q under EquateEmpty", s.First())
			}
		})
	})
}
