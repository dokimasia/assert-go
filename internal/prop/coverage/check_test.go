// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"math"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/coverage"
)

// The definition's values, pinned here rather than read from the code
// under test.
const (
	// tail is the probability past Z in one tail: QuickCheck's certainty
	// of 10^9, split between the two tails.
	tail = 5e-10
	// pinnedLowerBound is the lower bound of 27 in 100, as the definition's
	// executable reference computes it, so that a reordered expression
	// fails here.
	pinnedLowerBound = 0x1.7bf6bece04433p-4
)

// TestCheck checks the schedule of the checks, the Wilson bound, and the
// verdicts that it gives.
func TestCheck(t *testing.T) {
	t.Parallel()

	t.Run("Checks", func(t *testing.T) {
		t.Parallel()

		t.Run("returns 1, 2, 4 and 8 times the cases setting", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, coverage.Checks(), [4]int{1, 2, 4, 8}, "the multiples of the checks")
		})
	})

	t.Run("Bound", func(t *testing.T) {
		t.Parallel()

		t.Run("uses a Z equal to the normal quantile at 1 - 5e-10", func(t *testing.T) {
			t.Parallel()
			quantile := math.Sqrt2 * math.Erfinv(1-2*tail)
			assert.CloseTo(t, coverage.Z, quantile, 1e-6, "the quantile of the certainty")
		})

		t.Run("returns the pinned lower bound of 27 in 100", func(t *testing.T) {
			t.Parallel()
			got := coverage.Bound(27, 100, -coverage.Z)
			assert.Equal(t, math.Float64bits(got), math.Float64bits(pinnedLowerBound), "the exact double")
		})

		t.Run("returns bounds that enclose the observed share", func(t *testing.T) {
			t.Parallel()
			for _, c := range [][2]int{{0, 10}, {1, 10}, {5, 10}, {10, 10}, {27, 100}, {80, 800}, {1, 1}} {
				k, n := c[0], c[1]
				p := float64(k) / float64(n)
				msg := strconv.Itoa(k) + " of " + strconv.Itoa(n)
				assert.True(t, coverage.Bound(k, n, -coverage.Z) <= p, "the lower bound of "+msg)
				assert.True(t, coverage.Bound(k, n, coverage.Z) >= p, "the upper bound of "+msg)
			}
		})

		panics := []struct {
			name string
			k, n int
		}{
			{name: "panics for no trials", k: 0, n: 0},
			{name: "panics for a negative count", k: -1, n: 10},
			{name: "panics for more successes than trials", k: 11, n: 10},
		}
		for _, tt := range panics {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Panics(t, func() { coverage.Bound(tt.k, tt.n, coverage.Z) }, "counts that are no share")
			})
		}
	})

	t.Run("Decide", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			k, n  int
			share float64
			stage coverage.Stage
			want  coverage.Verdict
		}{
			{
				name:  "returns Met for 27 of 100 against 10%",
				k:     27,
				n:     100,
				share: 0.1,
				stage: coverage.Interim,
				want:  coverage.Met,
			},
			{
				name:  "returns Undecided for 26 of 100 against 10% at an interim check",
				k:     26,
				n:     100,
				share: 0.1,
				stage: coverage.Interim,
				want:  coverage.Undecided,
			},
			{
				name:  "returns Undecided for 10 of 100 against 10% at an interim check",
				k:     10,
				n:     100,
				share: 0.1,
				stage: coverage.Interim,
				want:  coverage.Undecided,
			},
			{
				name:  "returns Refuted for 0 of 100 against 50%",
				k:     0,
				n:     100,
				share: 0.5,
				stage: coverage.Interim,
				want:  coverage.Refuted,
			},
			{
				name:  "returns Undecided for 40 of 100 against 50% at an interim check",
				k:     40,
				n:     100,
				share: 0.5,
				stage: coverage.Interim,
				want:  coverage.Undecided,
			},
			{
				name:  "returns Met for 75 of 800 against 10% at the final check",
				k:     75,
				n:     800,
				share: 0.1,
				stage: coverage.Final,
				want:  coverage.Met,
			},
			{
				name:  "returns Unmet for 70 of 800 against 10% at the final check",
				k:     70,
				n:     800,
				share: 0.1,
				stage: coverage.Final,
				want:  coverage.Unmet,
			},
			{
				name:  "returns Refuted at the final check for a share that the interval refutes",
				k:     0,
				n:     800,
				share: 0.5,
				stage: coverage.Final,
				want:  coverage.Refuted,
			},
			{
				name:  "returns Met for 9 of 10 against 100% on an exhausted run",
				k:     9,
				n:     10,
				share: 1,
				stage: coverage.Exhausted,
				want:  coverage.Met,
			},
			{
				name:  "returns Met for 1 of 6 against 10% on an exhausted run",
				k:     1,
				n:     6,
				share: 0.1,
				stage: coverage.Exhausted,
				want:  coverage.Met,
			},
			{
				name:  "returns Unmet for 0 of 6 against 10% on an exhausted run",
				k:     0,
				n:     6,
				share: 0.1,
				stage: coverage.Exhausted,
				want:  coverage.Unmet,
			},
			{
				name:  "returns Met for 6 of 6 against 100% on an exhausted run",
				k:     6,
				n:     6,
				share: 1,
				stage: coverage.Exhausted,
				want:  coverage.Met,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, coverage.Decide(tt.k, tt.n, tt.share, tt.stage), tt.want, "the verdict")
			})
		}

		t.Run("returns Undecided for an upper bound equal to the share", func(t *testing.T) {
			t.Parallel()
			share := coverage.Bound(3, 100, coverage.Z)
			assert.Equal(t, coverage.Decide(3, 100, share, coverage.Interim), coverage.Undecided, "refuted means below")
		})

		t.Run("returns Met for a lower bound equal to the tolerance times the share", func(t *testing.T) {
			t.Parallel()
			lower := coverage.Bound(27, 100, -coverage.Z)
			guess := lower / coverage.Tolerance
			share, found := 0.0, false
			for _, c := range []float64{guess, math.Nextafter(guess, 0), math.Nextafter(guess, 1)} {
				if coverage.Tolerance*c == lower {
					share, found = c, true
				}
			}
			assert.True(t, found, "a share whose tolerance is the lower bound")
			assert.Equal(t, coverage.Decide(27, 100, share, coverage.Interim), coverage.Met, "met means at least")
		})

		t.Run("panics for no trials on an exhausted run", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { coverage.Decide(0, 0, 0.1, coverage.Exhausted) }, "counts that are no share")
		})
	})
}

// TestCheckZeroAlloc checks that Checks, Bound and Decide allocate
// nothing.
func TestCheckZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = coverage.Checks() }, 0, "Checks allocates nothing")
	assert.MaxAllocs(t, func() { _ = coverage.Bound(27, 100, -coverage.Z) }, 0, "Bound allocates nothing")
	assert.MaxAllocs(t, func() { _ = coverage.Decide(27, 100, 0.1, coverage.Interim) }, 0, "Decide allocates nothing")
}

// BenchmarkCheck measures Checks, Bound and Decide under a ceiling of no
// allocation.
func BenchmarkCheck(b *testing.B) {
	b.Run("Checks", func(b *testing.B) {
		var got [4]int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Checks()
		}
		assert.Equal(b, got[3], 8, "the last check")
	})

	b.Run("Bound", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Bound(27, 100, -coverage.Z)
		}
		assert.Equal(b, math.Float64bits(got), math.Float64bits(pinnedLowerBound), "the pinned bound")
	})

	b.Run("Decide", func(b *testing.B) {
		var got coverage.Verdict
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Decide(10, 100, 0.1, coverage.Interim)
		}
		assert.Equal(b, got, coverage.Undecided, "neither met nor refuted")
	})
}
