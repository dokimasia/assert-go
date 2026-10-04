// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
)

// testing.B satisfies B and assert.TB as written: B embeds assert.TB
// instead of declaring a second seat.
var (
	_ bench.B   = (*testing.B)(nil)
	_ assert.TB = (*testing.B)(nil)
)

func TestSeat(t *testing.T) {
	t.Parallel()

	t.Run("B", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the stated number of iterations through Loop", func(t *testing.T) {
			t.Parallel()

			var seat bench.B = newBenchSeat(2)

			ran := 0
			for seat.Loop() {
				ran++
			}
			assert.Equal(t, ran, 2, "Loop runs the stated number of iterations")
		})

		t.Run("keeps the value that ReportMetric receives", func(t *testing.T) {
			t.Parallel()

			seat := newBenchSeat(0)
			var b bench.B = seat
			b.ReportMetric(1.5, "unit")

			got, published := seat.metric("unit")
			assert.True(t, published, "the metric was recorded")
			assert.CloseTo(t, got, 1.5, 0, "the metric is the value it was given")
		})

		t.Run("reports the failure of an assertion that runs on it", func(t *testing.T) {
			t.Parallel()

			seat := newBenchSeat(1)
			assert.Equal(seat, 1, 2, "the benchmark seat reports failures")

			assert.True(t, seat.Failed(),
				"an assertion driven through the benchmark seat reports")
		})
	})
}
