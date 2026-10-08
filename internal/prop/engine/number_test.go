// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/random"
)

// The allocations of each number generator's constructor, measured.
const (
	// integerAllocs are the allocations of Integer and Duration: the
	// decode, the decode with its type erased, the conversion of a choice
	// value to the generator's type, and the inverse.
	integerAllocs = 4
	// floatAllocs are the allocations of Float: the decode, the decode with
	// its type erased, and the inverse.
	floatAllocs = 3
	// booleanAllocs are the allocations of Boolean: the decode and the
	// decode with its type erased. The inverse captures nothing.
	booleanAllocs = 2
)

// pinnedReuse are the first twelve integers over the signed 64-bit range
// that one case of seed 42 draws, from the definition's executable
// reference, so that a change to reuse fails here before a vector moves.
var pinnedReuse = []int64{
	11, 13, math.MaxInt64, math.MaxInt64, -207, -781715023583996500,
	math.MinInt64, -43, -12, -2609675888663766267, -2609675888663766267, 229,
}

// TestNumber checks the integer, duration, float and boolean generators:
// the values they decode, the choices they record, and their refusals.
func TestNumber(t *testing.T) {
	t.Parallel()

	t.Run("Integer", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			g        engine.Generator[int64]
			give     []choice.Choice
			want     int64
			recorded []choice.Choice
		}{
			{
				name:     "returns a choice inside its bounds",
				g:        engine.Integer[int64](-1000, 1000),
				give:     integers(-17),
				want:     -17,
				recorded: integers(-17),
			},
			{
				name:     "returns the target for a choice outside its bounds",
				g:        engine.Integer[int64](-9, -3),
				give:     integers(7),
				want:     -3,
				recorded: integers(-3),
			},
			{
				name:     "returns the target past the last choice",
				g:        engine.Integer[int64](3, 9),
				want:     3,
				recorded: integers(3),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, tt.g, tt.give...)
				assert.Equal(t, got, tt.want, "the decoded integer")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded choices")
			})
		}

		t.Run("returns a choice beyond 2^53", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, engine.Integer[uint64](0, math.MaxUint64), unsigned(math.MaxUint64))
			assert.Equal(t, got, uint64(math.MaxUint64), "the largest uint64")
		})

		t.Run("returns a value of a type defined over an integer type", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, engine.Integer[celsius](-10, 10), integers(-5)...)
			assert.Equal(t, got, celsius(-5), "the defined type")
		})

		t.Run("returns the pinned reuse of earlier values of seed 42", func(t *testing.T) {
			t.Parallel()
			wide := engine.Integer[int64](math.MinInt64, math.MaxInt64)
			got := make([]int64, 0, len(pinnedReuse))
			engine.Generate(func(c *engine.Case) {
				for range len(pinnedReuse) {
					got = append(got, engine.Draw(c, wide, drawn))
				}
			}, 42, 0, nil)
			assert.Equal(t, got, pinnedReuse, "the first twelve draws")
		})

		t.Run("returns the one value of bounds of one value without consuming the stream", func(t *testing.T) {
			t.Parallel()
			seven, wide := engine.Integer(7, 7), engine.Integer[uint64](0, 1_000_000_000)
			var got, want uint64
			engine.Generate(func(c *engine.Case) {
				for range 3 {
					engine.Draw(c, seven, drawn)
				}
				got = engine.Draw(c, wide, drawn)
			}, 7, 0, nil)
			engine.Generate(func(c *engine.Case) { want = engine.Draw(c, wide, drawn) }, 7, 0, nil)
			assert.Equal(t, got, want, "the draw a fresh stream gives")
		})

		t.Run("panics when lo exceeds hi", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { engine.Integer(5, 1) }, "no value")
		})
	})

	t.Run("Duration", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			lo, hi time.Duration
			give   []choice.Choice
			want   time.Duration
		}{
			{
				name: "returns nanoseconds",
				lo:   -time.Second,
				hi:   time.Second,
				give: integers(250_000_000),
				want: 250 * time.Millisecond,
			},
			{
				name: "returns the target for a choice outside its bounds",
				lo:   1000,
				hi:   2000,
				give: integers(5),
				want: 1000,
			},
			{name: "returns the target past the last choice", lo: -2000, hi: -1000, want: -1000},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, engine.Duration(tt.lo, tt.hi), tt.give...)
				assert.Equal(t, got, tt.want, "the decoded duration")
			})
		}

		t.Run("returns a generator whose id is duration", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, engine.Duration(0, time.Second).ID(), "duration", "the id")
		})
	})

	t.Run("Float", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a choice inside its bounds", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, engine.Float(-10.0, 10.0, choice.ExcludeNaN), float(2.5))
			assert.Equal(t, got, 2.5, "the decoded float")
		})

		t.Run("returns the target for NaN when the bounds exclude it", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, engine.Float(math.Inf(-1), math.Inf(1), choice.ExcludeNaN), float(math.NaN()))
			assert.Equal(t, math.Float64bits(got), uint64(0), "the target, +0")
		})

		t.Run("returns the target for a value a float32 cannot state", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, engine.Float[float32](0.25, 0.75, choice.ExcludeNaN), float(0.3))
			assert.Equal(t, got, float32(0.5), "the simplest binary32 value of the range")
			assert.True(t, sameChoices(e.Case.Choices(), []choice.Choice{float(0.5)}), "the recorded target")
		})

		t.Run("returns NaN when the bounds admit it", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, engine.Float(0.0, 1.0, choice.AdmitNaN), float(math.NaN()))
			assert.True(t, math.IsNaN(got), "NaN")
		})

		t.Run("panics when lo exceeds hi", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { engine.Float(1.0, 0.0, choice.ExcludeNaN) }, "no value")
		})
	})

	t.Run("Boolean", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			give     []choice.Choice
			want     bool
			recorded []choice.Choice
		}{
			{name: "returns true for 1", give: integers(1), want: true, recorded: integers(1)},
			{name: "returns false for a choice outside 0 and 1", give: integers(2), want: false, recorded: integers(0)},
			{name: "returns false past the last choice", want: false, recorded: integers(0)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, engine.Boolean(9, 10), tt.give...)
				assert.Equal(t, got, tt.want, "the decoded boolean")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded choice")
			})
		}

		t.Run("returns one coin of the odds in lowest terms", func(t *testing.T) {
			t.Parallel()
			twoThirds := engine.Boolean(4, 6)
			for seed := range uint64(50) {
				var got bool
				engine.Generate(func(c *engine.Case) { got = engine.Draw(c, twoThirds, drawn) }, seed, 0, nil)
				twin := random.ForCase(seed, 0)
				assert.Equal(t, got, twin.Coin(2, 3), "a coin of 2 in 3")
			}
		})

		t.Run("returns the values that the definition pins for odds of 4 in 6", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Boolean(4, 6), referenceSeed, 8)
			assert.Equal(t, values, []bool{true, true, false, true, true, true, true, true},
				"the first eight cases of seed 7")
		})

		t.Run("returns true for odds of one in one", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Boolean(1, 1), referenceSeed, 8)
			assert.Equal(t, values, []bool{true, true, true, true, true, true, true, true},
				"every case of seed 7")
		})

		refusals := []struct {
			name     string
			num, den uint64
		}{
			{name: "panics for a denominator of 0", num: 0, den: 0},
			{name: "panics for a numerator above the denominator", num: 3, den: 2},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Panics(t, func() { engine.Boolean(tt.num, tt.den) }, "no probability")
			})
		}
	})
}

