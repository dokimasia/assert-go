// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"flag"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The grains in which the goroutines of a parallel body take their
// iterations: about a hundredth of each goroutine's share of the run, and
// from 1 to 10,000 iterations, so the goroutines finish together and the
// shared count is seldom contended.
const (
	grainsPerGoroutine = 100
	maxGrain           = 10_000
)

// maxRunIterations is the number of iterations at which testing starts no
// further run of a benchmark, whatever the run's time.
const maxRunIterations = 1_000_000_000

// benchtimeFlag is the flag that states how long testing runs a benchmark,
// as a duration or as a count of iterations with the suffix x.
const benchtimeFlag = "test.benchtime"

// RunParallel runs body on GOMAXPROCS goroutines, which run b.N
// iterations in all, and measures every iteration. Each goroutine
// receives a [PB] of its own, and runs iterations while [PB.Next] reports
// true, as a body of testing.B.RunParallel does:
//
//	c := bench.Start(b).MaxLatency(50 * time.Microsecond).MaxAllocs(0)
//	defer c.End()
//
//	c.RunParallel(func(pb *bench.PB) {
//	    for pb.Next() {
//	        _, _ = store.Get(ctx, id)
//	    }
//	})
//
// The measurement starts once every goroutine has called Next for its
// first measured iteration, and ends once Next has reported false on every
// goroutine, before any goroutine runs on. The ceilings count neither the
// goroutines, nor the handles, nor what a body does before its first
// iteration or after its last. The goroutines run the iterations of
// [Contract.Warmup] before the measurement, its count in all. Each
// iteration is timed on its goroutine, from one call of Next to the next.
// [Contract.MaxLatency] bounds the p99 of the durations of every iteration
// and [Contract.MaxMean] their mean. [Contract.MaxAllocs] and
// [Contract.MaxBytes] bound the allocations and the bytes of the
// measurement, divided by b.N: the allocations rounded to the nearest
// whole number, and the bytes rounded down. RunParallel resets testing's
// timer at the start of the measurement and stops it at the end, so
// testing's own ns/op is the time of the measurement divided by b.N, about
// the mean latency divided by GOMAXPROCS, and its allocs/op counts the
// measurement as well.
//
// RunParallel runs in a benchmark that calls testing.B.Loop nowhere, as
// testing.B.RunParallel does, so testing calls the benchmark function once
// for each b.N that it tries, and reports the last call. [Contract.End]
// publishes what every call measured and checks the ceilings of the last
// call alone: the call after which testing starts no further one, by
// testing's count and time. Under -benchtime=Nx the last call ran N
// iterations. Otherwise it took -benchtime or longer, or ran 1,000,000,000
// iterations. End stops the timer before it applies the rule. A call that
// fails ends the calls as well, and testing reports no measurement of it,
// so End checks its ceilings only where the rule makes it the last call.
// RunParallel calls testing.Init, which registers testing's flags in a
// program that runs benchmarks without go test, so it can read
// -benchtime.
//
// The goroutines take their iterations from one shared count, so a
// quicker one runs more of them. testing.B.SetParallelism does not change
// the number of goroutines. A body that builds an input for each
// iteration builds them before RunParallel, and hands them out by an
// index that the goroutines share.
//
// A body that panics ends the run: RunParallel waits for every goroutine,
// and panics again on the calling goroutine with the panic of the
// lowest-numbered goroutine that panicked. A body that ends its goroutine
// with runtime.Goexit, as FailNow does, ends the run as well, and End then
// publishes nothing and checks no ceiling.
//
// # Panics
//
// RunParallel panics for a benchmark that is no *testing.B, after
// [Contract.Loop] or RunParallel has run, and when a body returns before
// [PB.Next] reports false.
//
// # Allocation contract
//
// RunParallel allocates one duration for each of the b.N iterations, a
// handle for each goroutine, and the goroutines, before the measurement
// starts. [PB.Next] allocates nothing.
func (c *Contract) RunParallel(body func(*PB)) {
	b, ok := c.b.(*testing.B)
	if !ok {
		panic("bench: RunParallel of a benchmark that is no *testing.B")
	}
	if c.ran {
		panic("bench: RunParallel after the contract has run its body")
	}
	c.ran, c.parallel = true, true

	r := newParallelRun(b.N, c.warmup, runtime.GOMAXPROCS(0))
	for i := range r.pbs {
		go r.work(i, body)
	}
	r.ready.Wait()
	b.ResetTimer()
	c.heapAtStart, c.bytesAtStart = heap()
	close(r.release)
	r.left.Wait()
	b.StopTimer()
	c.heapAtEnd, c.bytesAtEnd = heap()
	close(r.measured)
	r.done.Wait()

	for i := range r.pbs {
		if r.panics[i] != nil {
			panic(r.panics[i])
		}
	}
	for i := range r.pbs {
		pb := &r.pbs[i]
		if !pb.returned {
			return
		}
		if !pb.finished {
			panic("bench: RunParallel's body returned before Next reported false")
		}
	}
	c.each = r.durations
}

