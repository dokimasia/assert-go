// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"testing"
	"time"

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

// The fields of the detail of a ceiling's record.
const (
	// wantField is the ceiling that the caller stated.
	wantField = "want"
	// gotField is the measurement that exceeded the ceiling.
	gotField = "got"
)

// p99 is the quantile of the iteration durations that
// [Contract.MaxLatency] bounds.
const p99 = 0.99

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
// [Contract.Warmup] states iterations that run before the measurement,
// and [Contract.RunParallel] measures a body that runs on GOMAXPROCS
// goroutines at once in place of the loop.
//
// A Contract is not safe for concurrent use. Call its methods from the
// goroutine that runs the benchmark.
type Contract struct {
	// b is the benchmark being measured.
	b B

	// warmup is the number of warm-up iterations that the contract runs
	// before the first measured one, and warmed the number that Loop has
	// run.
	warmup, warmed int

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

	// excluded is the time that [Contract.Excluding] has taken out of
	// the current iteration. Each iteration starts it at zero.
	excluded time.Duration
	// excludedHeap and excludedHeapBytes are the allocations taken out of
	// the whole loop: the work passed to [Contract.Excluding] and the
	// growth of each. They accumulate across iterations, because the
	// counters are read once at each end of the loop.
	excludedHeap, excludedHeapBytes uint64

	// maxLatency, maxMean, maxAllocs and maxBytes are the stated
	// ceilings.
	maxLatency, maxMean time.Duration
	maxAllocs, maxBytes uint64

	// ran reports whether Loop or RunParallel has run, and parallel whether
	// RunParallel has.
	ran, parallel bool
	// measuring reports whether the loop is running. The first
	// [Contract.Loop] sets it, and the call that ends the loop clears it.
	measuring bool
	// excluding reports whether [Contract.Excluding] runs its work now.
	excluding bool
	// latencyStated, meanStated, allocsStated and bytesStated report which
	// ceilings the caller stated. [Contract.End] checks no other.
	latencyStated, meanStated, allocsStated, bytesStated bool
}

// Start returns a contract that measures b and states no ceiling.
//
// # Allocation contract
//
// Start allocates the contract: one allocation.
func Start(b B) *Contract {
	b.Helper()
	return &Contract{b: b}
}

// MaxLatency states the highest p99 latency per iteration that the
// benchmark may take, and returns the receiver.
//
// A p99 ceiling bounds the tail of the latency distribution, which a
// mean ceiling does not bound. [Contract.MaxMean] states a ceiling on the
// mean.
//
// # Allocation contract
//
// MaxLatency allocates nothing.
func (c *Contract) MaxLatency(d time.Duration) *Contract {
	c.maxLatency, c.latencyStated = d, true
	return c
}

// MaxMean states the highest mean latency per iteration that the
// benchmark may take, and returns the receiver.
//
// State it together with [Contract.MaxLatency]. A mean ceiling alone
// passes a benchmark whose tail latency grows while its mean does not
// change.
//
// # Allocation contract
//
// MaxMean allocates nothing.
func (c *Contract) MaxMean(d time.Duration) *Contract {
	c.maxMean, c.meanStated = d, true
	return c
}

// MaxAllocs states the most heap allocations per iteration that the
// benchmark may make, and returns the receiver.
//
// [Contract.Loop] reads the runtime's allocation counters before the
// first iteration and after the last. [Contract.End] divides the
// difference by the number of iterations and rounds the quotient to the
// nearest whole number, a half up, before it compares it with the
// ceiling. It publishes the quotient before rounding. A body that
// allocates past the ceiling on half of its iterations or more fails it,
// so a value that a pool's stock serves on one iteration and that leaks
// on every other fails a ceiling of zero.
//
// The count leaves out the work passed to [Contract.Excluding] and the
// contract's own bookkeeping. The counters are process-wide, so the count
// includes the allocations that the runtime makes for itself while the
// loop runs, such as for a garbage collection cycle. Fewer such
// allocations than half the iterations do not change the rounded count.
// In a run of one iteration, as -benchtime=1x gives, one runtime
// allocation fails a ceiling of zero.
//
// A language implementation of this standard that cannot count
// allocations declares a divergence. It does not approximate the count.
//
// In a build with the race detector, msan or asan, in one whose -gcflags
// turn off optimisation or inlining, and in a test binary that a mutation
// run instrumented, [Contract.End] publishes the count and does not check
// the ceiling, because those builds allocate differently from a
// production build.
// [go.dokimi.dev/assert.MaxAllocs] states the same ceiling in a test,
// which the ordinary test run checks.
//
// # Allocation contract
//
// MaxAllocs allocates nothing.
func (c *Contract) MaxAllocs(n uint64) *Contract {
	c.maxAllocs, c.allocsStated = n, true
	return c
}

// Warmup states n iterations that run before the first measured one, and
// returns the receiver. [Contract.Loop] returns true n times before it
// times an iteration, reads a counter or calls testing.B.Loop, so testing's
// own ns/op and allocs/op leave the warm-up out as well. A warm-up
// iteration runs the whole body: [Contract.Excluding] runs its work and
// takes nothing out. No ceiling covers a warm-up iteration, and
// [Contract.End] publishes nothing about one. Under -benchtime=1x the
// benchmark runs n warm-up iterations and one measured iteration.
//
// A warm-up leaves out what the first iterations build, such as the cache
// of an interface conversion, which the runtime allocates once for each
// conversion site after a random number of its executions. An iteration
// that executes the site many times builds the cache within one warm-up
// iteration, and one that executes it once needs about 20,000.
//
// [Contract.RunParallel] runs the warm-up on its goroutines, n iterations
// in all.
//
// # Panics
//
// Warmup panics for a negative n, and after [Contract.Loop] or
// [Contract.RunParallel] has run.
//
// # Allocation contract
//
// Warmup allocates nothing.
func (c *Contract) Warmup(n int) *Contract {
	if n < 0 {
		panic(fmt.Sprintf("bench: Warmup(%d) states a negative number of iterations", n))
	}
	if c.ran {
		panic("bench: Warmup after the contract has run its body")
	}
	c.warmup = n
	return c
}

