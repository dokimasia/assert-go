// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"encoding/json"
	"flag"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/record"
)

// benchtimeFlag is the flag of testing that states how long a benchmark
// runs.
const benchtimeFlag = "test.benchtime"

// calls is the record of the calls that a benchSeat receives.
type calls = record.Calls

// benchCall is what one call of a benchmark function of testing saw: its
// b.N, and whether the benchmark had failed when the call returned.
type benchCall struct {
	n      int
	failed bool
}

// benchmark runs body as the function of a benchmark of testing under
// -benchtime=stated, and restores the flag after the run. body measures the
// benchmark b, whose b.N is n. benchmark returns the benchmark's result and
// what each call of body saw. The flag belongs to the process, so a test
// that calls benchmark does not run in parallel.
func benchmark(t *testing.T, stated string, body func(b bench.B, n int)) (testing.BenchmarkResult, []benchCall) {
	t.Helper()

	before := flag.Lookup(benchtimeFlag).Value.String()
	assert.NoError(t, flag.Set(benchtimeFlag, stated), "-benchtime takes "+stated)
	defer func() { assert.NoError(t, flag.Set(benchtimeFlag, before), "-benchtime takes its value back") }()
	var seen []benchCall
	result := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		body(b, b.N)
		seen = append(seen, benchCall{n: b.N, failed: b.Failed()})
	})
	return result, seen
}

// spin is a parallel body whose iterations do nothing.
func spin(pb *bench.PB) {
	for pb.Next() {
	}
}

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
