// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

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
}