// MaxBytes states the most heap bytes per iteration that the benchmark
// may allocate, and returns the receiver.
//
// [Contract.End] counts the bytes over the same window, with the same
// exclusions, as the allocations of [Contract.MaxAllocs], and rounds the
// bytes per iteration down. Rounding down drops less than one byte per
// iteration, so a value that leaks on nearly every iteration still adds
// its size. It checks the ceiling in the same builds.
//
// # Allocation contract
//
// MaxBytes allocates nothing.
func (c *Contract) MaxBytes(n uint64) *Contract {
	c.maxBytes, c.bytesStated = n, true
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
// allocation counters before the first measured iteration and after the
// last, so the allocation ceilings count neither the setup before the
// loop, nor the iterations of [Contract.Warmup], nor the code after it.
//
// # Panics
//
// Loop panics after [Contract.RunParallel] has run, because a benchmark
// function measures either a loop or a parallel body.
//
// # Allocation contract
//
// Loop allocates nothing per iteration, amortized over the growth of the
// durations that it keeps. It takes that growth out of the count of the
// allocation ceilings.
func (c *Contract) Loop() bool {
	if c.parallel {
		panic("bench: Loop after RunParallel")
	}
	c.ran = true
	if c.warmed < c.warmup {
		c.warmed++
		return true
	}
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
// Call it deferred, so that it runs whatever the benchmark body does. A
// body that leaves the loop before [Contract.Loop] reports false, as a
// break or a return does, ends its iteration there: End counts the
// iteration's time and the allocations up to that point. End reports each
// exceeded ceiling as a record of its assertion, such as
// bench-max-latency, with the ceiling as want and the measurement as got,
// through Errorf, which records a failure and continues. The output of
// one run lists every ceiling that the benchmark exceeded. Each other
// stated ceiling passes, an allocation ceiling that the build does not
// check included. A run of no iteration publishes nothing and checks no
// ceiling.
//
// After [Contract.RunParallel], End checks the ceilings of the run that
// testing reports and of no earlier one, as RunParallel states.
//
// # Allocation contract
//
// End allocates a sorted copy of the durations of the iterations: one
// allocation. A ceiling that the benchmark exceeded allocates its record
// as well.
func (c *Contract) End() {
	c.b.Helper()

	if c.measuring {
		elapsed := time.Since(c.started) - c.excluded
		c.heapAtEnd, c.bytesAtEnd = heap()
		c.measuring = false
		c.each = append(c.each, elapsed)
	}
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

	if c.parallel && !lastRun(c.b.(*testing.B)) {
		return
	}
	if c.latencyStated {
		c.check("bench-max-latency", "the p99 latency per iteration is within its ceiling",
			//dokimi:mutate-skip ror-boundary: no test can make the wall clock time a p99 at its ceiling exactly
			tail > c.maxLatency, map[string]any{wantField: c.maxLatency, gotField: tail})
	}
	if c.meanStated {
		c.check("bench-max-mean", "the mean latency per iteration is within its ceiling",
			//dokimi:mutate-skip ror-boundary: no test can make the wall clock time a mean at its ceiling exactly
			mean > c.maxMean, map[string]any{wantField: c.maxMean, gotField: mean})
	}
	counted := matcher.AllocationsCounted()
	if c.allocsStated {
		rounded := math.Round(allocs)
		c.check("bench-max-allocs", "the allocations per iteration are within their ceiling",
			counted && rounded > float64(c.maxAllocs),
			map[string]any{wantField: c.maxAllocs, gotField: uint64(rounded)})
	}
	if c.bytesStated {
		c.check("bench-max-bytes", "the bytes allocated per iteration are within their ceiling",
			counted && math.Floor(bytes) > float64(c.maxBytes),
			map[string]any{wantField: c.maxBytes, gotField: uint64(bytes)})
	}
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
// work and takes nothing out, because no iteration is being measured. A
// call inside the work of another call runs its work and takes nothing out
// either, because the other call takes out all of its own work.
//
// # Allocation contract
//
// Excluding allocates nothing besides what work allocates.
func (c *Contract) Excluding(work func()) {
	if !c.measuring || c.excluding {
		work()
		return
	}

	c.excluding = true
	defer func() { c.excluding = false }()
	startedAt := time.Now()
	c.excludeHeap(work)
	c.excluded += time.Since(startedAt)
}

// check reports the verdict of the ceiling of assertion: a failure with
// detail when the measurement exceeded the ceiling, and a pass otherwise.
func (c *Contract) check(assertion, contract string, exceeded bool, detail map[string]any) {
	c.b.Helper()
	if exceeded {
		matcher.Fail(c.b, matcher.Soft, assertion, contract, detail)
		return
	}
	matcher.Pass(c.b, matcher.Soft, assertion, contract)
}

// perIteration returns the allocations and the bytes per iteration, over
// the counters that the loop's end read.
func (c *Contract) perIteration() (allocs, bytes float64) {
	n := float64(len(c.each))

	// The excluded allocations are subtracted, so the result covers the
	// measured work and not the whole loop body.
	return float64(c.heapAtEnd-c.heapAtStart-c.excludedHeap) / n,
		float64(c.bytesAtEnd-c.bytesAtStart-c.excludedHeapBytes) / n
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
