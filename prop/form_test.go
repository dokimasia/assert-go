// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// The allocations of the options of a form that its caller keeps,
// measured.
const (
	// usingAllocs are the allocations of Using: the generator with its type
	// erased, its inverse, and the option.
	usingAllocs = 3
	// exampleAllocs are the allocations of Example of one value: the list of
	// values and the boxed value.
	exampleAllocs = 2
)

// TestForm checks the options that a property form takes: the run's
// options, its assertion's relaxations, Using and Example, and the faults
// that fail a form before any case runs.
func TestForm(t *testing.T) {
	t.Parallel()

	t.Run("FormOption", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the run of the form with an option of the run", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.Equal(rec, same, next, contractOfForm, prop.Seed(11))
			assert.Equal(t, rec.Failures()[0].Detail[seedField], any("11"), "the seed of the option")
		})

		t.Run("fails the run at once for an option that its run refuses", func(t *testing.T) {
			t.Parallel()
			seat := &matchertest.Seat{}
			prop.Equal(seat, same, next, contractOfForm, prop.Draws("{"))
			expectOnlyFault(t, seat.Faults(), fault.Error{
				Op:     "prop.Equal",
				Path:   fault.Path{fault.Field("Draws")},
				Reason: "the entries are no JSON array of objects",
			})
			assert.Empty(t, seat.Records(), "no record of the form")
		})

		t.Run("runs the case of Draws, whose draw is labelled input", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x int8) bool { return x != 99 }, contractOfForm, prop.Seed(7), prop.Explain(false),
				prop.Draws(`[{"label": "input", "value": {"type": "int", "value": 99}}]`))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the case of the entries fails first")
			assert.Equal(t, detail[counterexampleField], any([]prop.Drawn{{Label: "input", Value: int8(99)}}),
				"the entry's value")
		})

		t.Run("relaxes the comparison of the form's assertion in every case", func(t *testing.T) {
			t.Parallel()
			nan := func(int8) float64 { return math.NaN() }
			rec := assert.NewRecorder()
			prop.Equal(rec, nan, nan, contractOfForm, prop.Seed(7), assert.EquateNaNs())
			assert.False(t, rec.Failed(), "NaN equals NaN under the relaxation")
			prop.Equal(rec, nan, nan, contractOfForm, prop.Seed(7))
			assert.True(t, rec.Failed(), "NaN does not equal NaN without it")
		})

		t.Run("fails the run at once for a relaxation of a form whose assertion takes none", func(t *testing.T) {
			t.Parallel()
			seat := &matchertest.Seat{}
			var calls int
			prop.True(seat, func(int8) bool { calls++; return true }, contractOfForm, assert.EquateEmpty())
			expectOnlyFault(t, seat.Faults(), fault.Error{
				Op:     "prop.True",
				Reason: "the form takes no relaxation, because its assertion takes none",
			})
			assert.Equal(t, calls, 0, "no case runs")
		})
	})

	t.Run("Using", func(t *testing.T) {
		t.Parallel()

		t.Run("generates the input with the stated generator", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x int8) bool { return x == 3 }, contractOfForm, prop.Seed(7),
				prop.Using(prop.Integer[int8](3, 3)))
			assert.False(t, rec.Failed(), "every input is 3")
		})

		t.Run("generates each value of the stated type that the input contains", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			allThree := func(o order) bool { return o.ID == 3 }
			prop.True(rec, allThree, contractOfForm, prop.Seed(7), prop.Using(prop.Integer[uint32](3, 3)))
			assert.False(t, rec.Failed(), "every id is 3")
		})

		t.Run("generates an input whose type the reader refuses", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			piped := make(chan int)
			prop.True(rec, func(c chan int) bool { return c == piped }, contractOfForm, prop.Seed(7),
				prop.Using(prop.Just(piped)))
			assert.False(t, rec.Failed(), "the stated channel")
		})

		t.Run("takes the last generator of one type", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x int8) bool { return x == 4 }, contractOfForm, prop.Seed(7),
				prop.Using(prop.Integer[int8](3, 3)), prop.Using(prop.Integer[int8](4, 4)))
			assert.False(t, rec.Failed(), "every input is 4")
		})

		t.Run("fails the run at once for an input whose type the reader refuses", func(t *testing.T) {
			t.Parallel()
			seat := &matchertest.Seat{}
			prop.True(seat, func(chan int) bool { return true }, contractOfForm)
			expectOnlyFault(t, seat.Faults(), fault.Error{
				Op:     "prop.True",
				Path:   fault.Path{fault.Field("chan int")},
				Reason: "chan int is no type that a shape states",
			})
		})
	})

	t.Run("Example", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the example before any other case, and shrinks it", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x int8) bool { return x < 90 }, contractOfForm, prop.Seed(7), prop.Explain(false),
				prop.Example[int8](120))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the example fails first")
			assert.Equal(t, detail[counterexampleField], any([]prop.Drawn{{Label: "input", Value: int8(90)}}),
				"the example shrinks to the smallest failing input")
		})

		t.Run("runs an example of a nil interface", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x any) bool { return x != nil }, contractOfForm, prop.Seed(7), prop.Shrink(0),
				prop.Using(prop.Just[any](nil)), prop.Example[any](nil))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the example fails first")
			assert.Equal(t, detail[counterexampleField], any([]prop.Drawn{{Label: "input", Value: nil}}),
				"the example's nil input")
		})

		t.Run("runs an example of each generated argument", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			ordered := func(a, b int8) bool { return a == 7 && b == 9 }
			prop.Commutative(rec, ordered, contractOfForm, prop.Seed(7), prop.Shrink(0), prop.Example[int8](7, 9))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the example fails first")
			got := detail[counterexampleField].([]prop.Drawn)
			assert.Equal(t, valuesOfDraws(got), []any{int8(7), int8(9)}, "the two values of the example")
		})

		tests := []struct {
			name string
			give func(tb assert.TB)
			want fault.Error
		}{
			{
				name: "fails the run at once for an example of another number of values",
				give: func(tb assert.TB) {
					prop.Commutative(tb, subtract, contractOfForm, prop.Example[int8](1))
				},
				want: fault.Error{
					Op:     "prop.Commutative",
					Path:   fault.Path{fault.Field("Example"), fault.Index(0)},
					Reason: "the example states 1 values, and the form generates 2",
				},
			},
			{
				name: "fails the run at once for an example of another type",
				give: func(tb assert.TB) {
					prop.Equal(tb, same, same, contractOfForm, prop.Example[int16](1))
				},
				want: fault.Error{
					Op:     "prop.Equal",
					Path:   fault.Path{fault.Field("Example"), fault.Index(0)},
					Reason: "the example states values of int16, and the input is of int8",
				},
			},
			{
				name: "fails the run at once at a value of an example that the generator does not produce",
				give: func(tb assert.TB) {
					prop.Commutative(tb, subtract, contractOfForm, prop.Using(prop.Integer[int8](0, 9)),
						prop.Example[int8](1, 2), prop.Example[int8](3, 12))
				},
				want: fault.Error{
					Op:     "prop.Commutative",
					Path:   fault.Path{fault.Field("Example"), fault.Index(1), fault.Index(1)},
					Kind:   engine.ErrCannotInvert,
					Reason: "12 is outside [0, 9]",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				seat := &matchertest.Seat{}
				tt.give(seat)
				expectOnlyFault(t, seat.Faults(), tt.want)
				assert.Empty(t, seat.Records(), "no record of the form")
			})
		}
	})
}

