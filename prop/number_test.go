// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// The allocations of the number generators, measured. Each is the
// engine's construction of the generator: the closure of its decode, of its
// erased decode and of its inverse, and for an integer the closure that
// converts a choice to its value.
const (
	// integerAllocs are the allocations of Integer.
	integerAllocs = 4
	// durationAllocs are the allocations of Duration.
	durationAllocs = 4
	// floatAllocs are the allocations of Float.
	floatAllocs = 3
	// booleanAllocs are the allocations of Boolean.
	booleanAllocs = 3
)

// cents is an integer type defined over int64, as an amount of money.
type cents int64

// TestNumber checks the number generators and their options: the values
// they decode, what their options change, and the arguments they panic
// for.
func TestNumber(t *testing.T) {
	t.Parallel()

	t.Run("Integer", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give uint64
			want int
		}{
			{name: "returns a replayed value inside its bounds", give: 7, want: 7},
			{name: "returns the target for a replayed value outside its bounds", give: 12, want: 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(prop.Integer(0, 9), tt.give), tt.want, "the digit")
			})
		}

		t.Run("returns the largest unsigned value for Integer of uint64", func(t *testing.T) {
			t.Parallel()
			got := first(prop.Integer[uint64](0, math.MaxUint64), math.MaxUint64)
			assert.Equal(t, got, uint64(math.MaxUint64), "the replayed maximum")
		})

		t.Run("returns a value of a type defined over an integer", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(prop.Integer[cents](-500, 500), 250), cents(250), "the replayed amount")
		})

		t.Run("panics for a lower bound above the upper bound", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.Integer(5, 1) }, "the bounds state no value")
			assert.Equal(t, got, any("prop: integer(5, 1) states no value"), "the panic names the bounds")
		})
	})

	t.Run("Duration", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a replayed number of nanoseconds", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(prop.Duration(0, time.Second), 1500), 1500*time.Nanosecond, "the duration")
		})

		t.Run("panics for a lower bound above the upper bound", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { prop.Duration(time.Second, 0) }, "the bounds state no value")
		})
	})

	t.Run("AllowNaN", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Generator[float64]
			want float64
		}{
			{name: "makes Float return a replayed NaN", give: prop.Float(0.0, 1.0, prop.AllowNaN()), want: math.NaN()},
			{name: "leaves a replayed NaN the target without it", give: prop.Float(0.0, 1.0), want: 0},
			{
				name: "leaves NaN a value after a zero option",
				give: prop.Float(0.0, 1.0, prop.AllowNaN(), prop.FloatOption{}),
				want: math.NaN(),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := replayedOf(tt.give, floating(math.NaN()))
				assert.Equal(t, got, tt.want, "the value of the NaN choice", assert.EquateNaNs())
			})
		}
	})

	t.Run("Float", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give float64
			want float32
		}{
			{name: "returns a replayed value of width 32", give: 0.5, want: 0.5},
			{name: "returns the target for a replayed value of width 64 alone", give: 0.1, want: 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, replayedOf(prop.Float[float32](0, 1), floating(tt.give)), tt.want, "the float32")
			})
		}

		t.Run("panics for a NaN bound", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { prop.Float(math.NaN(), 1) }, "a NaN bound states no range")
		})

		t.Run("panics for a lower bound above the upper bound", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { prop.Float(1.0, 0.0) }, "the bounds state no value")
		})
	})

	t.Run("Odds", func(t *testing.T) {
		t.Parallel()

		t.Run("makes Boolean draw true with the stated odds", func(t *testing.T) {
			t.Parallel()
			got := trace(prop.Boolean(prop.Odds(1, 4)))
			assert.Equal(t, got, engineTrace(engine.Boolean(1, 4)), "the engine's draws of the same odds")
			assert.NotEqual(t, got, trace(prop.Boolean()), "other draws than odds of 1/2")
		})

		t.Run("keeps the odds after a zero option", func(t *testing.T) {
			t.Parallel()
			got := trace(prop.Boolean(prop.Odds(1, 4), prop.BooleanOption{}))
			assert.Equal(t, got, engineTrace(engine.Boolean(1, 4)), "the stated odds")
		})

		tests := []struct {
			name string
			give [2]uint64
			want string
		}{
			{
				name: "panics for a denominator of 0",
				give: [2]uint64{0, 0},
				want: "prop: Odds(0, 0) states no probability",
			},
			{
				name: "panics for a numerator above the denominator",
				give: [2]uint64{2, 1},
				want: "prop: Odds(2, 1) states no probability",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := assert.Panics(t, func() { prop.Odds(tt.give[0], tt.give[1]) }, "the odds state no probability")
				assert.Equal(t, got, any(tt.want), "the panic names the option and its arguments")
			})
		}

		t.Run("returns odds of certainty", func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { prop.Odds(1, 1) }, "a numerator equal to the denominator")
		})
	})

	t.Run("Boolean", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give uint64
			want bool
		}{
			{name: "returns false for a replayed 0", give: 0, want: false},
			{name: "returns true for a replayed 1", give: 1, want: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(prop.Boolean(), tt.give), tt.want, "the boolean")
			})
		}

		t.Run("draws true with odds of 1/2 by default", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, trace(prop.Boolean()), engineTrace(engine.Boolean(1, 2)), "the engine's draws of even odds")
		})
	})
}