// TestNumberAllocs checks the allocation ceilings of the number
// generators' constructors, which allocate their decode functions.
func TestNumberAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.Integer(0, 9) }, integerAllocs, "Integer allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Duration(0, time.Second) }, integerAllocs, "Duration allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Float(0.0, 1.0, choice.ExcludeNaN) }, floatAllocs,
		"Float allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Boolean(1, 2) }, booleanAllocs, "Boolean allocates its decodes")
}

// BenchmarkNumber measures the constructors of the number generators.
func BenchmarkNumber(b *testing.B) {
	b.Run("Integer", func(b *testing.B) {
		var got engine.Generator[int]
		c := bench.Start(b).MaxAllocs(integerAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Integer(0, 9)
		}
		assert.Equal(b, got.ID(), "integer", "the id")
	})

	b.Run("Duration", func(b *testing.B) {
		var got engine.Generator[time.Duration]
		c := bench.Start(b).MaxAllocs(integerAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Duration(0, time.Second)
		}
		assert.Equal(b, got.ID(), "duration", "the id")
	})

	b.Run("Float", func(b *testing.B) {
		var got engine.Generator[float64]
		c := bench.Start(b).MaxAllocs(floatAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Float(0.0, 1.0, choice.ExcludeNaN)
		}
		assert.Equal(b, got.ID(), "float", "the id")
	})

	b.Run("Boolean", func(b *testing.B) {
		var got engine.Generator[bool]
		c := bench.Start(b).MaxAllocs(booleanAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Boolean(1, 2)
		}
		assert.Equal(b, got.ID(), "boolean", "the id")
	})
}
