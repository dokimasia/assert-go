// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"runtime"
	"slices"
	"time"

	"go.dokimi.dev/assert/expect"
)

// The units a contract publishes its measurements under. They appear
// in benchmark output beside the ceiling, so a reader sees what was
// measured and not only that it passed.
const (
	unitP99Latency  = "p99-ns/op"
	unitMeanLatency = "mean-ns/op"
	unitAllocs      = "allocs/op"
	unitBytes       = "bytes/op"
)

// p99 is the quantile [Contract.MaxLatency] holds. The tail is what a
// caller waits for; a mean hides it.
const p99 = 0.99

// unset marks a ceiling nobody stated, so an unstated one is never
// checked. Zero cannot mean that: a ceiling of zero allocations is a
// thing worth stating.
const unset = -1

// Contract measures a benchmark and fails it for exceeding a ceiling.
//
// Build one with [Start], state the ceilings, drive the benchmark with
// [Contract.Loop], and check them with [Contract.End]:
//
//	c := bench.Start(b).MaxLatency(50 * time.Microsecond).MaxAllocs(2)
//	defer c.End()
//
//	for c.Loop() {
//	    _, _ = store.Get(ctx, id)
//	}
//
// A ceiling not stated is not checked. Every ceiling is checked, so
// one run names each one exceeded rather than stopping at the first.
//
// A Contract is not safe for concurrent use. It belongs to the
// goroutine running the benchmark.
type Contract struct {
	// b is the benchmark being measured.
	b B

	// each holds one duration per iteration, which is what the
	// quantile is read from.
	each []time.Duration
	// started is when the current iteration began, zero before the
	// first.
	started time.Time

	// heapAtStart is the allocation count when measuring began.
	heapAtStart, bytesAtStart uint64
	// measuring says whether the counters above have been read.
	measuring bool

	// The stated ceilings, each unset until a caller states it.
	maxLatency, maxMean time.Duration
	maxAllocs, maxBytes int64
}

// Start begins a contract on b.
func Start(b B) *Contract {
	b.Helper()
	return &Contract{
		b:          b,
		maxLatency: unset,
		maxMean:    unset,
		maxAllocs:  unset,
		maxBytes:   unset,
	}
}

// MaxLatency states the highest p99 latency per iteration the
// benchmark may take, and returns the receiver so ceilings chain.
//
// The p99 rather than the mean, because the tail is what a caller
// waits for and a mean hides it. With fewer than a hundred iterations
// the p99 is the slowest one.
func (c *Contract) MaxLatency(d time.Duration) *Contract {
	c.maxLatency = d
	return c
}

// MaxMean states the highest mean latency per iteration the benchmark
// may take, and returns the receiver.
//
// Use it beside [Contract.MaxLatency] rather than instead of it: a
// mean that holds while the tail grows is the regression a mean alone
// misses.
func (c *Contract) MaxMean(d time.Duration) *Contract {
	c.maxMean = d
	return c
}

// MaxAllocs states the most heap allocations per iteration the
// benchmark may make, and returns the receiver.
//
// Measured by reading the runtime's allocation counter before the
// first iteration and after the last, then dividing. That counts every
// allocation the goroutine made, including any the benchmark's own
// bookkeeping caused, so a ceiling of nought is only reachable for a
// body that allocates nothing at all.
//
// Not every language implementing this standard can count
// allocations; those declare a divergence rather than approximate one.
func (c *Contract) MaxAllocs(n uint64) *Contract {
	c.maxAllocs = int64(n)
	return c
}

// MaxBytes states the most heap bytes per iteration the benchmark may
// allocate, and returns the receiver. Measured as
// [Contract.MaxAllocs] is.
func (c *Contract) MaxBytes(n uint64) *Contract {
	c.maxBytes = int64(n)
	return c
}

// Loop reports whether the benchmark should run another iteration, and
// is a drop-in for [testing.B.Loop]:
//
//	for c.Loop() {
//	    _, _ = store.Get(ctx, id)
//	}
//
// It times each iteration on the way past, which is where the latency
// measurements come from, and reads the allocation counters before the
// first iteration so the setup before the loop is not counted.
func (c *Contract) Loop() bool {
	if c.measuring {
		c.each = append(c.each, time.Since(c.started))
	} else {
		c.heapAtStart, c.bytesAtStart = heap()
		c.measuring = true
	}

	if !c.b.Loop() {
		return false
	}
	c.started = time.Now()
	return true
}

// End reports what was measured and fails the benchmark for every
// ceiling exceeded.
//
// Call it deferred, so it runs whatever the benchmark body does.
// Ceilings are reported through the recording surface, so one run
// names each one exceeded rather than stopping at the first.
func (c *Contract) End() {
	c.b.Helper()

	if len(c.each) == 0 {
		return
	}

	allocs, bytes := c.perIteration()
	sorted := slices.Clone(c.each)
	slices.Sort(sorted)

	tail := quantile(sorted, p99)
	mean := total(sorted) / time.Duration(len(sorted))

	c.b.ReportMetric(float64(tail.Nanoseconds()), unitP99Latency)
	c.b.ReportMetric(float64(mean.Nanoseconds()), unitMeanLatency)
	c.b.ReportMetric(allocs, unitAllocs)
	c.b.ReportMetric(bytes, unitBytes)

	if c.maxLatency != unset {
		expect.InRange(c.b, tail.Nanoseconds(), 0, float64(c.maxLatency.Nanoseconds()),
			"the p99 latency per iteration stays within its ceiling")
	}
	if c.maxMean != unset {
		expect.InRange(c.b, mean.Nanoseconds(), 0, float64(c.maxMean.Nanoseconds()),
			"the mean latency per iteration stays within its ceiling")
	}
	if c.maxAllocs != unset {
		expect.InRange(c.b, allocs, 0, float64(c.maxAllocs),
			"the allocations per iteration stay within their ceiling")
	}
	if c.maxBytes != unset {
		expect.InRange(c.b, bytes, 0, float64(c.maxBytes),
			"the bytes allocated per iteration stay within their ceiling")
	}
}

// perIteration answers the allocations and bytes each iteration cost.
func (c *Contract) perIteration() (allocs, bytes float64) {
	heapNow, bytesNow := heap()
	n := float64(len(c.each))

	return float64(heapNow-c.heapAtStart) / n, float64(bytesNow-c.bytesAtStart) / n
}

// heap reads the runtime's cumulative allocation counters.
func heap() (allocs, bytes uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Mallocs, m.TotalAlloc
}

// quantile answers the value at q through sorted, which must be
// ascending and non-empty. With few samples this is the slowest, which
// is the honest answer rather than an interpolated one.
func quantile(sorted []time.Duration, q float64) time.Duration {
	at := int(float64(len(sorted)-1) * q)
	return sorted[at]
}

// total sums durations.
func total(ds []time.Duration) time.Duration {
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	return sum
}
