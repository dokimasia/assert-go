// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/matcher"
)

// parallelSink keeps what a parallel body allocated, so escape analysis
// cannot remove the allocation. Each goroutine writes its own slot.
var parallelSink [64][]byte

// TestParallel checks a parallel body under a contract, in benchmarks of
// testing that run in this process. It does not run in parallel: it sets
// -benchtime for each run, and the allocation counter covers the whole
// process.
func TestParallel(t *testing.T) {
	t.Run("RunParallel", func(t *testing.T) {
		t.Run("runs b.N iterations in all on GOMAXPROCS goroutines", func(t *testing.T) {
			var goroutines, iterations atomic.Int64
			counted := map[int][2]int64{}
			benchmark(t, "100x", func(b bench.B, n int) {
				goroutines.Store(0)
				iterations.Store(0)
				c := bench.Start(b)
				c.RunParallel(func(pb *bench.PB) {
					goroutines.Add(1)
					for pb.Next() {
						iterations.Add(1)
					}
				})
				c.End()
				counted[n] = [2]int64{goroutines.Load(), iterations.Load()}
			})

			procs := int64(runtime.GOMAXPROCS(0))
			assert.Equal(t, counted, map[int][2]int64{1: {procs, 1}, 100: {procs, 100}},
				"each run starts one goroutine per GOMAXPROCS and runs b.N iterations in all")
		})

		t.Run("leaves the goroutines and the handles out of the allocation count", func(t *testing.T) {
			result, seen := benchmark(t, "1x", func(b bench.B, _ int) {
				c := bench.Start(b).MaxAllocs(0).MaxBytes(0)
				c.RunParallel(spin)
				c.End()
			})

			assert.Equal(t, seen, []benchCall{{n: 1}},
				"a body that allocates nothing meets ceilings of zero in a run of one iteration")
			assert.Equal(t, result.Extra["allocs/op"], 0.0, "the run publishes no allocation")
		})

		t.Run("reports allocations above their ceiling in the run that testing reports", func(t *testing.T) {
			_, seen := benchmark(t, "100x", func(b bench.B, _ int) {
				var slot atomic.Int64
				c := bench.Start(b).MaxAllocs(0)
				c.RunParallel(func(pb *bench.PB) {
					i := slot.Add(1) - 1
					for pb.Next() {
						parallelSink[i] = make([]byte, 64)
					}
				})
				c.End()
			})

			assert.Equal(t, seen[len(seen)-1].failed, matcher.AllocationsCounted(),
				"an allocation in every iteration exceeds a ceiling of zero in a build that checks it")
		})

		t.Run("checks no ceiling in a run that testing does not report", func(t *testing.T) {
			var once sync.Once
			_, seen := benchmark(t, "100x", func(b bench.B, _ int) {
				c := bench.Start(b).MaxAllocs(0)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						once.Do(func() { parallelSink[0] = make([]byte, 64) })
					}
				})
				c.End()
			})

			assert.Equal(t, seen, []benchCall{{n: 1}, {n: 100}},
				"testing reports the run of 100 after a run of 1, and the allocation of the first fails nothing")
		})

		t.Run("checks the ceilings of the one run under -benchtime=1x", func(t *testing.T) {
			var once sync.Once
			_, seen := benchmark(t, "1x", func(b bench.B, _ int) {
				c := bench.Start(b).MaxAllocs(0)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						once.Do(func() { parallelSink[0] = make([]byte, 64) })
					}
				})
				c.End()
			})

			assert.Equal(t, seen, []benchCall{{n: 1, failed: matcher.AllocationsCounted()}},
				"the one run fails on its allocation in a build that checks the ceiling")
		})

		t.Run("checks the ceilings of the run that takes a -benchtime of a duration", func(t *testing.T) {
			var once sync.Once
			_, seen := benchmark(t, "10ms", func(b bench.B, _ int) {
				c := bench.Start(b).MaxAllocs(0)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						once.Do(func() { parallelSink[0] = make([]byte, 64) })
						time.Sleep(time.Microsecond)
					}
				})
				c.End()
			})

			assert.True(t, len(seen) > 1, "testing runs the benchmark more than once before a run takes 10 ms")
			assert.False(t, seen[len(seen)-1].failed, "only the first run allocates, and testing does not report it")
		})

		t.Run("runs the warm-up on the goroutines before the measurement", func(t *testing.T) {
			var warmed atomic.Int64
			_, seen := benchmark(t, "1x", func(b bench.B, _ int) {
				c := bench.Start(b).Warmup(8).MaxAllocs(0)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						if warmed.Add(1) == 1 {
							parallelSink[0] = make([]byte, 64)
						}
					}
				})
				c.End()
			})

			assert.Equal(t, warmed.Load(), int64(9), "8 warm-up iterations and the one measured iteration")
			assert.Equal(t, seen, []benchCall{{n: 1}}, "a warm-up iteration's allocation counts against no ceiling")
		})

		t.Run("leaves what a body does after its last iteration out of every count", func(t *testing.T) {
			result, seen := benchmark(t, "1x", func(b bench.B, _ int) {
				var slot atomic.Int64
				c := bench.Start(b).MaxBytes(1024)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
					}
					parallelSink[slot.Add(1)-1] = make([]byte, 1<<20)
					time.Sleep(20 * time.Millisecond)
				})
				c.End()
			})

			assert.Equal(t, seen, []benchCall{{n: 1}}, "a MiB after the last iteration counts against no ceiling")
			assert.True(t, result.Extra["bytes/op"] < 1024, "the contract publishes no byte of it")
			assert.True(t, result.NsPerOp() < int64(20*time.Millisecond), "testing's time leaves out the sleep")
		})

		t.Run("leaves the warm-up out of testing's time", func(t *testing.T) {
			var warmed atomic.Int64
			result, _ := benchmark(t, "1x", func(b bench.B, _ int) {
				c := bench.Start(b).Warmup(1)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						if warmed.Add(1) == 1 {
							time.Sleep(20 * time.Millisecond)
						}
					}
				})
				c.End()
			})

			assert.True(
				t,
				result.NsPerOp() < int64(20*time.Millisecond),
				"testing's time leaves out the warm-up's sleep",
			)
		})

		t.Run("publishes the latency and the counts of the run", func(t *testing.T) {
			result, _ := benchmark(t, "100x", func(b bench.B, _ int) {
				c := bench.Start(b)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						time.Sleep(time.Millisecond)
					}
				})
				c.End()
			})

			for _, unit := range []string{"p99-ns/op", "mean-ns/op", "allocs/op", "bytes/op"} {
				_, published := result.Extra[unit]
				assert.True(t, published, "the contract publishes "+unit)
			}
			assert.True(t, result.Extra["mean-ns/op"] >= float64(time.Millisecond),
				"the mean latency of an iteration is its own time, a millisecond or more")
		})

		t.Run("reports a latency above its ceiling", func(t *testing.T) {
			_, seen := benchmark(t, "4x", func(b bench.B, _ int) {
				c := bench.Start(b).MaxLatency(time.Microsecond)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						time.Sleep(time.Millisecond)
					}
				})
				c.End()
			})

			assert.True(t, seen[len(seen)-1].failed, "iterations of a millisecond exceed a p99 of a microsecond")
		})

		t.Run("raises the panic of a body on the calling goroutine", func(t *testing.T) {
			var recovered any
			benchmark(t, "1x", func(b bench.B, _ int) {
				defer func() { recovered = recover() }()
				c := bench.Start(b)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
						panic("the body broke")
					}
				})
			})

			assert.Equal(t, recovered, any("the body broke"), "RunParallel panics with the body's panic")
		})

		t.Run("publishes nothing when a body ends its goroutine with Goexit", func(t *testing.T) {
			result, _ := benchmark(t, "1x", func(b bench.B, _ int) {
				c := bench.Start(b).MaxLatency(time.Nanosecond)
				c.RunParallel(func(pb *bench.PB) {
					pb.Next()
					runtime.Goexit()
				})
				c.End()
			})

			_, published := result.Extra["p99-ns/op"]
			assert.False(t, published, "a run that a body ended publishes nothing")
		})

		t.Run("panics when a body returns before Next reports false", func(t *testing.T) {
			var recovered any
			benchmark(t, "100x", func(b bench.B, _ int) {
				defer func() { recovered = recover() }()
				c := bench.Start(b)
				c.RunParallel(func(pb *bench.PB) { pb.Next() })
			})

			assert.Equal(t, recovered, any("bench: RunParallel's body returned before Next reported false"),
				"the message names the call")
		})

		t.Run("panics without waiting for a body that returns before its first call of Next", func(t *testing.T) {
			var recovered any
			benchmark(t, "1x", func(b bench.B, _ int) {
				defer func() { recovered = recover() }()
				c := bench.Start(b)
				c.RunParallel(func(*bench.PB) {})
			})

			assert.Equal(t, recovered, any("bench: RunParallel's body returned before Next reported false"),
				"the message names the call")
		})

		t.Run("panics for a benchmark that is no *testing.B", func(t *testing.T) {
			c := bench.Start(newBenchSeat(1))
			raised := assert.Panics(t, func() { c.RunParallel(spin) }, "a fake benchmark runs no parallel body")
			assert.Equal(t, raised, any("bench: RunParallel of a benchmark that is no *testing.B"),
				"the message names the call")
		})

		t.Run("panics after Loop has run", func(t *testing.T) {
			var recovered any
			benchmark(t, "1x", func(b bench.B, _ int) {
				c := bench.Start(b)
				for c.Loop() {
				}
				func() {
					defer func() { recovered = recover() }()
					c.RunParallel(spin)
				}()
			})

			assert.Equal(t, recovered, any("bench: RunParallel after the contract has run its body"),
				"the message names the call")
		})
	})

	t.Run("Next", func(t *testing.T) {
		t.Run("reports false again once it has reported false", func(t *testing.T) {
			var again atomic.Bool
			benchmark(t, "1x", func(b bench.B, _ int) {
				c := bench.Start(b)
				c.RunParallel(func(pb *bench.PB) {
					for pb.Next() {
					}
					if pb.Next() {
						again.Store(true)
					}
				})
				c.End()
			})

			assert.False(t, again.Load(), "no goroutine runs another iteration")
		})
	})
}

// BenchmarkParallel measures PB.Next under RunParallel, which allocates
// nothing per iteration.
func BenchmarkParallel(b *testing.B) {
	c := bench.Start(b).MaxAllocs(0)
	defer c.End()

	c.RunParallel(spin)
}
