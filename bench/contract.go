// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"math"
	"runtime"
	"slices"
	"time"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matcher"
)

// [Contract.End] publishes each measurement through [B.ReportMetric]
// under one of these units before it checks any ceiling.
const (
	unitP99Latency  = "p99-ns/op"
	unitMeanLatency = "mean-ns/op"
	unitAllocs      = "allocs/op"
	unitBytes       = "bytes/op"
)

// p99 is the quantile of the iteration durations that
// [Contract.MaxLatency] bounds.
const p99 = 0.99

// unset marks a ceiling that the caller did not state. [Contract.End]
// checks no unset ceiling. The marker is negative because zero is a
// ceiling that a caller can state.
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
// [Contract.End] checks every stated ceiling and reports each one that
// the benchmark exceeded. It checks no ceiling that was not stated.
//
// A Contract is not safe for concurrent use. Call its methods from the
// goroutine that runs the benchmark.
type Contract struct {
	// b is the benchmark being measured.
	b B

	// each contains one duration per iteration. [Contract.End] reads the
	// quantile and the mean from it.
	each []time.Duration
	// started is the time the current iteration began. It is zero before
	// the first iteration.
	started time.Time

	// heapAtStart and bytesAtStart are the allocation counters when the
	// loop began. heapAtEnd and bytesAtEnd are the counters when it
	// ended.
	heapAtStart, bytesAtStart uint64
	heapAtEnd, bytesAtEnd     uint64
	// measuring reports whether the loop is running. The first
	// [Contract.Loop] sets it, and the call that ends the loop clears it.
	measuring bool

	// excluded is the time that [Contract.Excluding] has taken out of
	// the current iteration. Each iteration starts it at zero.
	excluded time.Duration
	// excludedHeap and excludedHeapBytes are the allocations taken out of
	// the whole loop: the work passed to [Contract.Excluding] and the
	// growth of each. They accumulate across iterations, because the
	// counters are read once at each end of the loop.
	excludedHeap, excludedHeapBytes uint64

	// maxLatency, maxMean, maxAllocs and maxBytes are the stated
	// ceilings. Each is unset until the caller states it.
	maxLatency, maxMean time.Duration
	maxAllocs, maxBytes int64
}

// Start returns a contract that measures b and states no ceiling.
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

// MaxLatency states the highest p99 latency per iteration that the
// benchmark may take, and returns the receiver.
//
// A p99 ceiling bounds the tail of the latency distribution, which a
// mean ceiling does not bound. [Contract.MaxMean] states a ceiling on the
// mean.
func (c *Contract) MaxLatency(d time.Duration) *Contract {
	c.maxLatency = d
	return c
}

// MaxMean states the highest mean latency per iteration that the
// benchmark may take, and returns the receiver.
//
// State it together with [Contract.MaxLatency]. A mean ceiling alone
// passes a benchmark whose tail latency grows while its mean does not
// change.
func (c *Contract) MaxMean(d time.Duration) *Contract {
	c.maxMean = d
	return c
}

// MaxAllocs states the most heap allocations per iteration that the
// benchmark may make, and returns the receiver.
//
// [Contract.Loop] reads the runtime's allocation counters before the
// first iteration and after the last. [Contract.End] divides the
// difference by the number of iterations and rounds the quotient down
// before it compares it with the ceiling, as [testing.AllocsPerRun]
// does. It publishes the quotient before rounding.
//
// The count leaves out the work passed to [Contract.Excluding] and the
// contract's own bookkeeping. The counters are process-wide, so the count
// includes the allocations that the runtime makes for itself while the
// loop runs, such as for a garbage collection cycle. Fewer such
// allocations than there are iterations do not change the rounded count.
// A body that does not allocate meets a ceiling of zero in a run of more
// iterations than the runtime made allocations. In a run of one
// iteration, as -benchtime=1x gives, one runtime allocation fails a
// ceiling of zero.
//
// A language implementation of this standard that cannot count
// allocations declares a divergence. It does not approximate the count.
//
// In a build with the race detector, msan or asan, and in one whose
// -gcflags turn off optimisation or inlining, [Contract.End] publishes
// the count and does not check the ceiling, because those builds
// allocate differently from the one that ships.
// [go.dokimi.dev/assert.MaxAllocs] states the same ceiling in a test,
// which the ordinary test run checks.
func (c *Contract) MaxAllocs(n uint64) *Contract {
	c.maxAllocs = int64(n)
	return c
}

