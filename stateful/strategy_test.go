// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/stateful"
)

// TestStrategy checks the two strategies of a scheduler: the zero value is
// Uniform, and PCT refuses a depth below 1.
func TestStrategy(t *testing.T) {
	t.Parallel()

	t.Run("Uniform", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the zero Strategy", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, stateful.Uniform(), stateful.Strategy{}, "the zero value")
		})
	})

	t.Run("PCT", func(t *testing.T) {
		t.Parallel()

		t.Run("panics for a depth below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.PCT(0) }, "no depth")
			assert.Equal(t, got, any("stateful: PCT(0) is below 1"), "the panic names the strategy")
		})

		t.Run("returns a strategy other than Uniform for a depth of 1", func(t *testing.T) {
			t.Parallel()
			assert.NotEqual(t, stateful.PCT(1), stateful.Uniform(), "PCT without change points")
		})
	})
}

// TestStrategyAllocs checks that the strategies allocate nothing.
func TestStrategyAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = stateful.Uniform() }, 0, "Uniform allocates nothing")
	assert.MaxAllocs(t, func() { _ = stateful.PCT(3) }, 0, "PCT allocates nothing")
}

// BenchmarkStrategy measures the strategies under a ceiling of no
// allocation.
func BenchmarkStrategy(b *testing.B) {
	b.Run("Uniform", func(b *testing.B) {
		var got stateful.Strategy
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = stateful.Uniform()
		}
		assert.Equal(b, got, stateful.Strategy{}, "the zero value")
	})
	b.Run("PCT", func(b *testing.B) {
		var got stateful.Strategy
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = stateful.PCT(3)
		}
		assert.NotEqual(b, got, stateful.Strategy{}, "a strategy other than Uniform")
	})
}
