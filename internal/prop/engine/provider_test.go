// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/random"
)

// alternatingReuse are the values that case 0 of seed 42 draws from two
// integer generators in turn, over [0, 10^9] and [0, 10^9 - 1], from the
// definition's executable reference. The fourth value repeats the second
// and the eleventh the fifth: each generator reuses its own bounds' values.
var alternatingReuse = []int{51, 369775236, 481911150, 369775236, 8, 401985209, 47207, 45501803, 0, 411488570, 8, 12}

// TestProvider checks where the values of a case come from: the random
// stream with the reuse of earlier values while generating, and the
// recorded choices fitted to each request while replaying.
func TestProvider(t *testing.T) {
	t.Parallel()

	t.Run("Generate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns earlier values of the same bounds only", func(t *testing.T) {
			t.Parallel()
			wide, narrow := engine.Integer(0, 1_000_000_000), engine.Integer(0, 999_999_999)
			got := make([]int, 0, len(alternatingReuse))
			engine.Generate(func(c *engine.Case) {
				for i := range len(alternatingReuse) {
					g := wide
					if i%2 == 1 {
						g = narrow
					}
					got = append(got, engine.Draw(c, g, drawn))
				}
			}, 42, 0, nil)
			assert.Equal(t, got, alternatingReuse, "the pinned values")
		})

		t.Run("returns draws of a request without reuse from the stream alone", func(t *testing.T) {
			t.Parallel()
			var got [3]uint64
			engine.Generate(func(c *engine.Case) {
				source := c.Rand()
				got = [3]uint64{source.Uint64(), source.Uint64(), source.Uint64()}
			}, 7, 0, nil)
			twin := random.ForCase(7, 0)
			whole := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(math.MaxUint64))
			want := [3]uint64{
				random.Integer(&twin, whole).Magnitude(),
				random.Integer(&twin, whole).Magnitude(),
				random.Integer(&twin, whole).Magnitude(),
			}
			assert.Equal(t, got, want, "three draws without a reuse coin")
		})

		t.Run("returns no earlier value of a choice that a rewind removed, whatever its bounds", func(t *testing.T) {
			t.Parallel()
			wide := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1_000_000_000))
			narrow := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(999_999_999))
			for seed := range uint64(50) {
				attempts := 0
				g := engine.Composite(func(c *engine.Case) uint64 {
					attempts++
					if attempts == 1 {
						return c.Reusable(wide).Magnitude()
					}
					c.Reusable(narrow)
					return c.Reusable(wide).Magnitude()
				}).Filter(func(uint64) bool { return attempts > 1 })
				var got uint64
				engine.Generate(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, seed, 0, nil)
				twin := random.ForCase(seed, 0)
				random.Integer(&twin, wide)
				random.Integer(&twin, narrow)
				assert.Equal(t, got, random.Integer(&twin, wide).Magnitude(),
					"the second attempt's wide value, drawn without a coin, for seed "+strconv.FormatUint(seed, 10))
			}
		})
	})

	t.Run("Replay", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			give     []choice.Choice
			recorded []choice.Choice
		}{
			{
				name:     "returns the recorded choices that fit their requests",
				give:     []choice.Choice{unsigned(7), float(0.5), sequence(1, 0)},
				recorded: []choice.Choice{unsigned(7), float(0.5), sequence(1, 0)},
			},
			{
				name:     "returns each recorded choice fitted to its request's bounds",
				give:     []choice.Choice{unsigned(12), float(math.NaN()), sequence(5, 1, 0, 1)},
				recorded: []choice.Choice{unsigned(0), float(0), sequence(0, 1, 0)},
			},
			{
				name:     "returns the target of a request past the last recorded choice",
				give:     []choice.Choice{unsigned(7)},
				recorded: []choice.Choice{unsigned(7), float(0), sequence()},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				digit, unit := engine.Integer(0, 9), engine.Float(-1.0, 1.0, choice.ExcludeNaN)
				word := engine.StringOver("ab", sizes(t, 0, 3))
				e := engine.Replay(func(c *engine.Case) {
					engine.Draw(c, digit, "digit")
					engine.Draw(c, unit, "unit")
					engine.Draw(c, word, "word")
				}, tt.give, nil)
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded choices")
			})
		}
	})
}
