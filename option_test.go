// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// option keeps the option that a call of an option's constructor returns.
var option assert.Option

// equateByAllocs is the ceiling of the allocations of EquateBy: its transform
// and the option.
const equateByAllocs = 3

// stamp is a value with a count of reads of its own, which stampID leaves
// out.
type stamp struct {
	ID    int
	reads int
}

// stampID returns the ID of s.
func stampID(s stamp) int { return s.ID }

// TestOption checks that an option changes the call it is passed to alone.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("EquateEmpty", func(t *testing.T) {
		t.Parallel()

		t.Run("applies to the call it is passed to", func(t *testing.T) {
			t.Parallel()

			relaxed, strict := &matchertest.Seat{}, &matchertest.Seat{}
			var nilSlice []int

			assert.Equal(relaxed, nilSlice, []int{}, "opted in", assert.EquateEmpty())
			assert.Equal(strict, nilSlice, []int{}, "not opted in")

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

			assert.Equal(relaxed, nan, nan, "opted in", assert.EquateNaNs())
			assert.Equal(strict, nan, nan, "not opted in")

			if relaxed.Failed() {
				t.Fatalf("the opted-in call reported %q", relaxed.First())
			}
			if !strict.Failed() {
				t.Fatal("the second call inherited the first call's option")
			}
		})
	})

	t.Run("ByIdentity", func(t *testing.T) {
		t.Parallel()

		t.Run("applies to the call it is passed to", func(t *testing.T) {
			t.Parallel()

			narrowed, structural := &matchertest.Seat{}, &matchertest.Seat{}
			first, second := new(1), new(1)

			assert.Equal(narrowed, first, second, "opted in", assert.ByIdentity())
			assert.Equal(structural, first, second, "not opted in")

			if !narrowed.Failed() {
				t.Fatal("two allocations of one value are the same object under the option")
			}
			if structural.Failed() {
				t.Fatalf("the second call inherited the first call's option: %q", structural.First())
			}
		})
	})

	t.Run("EquateBy", func(t *testing.T) {
		t.Parallel()

		t.Run("applies to the call it is passed to", func(t *testing.T) {
			t.Parallel()

			transformed, structural := &matchertest.Seat{}, &matchertest.Seat{}
			first, second := stamp{ID: 1, reads: 5}, stamp{ID: 1, reads: 9}

			assert.Equal(transformed, first, second, "opted in", assert.EquateBy(stampID))
			assert.Equal(structural, first, second, "not opted in")

			if transformed.Failed() {
				t.Fatalf("the opted-in call reported %q", transformed.First())
			}
			if !structural.Failed() {
				t.Fatal("the second call inherited the first call's option")
			}
		})

		t.Run("panics for a nil function", func(t *testing.T) {
			t.Parallel()

			got := assert.Panics(t, func() { assert.EquateBy[stamp, int](nil) }, "no transform")
			assert.Equal(t, got, any("assert: EquateBy(nil) states no transform"), "the panic names the option")
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
// allocation ceiling.
func optionCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "EquateEmpty", Call: func(assert.TB) { option = assert.EquateEmpty() }},
		{Name: "EquateNaNs", Call: func(assert.TB) { option = assert.EquateNaNs() }},
		{Name: "ByIdentity", Call: func(assert.TB) { option = assert.ByIdentity() }},
		{Name: "EquateBy", Call: func(assert.TB) { option = assert.EquateBy(stampID) }, Allocs: equateByAllocs},
	}
}
