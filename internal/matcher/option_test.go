// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.dokimi.dev/assert/internal/matcher"
)

func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("Options", func(t *testing.T) {
		t.Parallel()

		t.Run("a nil collection does not equal an empty one", func(t *testing.T) {
			t.Parallel()

			var nilSlice []int
			if cmp.Equal(nilSlice, []int{}, matcher.Options()...) {
				t.Fatal("a nil slice equals an empty slice, want unequal")
			}

			var nilMap map[string]int
			if cmp.Equal(nilMap, map[string]int{}, matcher.Options()...) {
				t.Fatal("a nil map equals an empty map, want unequal")
			}
		})

		t.Run("unexported fields take part", func(t *testing.T) {
			t.Parallel()

			type hidden struct{ n int }

			if cmp.Equal(hidden{n: 1}, hidden{n: 2}, matcher.Options()...) {
				t.Fatal("structs differing in an unexported field compared equal")
			}
			if !cmp.Equal(hidden{n: 1}, hidden{n: 1}, matcher.Options()...) {
				t.Fatal("identical structs with an unexported field compared unequal")
			}
		})

		t.Run("a function equals itself", func(t *testing.T) {
			t.Parallel()

			f := func() {}
			if !cmp.Equal(f, f, matcher.Options()...) {
				t.Fatal("a function compared against itself is unequal")
			}
		})

		t.Run("two functions are unequal", func(t *testing.T) {
			t.Parallel()

			f, g := func() {}, func() {}
			if cmp.Equal(f, g, matcher.Options()...) {
				t.Fatal("two distinct functions compared equal")
			}
		})

		t.Run("NaN does not equal NaN", func(t *testing.T) {
			t.Parallel()

			nan := math.NaN()
			if cmp.Equal(nan, nan, matcher.Options()...) {
				t.Fatal("NaN equals NaN, want unequal")
			}
		})

		t.Run("floats compare exactly", func(t *testing.T) {
			t.Parallel()

			// Variables, not constants: untyped constant arithmetic is
			// exact at compile time and would test nothing.
			tenth, fifth := 0.1, 0.2
			if cmp.Equal(tenth+fifth, 0.3, matcher.Options()...) {
				t.Fatal("0.1+0.2 equals 0.3; the comparison applied a tolerance")
			}
		})

		t.Run("returns a slice the caller owns", func(t *testing.T) {
			t.Parallel()

			first := matcher.Options()
			second := matcher.Options()
			first[0] = nil

			if second[0] == nil {
				t.Fatal("two calls share backing memory; the caller does not own the result")
			}
		})
	})

	t.Run("EquateEmpty", func(t *testing.T) {
		t.Parallel()

		t.Run("makes a nil collection equal an empty one", func(t *testing.T) {
			t.Parallel()

			var nilSlice []int
			if !cmp.Equal(nilSlice, []int{}, matcher.Options(matcher.EquateEmpty())...) {
				t.Fatal("a nil slice differs from an empty slice under EquateEmpty")
			}
		})

		t.Run("passing it twice is passing it once", func(t *testing.T) {
			t.Parallel()

			var nilSlice []int
			repeated := []matcher.Option{matcher.EquateEmpty(), matcher.EquateEmpty()}

			if !cmp.Equal(nilSlice, []int{}, matcher.Options(repeated...)...) {
				t.Fatal("a repeated option changed the comparison")
			}
		})
	})

	t.Run("EquateNaNs", func(t *testing.T) {
		t.Parallel()

		t.Run("makes NaN equal itself", func(t *testing.T) {
			t.Parallel()

			nan := math.NaN()
			if !cmp.Equal(nan, nan, matcher.Options(matcher.EquateNaNs())...) {
				t.Fatal("NaN differs from NaN under EquateNaNs")
			}
		})

		t.Run("does not relax the empty rule", func(t *testing.T) {
			t.Parallel()

			var nilSlice []int
			if cmp.Equal(nilSlice, []int{}, matcher.Options(matcher.EquateNaNs())...) {
				t.Fatal("EquateNaNs also equated nil with empty; the flags are not independent")
			}
		})
	})
}
