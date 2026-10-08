// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestRequest checks how the random phase draws each kind of request: an
// integer, a float or a sequence from its bounds, a coin from its odds and
// a continue flag from its collection's sizes, pinned to the values that
// the definition's executable reference draws.
func TestRequest(t *testing.T) {
	t.Parallel()

	t.Run("Generate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns integers drawn from a signed range", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Integer(-100, 100), 42, 6)
			assert.Equal(t, values, []int{11, 0, 76, 0, 96, 11}, "the values of the first six cases of seed 42")
		})

		t.Run("returns durations drawn from a negative range", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Duration(-time.Millisecond, -1), 7, 6)
			assert.Equal(t, values, []time.Duration{-545470, -36, -37325, -16, -10, -38314},
				"the values of the first six cases of seed 7")
		})

		t.Run("returns durations drawn from a range of nanoseconds", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Duration(0, time.Minute), 42, 6)
			assert.Equal(t, values, []time.Duration{51, 23677194335, 13, 0, 8, 40},
				"the values of the first six cases of seed 42")
		})

		t.Run("returns floats drawn from a finite range", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Float(-10.0, 10.0, choice.ExcludeNaN), 42, 8)
			assert.Equal(t, values, []float64{
				1.602474125133466e-183, -1.882663670876988e-96, -1, 0, -1, 2.6041284494948507e-210, -10, 10,
			}, "the values of the first eight cases of seed 42")
		})

		t.Run("returns floats drawn from every float", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Float(math.Inf(-1), math.Inf(1), choice.AdmitNaN), 7, 8)
			assert.Equal(t, values, []float64{
				-34533, 137, -2.1166818007598891e+43, 9007199254740991, 205, -108, 5, 0,
			}, "the values of the first eight cases of seed 7")
		})

		t.Run("returns coins with even odds", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.Boolean(1, 2), 42, 6)
			assert.Equal(t, values, []bool{false, false, true, true, true, false},
				"the values of the first six cases of seed 42")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				integers(0), integers(0), integers(1), integers(1), integers(1), integers(0),
			}), "one choice in [0, 1] for each coin")
		})

		t.Run("returns coins with stated odds", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Boolean(1, 3), 7, 6)
			assert.Equal(t, values, []bool{true, false, false, false, false, true},
				"the values of the first six cases of seed 7")
		})
	})
}
