// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// exampleAllocs is the ceiling of the allocations of Example and of Examples
// of one value: the list of values and the option.
const exampleAllocs = 3

// TestForm checks the options that a property form takes: the run's
// options, its assertion's relaxations, Example and Examples, and the
// faults that fail a form before any case runs.
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
			assert.Equal(t, detail[counterexampleField], any([]prop.Entry{prop.Drawn{Label: "input", Value: int8(99)}}),
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

	t.Run("Example", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the example before any other case, and shrinks it", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x int8) bool { return x < 90 }, contractOfForm, prop.Seed(7), prop.Explain(false),
				prop.Example[int8](120))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the example fails first")
			assert.Equal(t, detail[counterexampleField], any([]prop.Entry{prop.Drawn{Label: "input", Value: int8(90)}}),
				"the example shrinks to the smallest failing input")
		})

		t.Run("runs an example of a nil interface", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x any) bool { return x != nil }, contractOfForm, prop.Seed(7), prop.Shrink(0),
				prop.Using(prop.Just[any](nil)), prop.Example[any](nil))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the example fails first")
			assert.Equal(t, detail[counterexampleField], any([]prop.Entry{prop.Drawn{Label: "input", Value: nil}}),
				"the example's nil input")
		})

		t.Run("runs an example of each generated argument", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			ordered := func(a, b int8) bool { return a == 7 && b == 9 }
			prop.Commutative(rec, ordered, contractOfForm, prop.Seed(7), prop.Shrink(0), prop.Example[int8](7, 9))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the example fails first")
			got := detail[counterexampleField].([]prop.Entry)
			assert.Equal(t, valuesOfDraws(got), []any{int8(7), int8(9)}, "the two values of the example")
		})

		t.Run("runs an example of an input without an inverse on its value, as found", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			mapped := prop.Integer[int8](0, 9).Map(func(x int8) int8 { return x })
			prop.True(rec, func(x int8) bool { return x >= 0 }, contractOfForm, prop.Seed(7),
				prop.Using(mapped), prop.Example[int8](-5))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(0), "the example fails first")
			got := detail[counterexampleField].([]prop.Entry)
			assert.Equal(t, valuesOfDraws(got), []any{int8(-5)}, "the stated value, outside the source's domain")
			assert.Nil(t, detail[choicesField], "no replay token")
			assert.Equal(t, detail[othersField], any([]prop.Other{}), "no other failure")
		})

		t.Run("writes no store entry for an example of values", func(t *testing.T) {
			t.Parallel()
			rec, dir := assert.NewRecorder(), t.TempDir()
			mapped := prop.Integer[int8](0, 9).Map(func(x int8) int8 { return x })
			prop.True(rec, func(x int8) bool { return x >= 0 }, contractOfForm, prop.Seed(7), prop.Store(dir),
				prop.Using(mapped), prop.Example[int8](-5))
			entries, err := os.ReadDir(dir)
			assert.NoError(t, err, "the store's directory reads")
			assert.True(t, rec.Failed(), "the example fails")
			assert.Empty(t, entries, "no entry")
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

	t.Run("Examples", func(t *testing.T) {
		t.Parallel()

		t.Run("runs a case of each value, in order, before any other case", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x int8) bool { return x < 90 }, contractOfForm, prop.Seed(7), prop.Explain(false),
				prop.Examples[int8](3, 120, 5))
			detail := rec.Failures()[0].Detail
			assert.Equal(t, detail[casesField], any(1), "the second example fails after the first passes")
			assert.Equal(t, detail[counterexampleField], any([]prop.Entry{prop.Drawn{Label: "input", Value: int8(90)}}),
				"the example shrinks to the smallest failing input")
		})

		t.Run("runs Example and Examples in the order of the options", func(t *testing.T) {
			t.Parallel()
			var seen []int8
			prop.True(assert.NewRecorder(), func(x int8) bool { seen = append(seen, x); return true }, contractOfForm,
				prop.Seed(7), prop.Examples[int8](1, 2), prop.Example[int8](3))
			assert.Equal(t, seen[:3], []int8{1, 2, 3}, "the values of the two options")
		})

		tests := []struct {
			name string
			give func(tb assert.TB)
			want fault.Error
		}{
			{
				name: "fails the run at once for a form over more than one input",
				give: func(tb assert.TB) {
					prop.Commutative(tb, subtract, contractOfForm, prop.Examples[int8](1))
				},
				want: fault.Error{
					Op:     "prop.Commutative",
					Path:   fault.Path{fault.Field("Examples"), fault.Index(0)},
					Reason: "the examples state one value per case, and the form generates 2",
				},
			},
			{
				name: "fails the run at once for examples of another type",
				give: func(tb assert.TB) {
					prop.Equal(tb, same, same, contractOfForm, prop.Example[int8](1), prop.Examples[int16](1))
				},
				want: fault.Error{
					Op:     "prop.Equal",
					Path:   fault.Path{fault.Field("Examples"), fault.Index(1)},
					Reason: "the example states values of int16, and the input is of int8",
				},
			},
			{
				name: "fails the run at once at a value that the generator does not produce",
				give: func(tb assert.TB) {
					prop.True(tb, func(int8) bool { return true }, contractOfForm,
						prop.Using(prop.Integer[int8](0, 9)), prop.Examples[int8](1, 12))
				},
				want: fault.Error{
					Op:     "prop.True",
					Path:   fault.Path{fault.Field("Examples"), fault.Index(0), fault.Index(1)},
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

// TestFormAllocs checks the allocation ceilings of Example and Examples.
func TestFormAllocs(t *testing.T) {
	var kept prop.FormOption
	assert.MaxAllocs(t, func() { kept = prop.Example[int8](1) }, exampleAllocs, "Example allocates its option")
	assert.MaxAllocs(t, func() { kept = prop.Examples[int8](1) }, exampleAllocs, "Examples allocates its option")
	assert.NotNil(t, kept, "the kept option")
}

// BenchmarkForm measures Example and Examples.
func BenchmarkForm(b *testing.B) {
	b.Run("Example", func(b *testing.B) {
		var got prop.FormOption
		c := bench.Start(b).MaxAllocs(exampleAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Example[int8](1)
		}
		assert.NotNil(b, got, "the option")
	})

	b.Run("Examples", func(b *testing.B) {
		var got prop.FormOption
		c := bench.Start(b).MaxAllocs(exampleAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Examples[int8](1)
		}
		assert.NotNil(b, got, "the option")
	})
}
