// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/matchertest"
)

// testing.B satisfies the seam as written, which is the point of
// embedding the seat rather than declaring a second one.
var (
	_ bench.B   = (*testing.B)(nil)
	_ assert.TB = (*testing.B)(nil)
)

// benchSeat is a benchmark stand-in: it records what a contract
// reported, and runs a fixed number of iterations.
//
// It embeds the shared seat rather than reimplementing the failure
// surface, and adds only what a benchmark has.
type benchSeat struct {
	*matchertest.Seat

	mu        sync.Mutex
	remaining int
	metrics   map[string]float64
}

func newBenchSeat(iterations int) *benchSeat {
	return &benchSeat{
		Seat:      &matchertest.Seat{},
		remaining: iterations,
		metrics:   map[string]float64{},
	}
}

func (b *benchSeat) Loop() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.remaining == 0 {
		return false
	}
	b.remaining--
	return true
}

func (b *benchSeat) ReportMetric(n float64, unit string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.metrics[unit] = n
}

func (b *benchSeat) metric(unit string) (float64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n, ok := b.metrics[unit]
	return n, ok
}

func TestSeat(t *testing.T) {
	t.Parallel()

	t.Run("B", func(t *testing.T) {
		t.Parallel()

		t.Run("Loop bounds the iterations", func(t *testing.T) {
			t.Parallel()

			var seat bench.B = newBenchSeat(2)

			ran := 0
			for seat.Loop() {
				ran++
			}
			assert.Equal(t, ran, 2, "Loop runs the stated number of iterations")
		})

		t.Run("ReportMetric records what it is given", func(t *testing.T) {
			t.Parallel()

			seat := newBenchSeat(0)
			var b bench.B = seat
			b.ReportMetric(1.5, "unit")

			got, published := seat.metric("unit")
			assert.True(t, published, "the metric was recorded")
			assert.CloseTo(t, got, 1.5, 0, "the metric holds the value it was given")
		})

		t.Run("it carries the seat every assertion reports through", func(t *testing.T) {
			t.Parallel()

			seat := newBenchSeat(1)
			assert.Equal(seat, 1, 2, "the benchmark seat reports failures")

			assert.True(t, seat.Failed(),
				"an assertion driven through the benchmark seat reports")
		})
	})
}
