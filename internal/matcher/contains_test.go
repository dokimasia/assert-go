// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestContains runs the shared cases of the containment assertions.
func TestContains(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				matcher.Contains(s, matcher.Fatal, haystack, needle, msg)
			})
	})

	t.Run("NotContains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				matcher.NotContains(s, matcher.Fatal, haystack, needle, msg)
			})
	})

	t.Run("ContainsInOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsInOrderCases(),
			func(s *matchertest.Seat, got, needles any, msg string) {
				matcher.ContainsInOrder(s, matcher.Fatal, got, needles.([]string), msg)
			})
	})

	t.Run("Permutation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPermutation(t, func(s *matchertest.Seat, got, want []any, msg string, opts ...matcher.Option) {
			matcher.Permutation(s, matcher.Fatal, got, want, msg, opts...)
		})
	})
}

// TestContainsAllocs checks the allocation ceiling of a passing call of
// each containment assertion.
func TestContainsAllocs(t *testing.T) {
	checkAllocs(t, containsCases())
}

// BenchmarkContains measures a passing call of each containment assertion.
func BenchmarkContains(b *testing.B) {
	benchAllocs(b, containsCases())
}

// containsCases returns a passing call of each containment assertion, with
// its allocation ceiling: text for the assertions over text, and slices of
// three ints for Permutation. Contains also runs on a slice whose last
// element is the needle, and ContainsInOrder on bytes.
func containsCases() []allocCase {
	needles := []string{"cart", "items"}
	got, want := []int{1, 2, 3}, []int{3, 1, 2}
	bytes := []byte("a cart of three items")
	return []allocCase{
		{name: "Contains", call: func(seat matcher.Seat) {
			matcher.Contains(seat, matcher.Fatal, "a cart of three items", "cart", allocContract)
		}},
		{name: "Contains of a slice", allocs: 2, call: func(seat matcher.Seat) {
			matcher.Contains(seat, matcher.Fatal, got, 3, allocContract)
		}},
		{name: "ContainsInOrder of bytes", allocs: 3, call: func(seat matcher.Seat) {
			matcher.ContainsInOrder(seat, matcher.Fatal, bytes, needles, allocContract)
		}},
		{name: "NotContains", call: func(seat matcher.Seat) {
			matcher.NotContains(seat, matcher.Fatal, "a cart of three items", "truck", allocContract)
		}},
		{name: "ContainsInOrder", call: func(seat matcher.Seat) {
			matcher.ContainsInOrder(seat, matcher.Fatal, "a cart of three items", needles, allocContract)
		}},
		{name: "Permutation", allocs: 3, call: func(seat matcher.Seat) {
			matcher.Permutation(seat, matcher.Fatal, got, want, allocContract)
		}},
	}
}
