// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.dokimi.dev/assert/internal/matcher"
)

// The results of the allocation cases, which keep each call.
var (
	option  matcher.Option
	options []cmp.Option
)

// TestOption checks the rules of a comparison and the options that relax
// them.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("Options", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a nil collection unequal to an empty one", func(t *testing.T) {
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

		t.Run("compares unexported fields", func(t *testing.T) {
			t.Parallel()

			type hidden struct{ n int }

			if cmp.Equal(hidden{n: 1}, hidden{n: 2}, matcher.Options()...) {
				t.Fatal("structs differing in an unexported field compared equal")
			}
			if !cmp.Equal(hidden{n: 1}, hidden{n: 1}, matcher.Options()...) {
				t.Fatal("identical structs with an unexported field compared unequal")
			}
		})

		t.Run("makes a function equal itself", func(t *testing.T) {
			t.Parallel()

			f := func() {}
			if !cmp.Equal(f, f, matcher.Options()...) {
				t.Fatal("a function compared against itself is unequal")
			}
		})

		t.Run("keeps two different functions unequal", func(t *testing.T) {
			t.Parallel()

			f, g := func() {}, func() {}
			if cmp.Equal(f, g, matcher.Options()...) {
				t.Fatal("two distinct functions compared equal")
			}
		})

		t.Run("keeps NaN unequal to NaN", func(t *testing.T) {
			t.Parallel()

			nan := math.NaN()
			if cmp.Equal(nan, nan, matcher.Options()...) {
				t.Fatal("NaN equals NaN, want unequal")
			}
		})

		t.Run("compares floats exactly", func(t *testing.T) {
			t.Parallel()

			// Variables, not constants: untyped constant arithmetic is
			// exact at compile time, and 0.1+0.2 would equal 0.3.
			tenth, fifth := 0.1, 0.2
			if cmp.Equal(tenth+fifth, 0.3, matcher.Options()...) {
				t.Fatal("0.1+0.2 equals 0.3, want an exact comparison without a tolerance")
			}
		})

		t.Run("returns a new slice on every call", func(t *testing.T) {
			t.Parallel()

			first := matcher.Options()
			second := matcher.Options()
			first[0] = nil

			if second[0] == nil {
				t.Fatal("two calls share backing memory, want a new slice per call")
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

		t.Run("has the effect of one option when passed twice", func(t *testing.T) {
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

		t.Run("keeps a nil collection unequal to an empty one", func(t *testing.T) {
			t.Parallel()

			var nilSlice []int
			if cmp.Equal(nilSlice, []int{}, matcher.Options(matcher.EquateNaNs())...) {
				t.Fatal("EquateNaNs also equated nil with empty, want independent flags")
			}
		})
	})
}

// TestOptionAllocs checks the allocation ceiling of each function and
// method of option.go.
func TestOptionAllocs(t *testing.T) {
	checkAllocs(t, optionCases())
}

// BenchmarkOption measures each function and method of option.go.
func BenchmarkOption(b *testing.B) {
	benchAllocs(b, optionCases())
}

// optionCases returns a call of each function and method of option.go,
// with its allocation ceiling, measured: Options of one option.
func optionCases() []allocCase {
	empty := matcher.EquateEmpty()
	return []allocCase{
		{name: "EquateEmpty", call: func(matcher.Seat) { option = matcher.EquateEmpty() }},
		{name: "EquateNaNs", call: func(matcher.Seat) { option = matcher.EquateNaNs() }},
		{name: "Options", call: func(matcher.Seat) { options = matcher.Options(empty) }, allocs: 9},
		{name: "Option.FormOption", call: func(matcher.Seat) { empty.FormOption(matcher.FormSeal{}) }},
	}
}
