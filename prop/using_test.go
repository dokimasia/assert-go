// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/prop"
)

// usingAllocs is the ceiling of the allocations of Using: the generator
// with its type erased, its inverse, the generator of the type as an any,
// and the option.
const usingAllocs = 5

// tagged is a struct whose field of int8 a prop tag bounds.
type tagged struct {
	N int8 `json:"n" prop:"min=1"`
}

// TestUsing checks that Using makes a generator the generator of its type
// in a run of ForAll and of a property form, and in each value of a draw of
// Of.
func TestUsing(t *testing.T) {
	t.Parallel()

	threes := prop.Integer[int8](3, 3)

	t.Run("Using", func(t *testing.T) {
		t.Parallel()

		t.Run("draws Of of the stated type from the stated generator", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, func(c *prop.Case) {
				assert.Equal(c, c.Draw(prop.Of[int8](), drawn), int8(3), "a value of the stated generator")
			}, prop.Seed(7), prop.Using(threes))
			assert.False(t, rec.Failed(), "every draw is 3")
		})

		t.Run("draws each value of the stated type in a value of Of", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, func(c *prop.Case) {
				assert.Equal(c, c.Draw(prop.Of[order](), drawn).ID, uint32(3), "the stated id")
			}, prop.Seed(7), prop.Using(prop.Integer[uint32](3, 3)))
			assert.False(t, rec.Failed(), "every id is 3")
		})

		t.Run("draws Of inside another generator from the stated generator", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			lists := prop.List(prop.Of[int8]())
			prop.ForAll(rec, contract, func(c *prop.Case) {
				for _, v := range c.Draw(lists, drawn) {
					assert.Equal(c, v, int8(3), "an element of the stated generator")
				}
			}, prop.Seed(7), prop.Using(threes))
			assert.False(t, rec.Failed(), "every element is 3")
		})

		t.Run("takes the stated generator over a registration", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, func(c *prop.Case) {
				assert.Equal(c, c.Draw(prop.Of[status](), drawn), statusShipped, "the stated status")
			}, prop.Seed(7), prop.Using(prop.Just(statusShipped)))
			assert.False(t, rec.Failed(), "every status is shipped")
		})

		t.Run("leaves a generator of its own domain as it is", func(t *testing.T) {
			t.Parallel()
			var seen []int8
			prop.ForAll(assert.NewRecorder(), contract, func(c *prop.Case) {
				seen = append(seen, c.Draw(prop.Integer[int8](10, 20), drawn))
			}, prop.Seed(7), prop.Using(threes))
			assert.NotEmpty(t, seen, "the run drew values")
			assert.True(t, slices.Min(seen) >= 10, "every value is in the integer's own domain")
		})

		t.Run("draws Of of the stated type inside the stated generator from its other generator", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			naturals := prop.Of[int8]().Filter(func(v int8) bool { return v >= 0 })
			prop.ForAll(rec, contract, func(c *prop.Case) {
				assert.True(c, c.Draw(prop.Of[int8](), drawn) >= 0, "a value of the filtered derivation")
			}, prop.Seed(7), prop.Using(naturals))
			assert.False(t, rec.Failed(), "no value is negative")
		})

		t.Run("applies the run's other generators inside a stated generator", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			same := prop.Of[order]().Map(func(o order) order { return o })
			prop.ForAll(rec, contract, func(c *prop.Case) {
				assert.Equal(c, c.Draw(prop.Of[order](), drawn).ID, uint32(3), "the stated id inside the stated order")
			}, prop.Seed(7), prop.Using(same), prop.Using(prop.Integer[uint32](3, 3)))
			assert.False(t, rec.Failed(), "every id is 3")
		})

		t.Run("takes the last generator of one type", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, func(c *prop.Case) {
				assert.Equal(c, c.Draw(prop.Of[int8](), drawn), int8(4), "a value of the last generator")
			}, prop.Seed(7), prop.Using(threes), prop.Using(prop.Integer[int8](4, 4)))
			assert.False(t, rec.Failed(), "every draw is 4")
		})

		t.Run("runs an entry of Draws back through the stated generator", func(t *testing.T) {
			t.Parallel()
			var first []int8
			prop.ForAll(assert.NewRecorder(), contract, func(c *prop.Case) {
				values := c.Draw(prop.List(prop.Of[int8]()), drawn)
				if first == nil {
					first = values
				}
			}, prop.Seed(7), prop.Using(prop.SampledFrom[int8](9, 5)),
				prop.Draws(`[{"label": "value", "value": {"type": "list", "items": [{"type": "int", "value": 5}]}}]`))
			assert.Equal(t, first, []int8{5}, "the list of the entry")
		})

		t.Run("fails the case of a draw of a type that does not read with the stated generators", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, func(c *prop.Case) { c.Draw(prop.Of[tagged](), drawn) },
				prop.Seed(7), prop.Using(threes))
			got, ok := rec.Failures()[0].Detail[failureField].(assert.Failure)
			assert.True(t, ok, "the failure of the case")
			assert.Equal(t, got.Contract,
				"prop: Of[prop_test.tagged]: tagged.N: the tag key min applies to no part of int8",
				"the panic of the draw")
		})

		t.Run("generates the input of a form with the stated generator", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.True(rec, func(x int8) bool { return x == 3 }, contractOfForm, prop.Seed(7), prop.Using(threes))
			assert.False(t, rec.Failed(), "every input is 3")
		})

		t.Run("generates each value of the stated type that the input of a form contains", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			allThree := func(o order) bool { return o.ID == 3 }
			prop.True(rec, allThree, contractOfForm, prop.Seed(7), prop.Using(prop.Integer[uint32](3, 3)))
			assert.False(t, rec.Failed(), "every id is 3")
		})

		t.Run("generates the input of a form whose type the reader refuses", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			piped := make(chan int)
			prop.True(rec, func(c chan int) bool { return c == piped }, contractOfForm, prop.Seed(7),
				prop.Using(prop.Just(piped)))
			assert.False(t, rec.Failed(), "the stated channel")
		})

		t.Run("runs an example of a form back through the stated generator", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			other := status("other")
			prop.True(rec, func(s []status) bool { return len(s) < 1 || s[0] == other }, contractOfForm,
				prop.Seed(7), prop.Shrink(0), prop.Using(prop.Just(other)), prop.Example([]status{other}))
			assert.Empty(t, rec.Faults(), "the example runs back")
			assert.False(t, rec.Failed(), "every status is the stated one")
		})

		t.Run("fails a form at once for an input whose type the reader refuses", func(t *testing.T) {
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
}

// TestUsingAllocs checks the allocation ceiling of Using.
func TestUsingAllocs(t *testing.T) {
	digits := prop.Integer[int8](0, 9)
	var kept prop.Option
	assert.MaxAllocs(t, func() { kept = prop.Using(digits) }, usingAllocs, "Using allocates its option")
	assert.NotNil(t, kept, "the kept option")
}

// BenchmarkUsing measures Using.
func BenchmarkUsing(b *testing.B) {
	b.Run("Using", func(b *testing.B) {
		digits := prop.Integer[int8](0, 9)
		var got prop.Option
		c := bench.Start(b).MaxAllocs(usingAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Using(digits)
		}
		assert.NotNil(b, got, "the option")
	})
}