// PB hands out the iterations of one goroutine of a parallel body, and
// times each one. [Contract.RunParallel] gives each goroutine its own.
//
// A PB is not safe for concurrent use. Call Next from the goroutine that
// received it.
type PB struct {
	// run is the parallel run that the goroutine belongs to.
	run *parallelRun

	// arrived reports whether the goroutine has passed the start of the
	// measurement, and finished whether Next has reported false.
	arrived, finished bool
	// returned reports whether the goroutine's body returned.
	returned bool

	// timing reports whether an iteration runs, which index is, and which
	// started at started.
	timing  bool
	index   int64
	started time.Time
	// next and end bound the iterations of the grain that the goroutine
	// took last: next is the first that it has not run.
	next, end int64
}

// Next reports whether the goroutine runs another iteration. Each call
// ends the iteration that the call before it started. The first calls run
// the iterations of [Contract.Warmup], untimed, while any remain. The first
// call after them waits until every goroutine has made the same call, and
// then starts the measurement. The call that first reports false waits
// until Next has reported false on every goroutine, and the measurement
// has ended. Once Next has reported false, it reports false again.
//
// # Allocation contract
//
// Next allocates nothing.
func (pb *PB) Next() bool {
	now := time.Now()
	r := pb.run
	if pb.timing {
		r.durations[pb.index] = now.Sub(pb.started)
		pb.timing = false
	}
	if pb.finished {
		return false
	}
	if !pb.arrived {
		if r.warmup.Add(-1) >= 0 {
			return true
		}
		pb.arrive()
		now = time.Now()
	}
	if pb.next == pb.end {
		end := r.taken.Add(r.grain)
		start := end - r.grain
		if start >= r.n {
			pb.finished = true
			r.left.Done()
			<-r.measured
			return false
		}
		pb.next, pb.end = start, min(end, r.n)
	}
	pb.index = pb.next
	pb.next++
	pb.timing, pb.started = true, now
	return true
}

// arrive marks the goroutine as ready to measure, and waits until every
// goroutine is.
func (pb *PB) arrive() {
	pb.arrived = true
	pb.run.ready.Done()
	<-pb.run.release
}

// parallelRun is one run of a parallel body: its iterations, its
// goroutines' handles, and the barriers at both ends of the measurement.
type parallelRun struct {
	// n is the number of measured iterations, and grain the number that a
	// goroutine takes at once.
	n, grain int64
	// taken counts the measured iterations that the goroutines have taken,
	// and warmup the warm-up iterations that remain.
	taken, warmup atomic.Int64
	// durations is the duration of each measured iteration, by its index.
	durations []time.Duration
	// pbs are the goroutines' handles, and panics what each goroutine's body
	// panicked with.
	pbs    []PB
	panics []any
	// ready counts the goroutines that have not arrived at the start of the
	// measurement, and release starts it. left counts the goroutines whose
	// Next has not reported false, and measured ends the measurement. done
	// counts the goroutines that run.
	ready    sync.WaitGroup
	release  chan struct{}
	left     sync.WaitGroup
	measured chan struct{}
	done     sync.WaitGroup
}

// newParallelRun returns a run of n measured iterations, after warmup
// warm-up iterations, on the given number of goroutines.
func newParallelRun(n, warmup, goroutines int) *parallelRun {
	r := &parallelRun{
		n:         int64(n),
		grain:     int64(min(max(n/(goroutines*grainsPerGoroutine), 1), maxGrain)),
		durations: make([]time.Duration, n),
		pbs:       make([]PB, goroutines),
		panics:    make([]any, goroutines),
		release:   make(chan struct{}),
		measured:  make(chan struct{}),
	}
	r.warmup.Store(int64(warmup))
	r.ready.Add(goroutines)
	r.left.Add(goroutines)
	r.done.Add(goroutines)
	for i := range r.pbs {
		r.pbs[i].run = r
	}
	return r
}

// work runs body on the handle of goroutine i. A body that ends before it
// arrives at the start of the measurement counts as arrived, and one that
// ends before its Next reports false counts as having left it, so the other
// goroutines do not wait for it.
func (r *parallelRun) work(i int, body func(*PB)) {
	pb := &r.pbs[i]
	defer r.done.Done()
	defer func() {
		r.panics[i] = recover()
		if !pb.arrived {
			pb.arrived = true
			r.ready.Done()
		}
		if !pb.finished {
			r.left.Done()
		}
	}()
	body(pb)
	pb.returned = true
}

// lastRun reports whether the run of b that ends now is the run after
// which testing starts no further one, by its count and its time. It stops
// b's timer, so the run's time is the time that testing compares with
// -benchtime.
func lastRun(b *testing.B) bool {
	b.Helper()
	b.StopTimer()
	count, d := benchtime()
	if count > 0 {
		return b.N == count
	}
	return b.Elapsed() >= d || b.N >= maxRunIterations
}

// benchtime returns the count of iterations or the time that -benchtime
// states, as testing reads it. testing.Init registers the flag in a
// program that runs benchmarks without go test, and does nothing in a test.
func benchtime() (count int, d time.Duration) {
	testing.Init()
	stated := flag.Lookup(benchtimeFlag).Value.String()
	if n, isCount := strings.CutSuffix(stated, "x"); isCount {
		count, _ = strconv.Atoi(n)
		return count, 0
	}
	d, _ = time.ParseDuration(stated)
	return 0, d
}