// TestFormAllocs checks the allocation ceilings of Using and Example.
func TestFormAllocs(t *testing.T) {
	digits := prop.Integer[int8](0, 9)
	var kept prop.FormOption
	assert.MaxAllocs(t, func() { kept = prop.Using(digits) }, usingAllocs, "Using allocates its option")
	assert.MaxAllocs(t, func() { kept = prop.Example[int8](1) }, exampleAllocs, "Example allocates its option")
	assert.NotNil(t, kept, "the kept option")
}

// BenchmarkForm measures Using and Example.
func BenchmarkForm(b *testing.B) {
	b.Run("Using", func(b *testing.B) {
		digits := prop.Integer[int8](0, 9)
		var got prop.FormOption
		c := bench.Start(b).MaxAllocs(usingAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Using(digits)
		}
		assert.NotNil(b, got, "the option")
	})

	b.Run("Example", func(b *testing.B) {
		var got prop.FormOption
		c := bench.Start(b).MaxAllocs(exampleAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Example[int8](1)
		}
		assert.NotNil(b, got, "the option")
	})
}

// valuesOfDraws returns the values of draws, in order.
func valuesOfDraws(draws []prop.Drawn) []any {
	out := make([]any, len(draws))
	for i, d := range draws {
		out[i] = d.Value
	}
	return out
}