// TestNumberAllocs checks the allocation ceilings of the number
// generators.
func TestNumberAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.Integer(0, 9) }, integerAllocs, "Integer allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.Duration(0, time.Second) }, durationAllocs, "Duration allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.Float(0.0, 1.0) }, floatAllocs, "Float allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.AllowNaN() }, 0, "AllowNaN allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.Odds(1, 4) }, 0, "Odds allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.Boolean() }, booleanAllocs, "Boolean allocates its decode")
}

// BenchmarkNumber measures each number generator and option.
func BenchmarkNumber(b *testing.B) {
	b.Run("Integer", func(b *testing.B) {
		var got prop.Generator[int]
		c := bench.Start(b).MaxAllocs(integerAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Integer(0, 9)
		}
		assert.Equal(b, first(got, 7), 7, "the digit")
	})

	b.Run("Duration", func(b *testing.B) {
		var got prop.Generator[time.Duration]
		c := bench.Start(b).MaxAllocs(durationAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Duration(0, time.Second)
		}
		assert.Equal(b, first(got, 7), 7*time.Nanosecond, "the duration")
	})

	b.Run("Float", func(b *testing.B) {
		var got prop.Generator[float64]
		c := bench.Start(b).MaxAllocs(floatAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Float(0.0, 1.0)
		}
		assert.Equal(b, replayedOf(got, floating(0.5)), 0.5, "the float")
	})

	b.Run("AllowNaN", func(b *testing.B) {
		var got prop.FloatOption
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.AllowNaN()
		}
		assert.Equal(b, got, prop.AllowNaN(), "the option")
	})

	b.Run("Odds", func(b *testing.B) {
		var got prop.BooleanOption
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Odds(1, 4)
		}
		assert.Equal(b, got, prop.Odds(1, 4), "the option")
	})

	b.Run("Boolean", func(b *testing.B) {
		var got prop.Generator[bool]
		c := bench.Start(b).MaxAllocs(booleanAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Boolean()
		}
		assert.True(b, first(got, 1), "the boolean")
	})
}

// trace returns the values that a run of seed 7 draws from g, in the order
// its body drew them. The body draws a wide integer before each value, so
// the run does not test every input of a small domain early.
func trace[T any](g prop.Generator[T]) []T {
	var out []T
	prop.ForAll(assert.NewRecorder(), contract, func(c *prop.Case) {
		c.Draw(prop.Integer(0, 1000000000), "wide")
		out = append(out, c.Draw(g, drawn))
	}, prop.Seed(7))
	return out
}

// engineTrace returns the values that the engine's run of seed 7, with the
// body of trace, draws from g.
func engineTrace[T any](g engine.Generator[T]) []T {
	var out []T
	settings := engine.Settings{
		Seed:       7,
		Cases:      engine.DefaultCases,
		MaxChoices: engine.MaxChoices,
		Shrink:     engine.DefaultShrink,
		Explain:    true,
		Workers:    1,
	}
	engine.Run(func(c *engine.Case) {
		engine.Draw(c, engine.Integer(0, 1000000000), "wide")
		out = append(out, engine.Draw(c, g, drawn))
	}, settings)
	return out
}