// MaxBytes states the most heap bytes per iteration that the benchmark
// may allocate, and returns the receiver.
//
// [Contract.End] counts the bytes over the same window, with the same
// exclusions and the same rounding, as the allocations of
// [Contract.MaxAllocs]. It checks the ceiling in the same builds.
func (c *Contract) MaxBytes(n uint64) *Contract {
	c.maxBytes = int64(n)
	return c
}

// Loop reports whether the benchmark should run another iteration. It
// replaces [testing.B.Loop] in the loop of a benchmark:
//
//	for c.Loop() {
//	    _, _ = store.Get(ctx, id)
//	}
//
// Loop times each iteration for the latency ceilings. It reads the
// allocation counters before the first iteration and after the last, so
// the allocation ceilings count neither the setup before the loop nor
// the code after it.
func (c *Contract) Loop() bool {
	if c.measuring {
		elapsed := time.Since(c.started) - c.excluded
		c.excluded = 0

		// A growth of each is the contract's allocation, not the body's.
		if len(c.each) == cap(c.each) {
			c.excludeHeap(func() { c.each = slices.Grow(c.each, 1) })
		}
		c.each = append(c.each, elapsed)
	} else {
		c.heapAtStart, c.bytesAtStart = heap()
		c.measuring = true
	}

	if !c.b.Loop() {
		c.heapAtEnd, c.bytesAtEnd = heap()
		c.measuring = false
		return false
	}
	c.started = time.Now()
	return true
}

// End publishes what the contract measured and fails the benchmark for
// every ceiling it exceeded.
//
// Call it deferred, so that it runs whatever the benchmark body does. It
// reports each exceeded ceiling through [go.dokimi.dev/assert/expect],
// which records a failure and continues. The output of one run lists
// every ceiling that the benchmark exceeded.
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
	counted := matcher.AllocationsCounted()
	if c.maxAllocs != unset && counted {
		expect.InRange(c.b, math.Floor(allocs), 0, float64(c.maxAllocs),
			"the allocations per iteration stay within their ceiling")
	}
	if c.maxBytes != unset && counted {
		expect.InRange(c.b, math.Floor(bytes), 0, float64(c.maxBytes),
			"the bytes allocated per iteration stay within their ceiling")
	}
}

// perIteration returns the allocations and the bytes per iteration.
// When the body left the loop before [Contract.Loop] reported false, it
// reads the end counters itself.
func (c *Contract) perIteration() (allocs, bytes float64) {
	if c.measuring {
		c.heapAtEnd, c.bytesAtEnd = heap()
	}
	n := float64(len(c.each))

	// The excluded allocations are subtracted, so the result covers the
	// measured work and not the whole loop body.
	return float64(c.heapAtEnd-c.heapAtStart-c.excludedHeap) / n,
		float64(c.bytesAtEnd-c.bytesAtStart-c.excludedHeapBytes) / n
}

// Excluding runs work outside the measurement.
//
// A benchmark whose operation consumes its input builds a fresh input in
// each iteration. Excluding takes the time and the allocations of that
// build out of the iteration, so the ceilings apply to the operation
// alone. The measured work receives the input through a variable that
// the caller declares:
//
//	for c.Loop() {
//	    var store *Store
//	    c.Excluding(func() { store = freshStore() })
//	    store.Settle()
//	}
//
// Before the first [Contract.Loop] and after the last, Excluding runs
// work and takes nothing out, because no iteration is being measured.
func (c *Contract) Excluding(work func()) {
	if !c.measuring {
		work()
		return
	}

	startedAt := time.Now()
	c.excludeHeap(work)
	c.excluded += time.Since(startedAt)
}

// excludeHeap runs work and takes the heap allocations that work makes
// out of the count.
func (c *Contract) excludeHeap(work func()) {
	allocsBefore, bytesBefore := heap()

	work()

	allocsAfter, bytesAfter := heap()
	c.excludedHeap += allocsAfter - allocsBefore
	c.excludedHeapBytes += bytesAfter - bytesBefore
}

// heap returns the process's cumulative count of heap allocations and of
// heap bytes allocated. It reads both through [runtime.ReadMemStats],
// which stops the world.
func heap() (allocs, bytes uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Mallocs, m.TotalAlloc
}

// quantile returns the element of sorted at index (len(sorted)-1)*q,
// rounded down, without interpolation. sorted must be ascending and
// non-empty.
func quantile(sorted []time.Duration, q float64) time.Duration {
	at := int(float64(len(sorted)-1) * q)
	return sorted[at]
}

// total returns the sum of ds.
func total(ds []time.Duration) time.Duration {
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	return sum
}
