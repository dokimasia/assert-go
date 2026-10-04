// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// option keeps the option that a call of an option's constructor returns.
var option expect.Option

// TestOption checks that an option relaxes the call it is passed to alone.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("EquateEmpty", func(t *testing.T) {
		t.Parallel()

		t.Run("applies to the call it is passed to", func(t *testing.T) {
			t.Parallel()

			relaxed, strict := &matchertest.Seat{}, &matchertest.Seat{}
			var nilSlice []int

			expect.Equal(relaxed, nilSlice, []int{}, "opted in", expect.EquateEmpty())
			expect.Equal(strict, nilSlice, []int{}, "not opted in")

			if relaxed.Failed() {
				t.Fatalf("the opted-in call reported %q", relaxed.First())
			}
			if !strict.Failed() {
				t.Fatal("the second call inherited the first call's option")
			}
		})
	})

	t.Run("EquateNaNs", func(t *testing.T) {
		t.Parallel()

		t.Run("applies to the call it is passed to", func(t *testing.T) {
			t.Parallel()

			relaxed, strict := &matchertest.Seat{}, &matchertest.Seat{}
			nan := math.NaN()

			expect.Equal(relaxed, nan, nan, "opted in", expect.EquateNaNs())
			expect.Equal(strict, nan, nan, "not opted in")

			if relaxed.Failed() {
				t.Fatalf("the opted-in call reported %q", relaxed.First())
			}
			if !strict.Failed() {
				t.Fatal("the second call inherited the first call's option")
			}
		})
	})
}

// TestOptionAllocs checks the allocation ceiling of each constructor of an
// option.
func TestOptionAllocs(t *testing.T) {
	alloctest.Check(t, optionCases())
}

// BenchmarkOption measures each constructor of an option.
func BenchmarkOption(b *testing.B) {
	for _, c := range optionCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// optionCases returns a call of each constructor of an option, with its
// allocation ceiling, measured.
func optionCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "EquateEmpty", Call: func(assert.TB) { option = expect.EquateEmpty() }},
		{Name: "EquateNaNs", Call: func(assert.TB) { option = expect.EquateNaNs() }},
	}
}
