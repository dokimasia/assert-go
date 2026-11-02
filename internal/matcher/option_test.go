// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// option keeps the result of an allocation case.
var option matcher.Option

// equateByAllocs is the ceiling of the allocations of EquateBy: the closure
// of the transform and the closure of the option.
const equateByAllocs = 3

// stamped is a value with a count of reads of its own, which byID leaves out.
type stamped struct {
	ID   int
	seen int
}

// byID returns the ID of s.
func byID(s stamped) int { return s.ID }

// bySeen returns the count of reads of s.
func bySeen(s stamped) int { return s.seen }

// recordOf returns the detail of the one record that call reports on a seat.
func recordOf(t *testing.T, call func(seat matcher.Seat)) map[string]any {
	t.Helper()
	seat := &matchertest.Seat{}
	call(seat)
	records := seat.Records()
	if len(records) != 1 {
		t.Fatalf("the call reports %d records, want 1", len(records))
	}
	return records[0].Detail
}

// passes reports whether matcher.Equal passes on got and want under opts.
func passes[T any](got, want T, opts ...matcher.Option) bool {
	seat := &matchertest.Seat{}
	matcher.Equal(seat, matcher.Soft, got, want, "the values are equal", opts...)
	return !seat.Failed()
}

// TestOption checks the options that relax a comparison.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("EquateEmpty", func(t *testing.T) {
		t.Parallel()

		t.Run("makes a nil slice and a nil map equal an empty one", func(t *testing.T) {
			t.Parallel()

			if !passes([]int(nil), []int{}, matcher.EquateEmpty()) {
				t.Fatal("a nil slice differs from an empty slice under EquateEmpty")
			}
			if !passes(map[string]int(nil), map[string]int{}, matcher.EquateEmpty()) {
				t.Fatal("a nil map differs from an empty map under EquateEmpty")
			}
		})

		t.Run("has the effect of one option when passed twice", func(t *testing.T) {
			t.Parallel()

			repeated := []matcher.Option{matcher.EquateEmpty(), matcher.EquateEmpty()}
			if !passes([]int(nil), []int{}, repeated...) {
				t.Fatal("a repeated option changed the comparison")
			}
		})

		t.Run("keeps NaN unequal to NaN", func(t *testing.T) {
			t.Parallel()

			if passes(math.NaN(), math.NaN(), matcher.EquateEmpty()) {
				t.Fatal("EquateEmpty also equated NaN with NaN, want independent flags")
			}
		})
	})

	t.Run("EquateNaNs", func(t *testing.T) {
		t.Parallel()

		t.Run("makes NaN equal NaN", func(t *testing.T) {
			t.Parallel()

			if !passes(math.NaN(), math.NaN(), matcher.EquateNaNs()) {
				t.Fatal("NaN differs from NaN under EquateNaNs")
			}
		})

		t.Run("keeps a nil collection unequal to an empty one", func(t *testing.T) {
			t.Parallel()

			if passes([]int(nil), []int{}, matcher.EquateNaNs()) {
				t.Fatal("EquateNaNs also equated nil with empty, want independent flags")
			}
		})
	})

	t.Run("ByIdentity", func(t *testing.T) {
		t.Parallel()

		t.Run("makes a pointer equal itself and differ from a pointer to an equal value", func(t *testing.T) {
			t.Parallel()

			object := new(1)
			if !passes(object, object, matcher.ByIdentity()) {
				t.Fatal("a pointer differs from itself under ByIdentity")
			}
			if passes(object, new(1), matcher.ByIdentity()) {
				t.Fatal("two allocations of one value are equal under ByIdentity")
			}
		})

		t.Run("keeps NaN unequal to NaN", func(t *testing.T) {
			t.Parallel()

			if passes(math.NaN(), math.NaN(), matcher.ByIdentity()) {
				t.Fatal("ByIdentity also equated NaN with NaN, want independent flags")
			}
		})
	})

	t.Run("EquateBy", func(t *testing.T) {
		t.Parallel()

		t.Run("compares a value of the type by the result of f", func(t *testing.T) {
			t.Parallel()

			if !passes(stamped{ID: 1, seen: 5}, stamped{ID: 1, seen: 9}, matcher.EquateBy(byID)) {
				t.Fatal("two values of one ID differ under EquateBy of the ID")
			}
			if passes(stamped{ID: 1, seen: 5}, stamped{ID: 2, seen: 5}, matcher.EquateBy(byID)) {
				t.Fatal("two values of two IDs are equal under EquateBy of the ID")
			}
		})

		t.Run("applies the last EquateBy of a type", func(t *testing.T) {
			t.Parallel()

			last := []matcher.Option{matcher.EquateBy(byID), matcher.EquateBy(bySeen)}
			if passes(stamped{ID: 1, seen: 5}, stamped{ID: 1, seen: 9}, last...) {
				t.Fatal("two values of two counts are equal, want the last EquateBy to apply")
			}
		})

		t.Run("states the results in the record of Equal", func(t *testing.T) {
			t.Parallel()

			got := recordOf(t, func(seat matcher.Seat) {
				matcher.Equal(seat, matcher.Soft, stamped{ID: 1}, stamped{ID: 2}, "the IDs match",
					matcher.EquateBy(byID))
			})
			if got["want"] != 2 || got["got"] != 1 {
				t.Fatalf("the record states want %v and got %v, want 2 and 1", got["want"], got["got"])
			}
		})

		t.Run("states the result in the record of NotEqual", func(t *testing.T) {
			t.Parallel()

			got := recordOf(t, func(seat matcher.Seat) {
				matcher.NotEqual(seat, matcher.Soft, stamped{ID: 1, seen: 5}, stamped{ID: 1, seen: 9}, "the IDs differ",
					matcher.EquateBy(byID))
			})
			if got["got"] != 1 {
				t.Fatalf("the record states got %v, want 1", got["got"])
			}
		})

		t.Run("states a value that matches no transform as it is", func(t *testing.T) {
			t.Parallel()

			got := recordOf(t, func(seat matcher.Seat) {
				matcher.Equal(seat, matcher.Soft, 1, 2, "the counts match", matcher.EquateBy(byID))
			})
			if got["want"] != 2 || got["got"] != 1 {
				t.Fatalf("the record states want %v and got %v, want 2 and 1", got["want"], got["got"])
			}
		})

		t.Run("states nil for a nil value", func(t *testing.T) {
			t.Parallel()

			got := recordOf(t, func(seat matcher.Seat) {
				matcher.Equal[any](seat, matcher.Soft, nil, 2, "the values match", matcher.EquateBy(byID))
			})
			if got["got"] != nil {
				t.Fatalf("the record states got %v, want nil", got["got"])
			}
		})
	})

	t.Run("relaxes nothing without an option", func(t *testing.T) {
		t.Parallel()

		if passes([]int(nil), []int{}) || passes(math.NaN(), math.NaN()) || !passes(new(1), new(1)) {
			t.Fatal("the comparison without options equated nil with empty or NaN with NaN, or compared by identity")
		}
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

// optionCases returns a call of each function and method of option.go, with
// its allocation ceiling.
func optionCases() []allocCase {
	empty := matcher.EquateEmpty()
	return []allocCase{
		{name: "EquateEmpty", call: func(matcher.Seat) { option = matcher.EquateEmpty() }},
		{name: "EquateNaNs", call: func(matcher.Seat) { option = matcher.EquateNaNs() }},
		{name: "ByIdentity", call: func(matcher.Seat) { option = matcher.ByIdentity() }},
		{name: "EquateBy", call: func(matcher.Seat) { option = matcher.EquateBy(byID) }, allocs: equateByAllocs},
		{name: "Option.FormOption", call: func(matcher.Seat) { empty.FormOption(matcher.FormSeal{}) }},
	}
}
