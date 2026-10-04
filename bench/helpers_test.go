// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"encoding/json"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/record"
)

// calls is the record of the calls that a benchSeat receives.
type calls = record.Calls

// benchSeat is a fake benchmark. It records what a contract reported and
// keeps the call record of each call. Its Loop runs a fixed number of
// iterations.
//
// It embeds the shared seat instead of implementing Helper, Fatalf and
// Errorf again, and adds the methods of a benchmark.
type benchSeat struct {
	*matchertest.Seat
	calls

	mu        sync.Mutex
	remaining int
	metrics   map[string]float64
}

// newBenchSeat returns a fake benchmark that runs iterations iterations.
func newBenchSeat(iterations int) *benchSeat {
	b := &benchSeat{
		Seat:      &matchertest.Seat{},
		remaining: iterations,
		metrics:   map[string]float64{},
	}
	record.Keep(&b.calls)
	return b
}

// Loop reports whether another iteration runs, and counts it.
func (b *benchSeat) Loop() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.remaining == 0 {
		return false
	}
	b.remaining--
	return true
}

// ReportMetric keeps n under unit.
func (b *benchSeat) ReportMetric(n float64, unit string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.metrics[unit] = n
}

// metric returns the value kept under unit, and whether one was kept.
func (b *benchSeat) metric(unit string) (float64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n, ok := b.metrics[unit]
	return n, ok
}

// verdicts returns the assertion and the verdict of each call record that
// the seat kept, in order, as "assertion verdict".
func (b *benchSeat) verdicts(t *testing.T) []string {
	t.Helper()

	var out []string
	for _, line := range record.Lines(&b.calls) {
		var got struct {
			Assertion string `json:"assertion"`
			Verdict   string `json:"verdict"`
		}
		assert.NoError(t, json.Unmarshal([]byte(line), &got), "the call record is JSON")
		out = append(out, got.Assertion+" "+got.Verdict)
	}
	return out
}
