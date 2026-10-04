// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestContains runs the shared cases of the containment assertions.
func TestContains(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				assert.Contains(s, haystack, needle, msg)
			})
	})

	t.Run("NotContains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				assert.NotContains(s, haystack, needle, msg)
			})
	})

	t.Run("ContainsInOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsInOrderCases(),
			func(s *matchertest.Seat, got, needles any, msg string) {
				assert.ContainsInOrder(s, got, needles.([]string), msg)
			})
	})

	t.Run("Permutation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPermutation(t, func(s *matchertest.Seat, got, want []any, msg string, opts ...assert.Option) {
			assert.Permutation(s, got, want, msg, opts...)
		})
	})
}

// TestContainsAllocs checks the allocation ceiling of a passing call of
// each containment assertion.
func TestContainsAllocs(t *testing.T) {
	alloctest.Check(t, containsCases())
}

// BenchmarkContains measures a passing call of each containment assertion.
func BenchmarkContains(b *testing.B) {
	for _, c := range containsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// containsCases returns a passing call of each containment assertion, with
// its allocation ceiling, measured: text for the assertions over text, and
// slices of three ints for Permutation. Contains also runs on a slice whose
// last element is the needle, and ContainsInOrder on bytes.
func containsCases() []alloctest.Case {
	needles := []string{"cart", "items"}
	got, want := []int{1, 2, 3}, []int{3, 1, 2}
	bytes := []byte("a cart of three items")
	return []alloctest.Case{
		{Name: "Contains", Call: func(tb assert.TB) {
			assert.Contains(tb, "a cart of three items", "cart", allocContract)
		}},
		{
			Name:   "Contains of a slice",
			Call:   func(tb assert.TB) { assert.Contains(tb, got, 3, allocContract) },
			Allocs: 76,
		},
		{Name: "NotContains", Call: func(tb assert.TB) {
			assert.NotContains(tb, "a cart of three items", "truck", allocContract)
		}},
		{Name: "ContainsInOrder", Call: func(tb assert.TB) {
			assert.ContainsInOrder(tb, "a cart of three items", needles, allocContract)
		}},
		{Name: "ContainsInOrder of bytes", Allocs: 2, Call: func(tb assert.TB) {
			assert.ContainsInOrder(tb, bytes, needles, allocContract)
		}},
		{
			Name:   "Permutation",
			Call:   func(tb assert.TB) { assert.Permutation(tb, got, want, allocContract) },
			Allocs: 120,
		},
	}
}
