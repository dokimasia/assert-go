// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"path/filepath"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// This package consumes the library, so its tests use the library's own
// assertions.

// iterations is the number of iterations the fake benchmark runs. At 100
// iterations the p99 is a different sample from the slowest one.
const iterations = 100

// run drives a contract over body and returns the fake benchmark, which
// records what the contract reported.
func run(iterations int, state func(*bench.Contract) *bench.Contract, body func()) *benchSeat {
	seat := newBenchSeat(iterations)

	c := state(bench.Start(seat))
	for c.Loop() {
		body()
	}
	c.End()

	return seat
}

// noop is a body that does nothing.
func noop() {}

// verdictOf returns the verdict of a ceiling that a body exceeds: fail in a
// build that checks it, and pass in any other.
func verdictOf(checked bool) string {
	if checked {
		return "fail"
	}
	return "pass"
}

// allocating returns a body that makes one 4,096-byte heap allocation per
// call. The body appends each slice to a slice of its own, so the slice
// escapes to the heap and parallel tests do not share a variable.
func allocating() func() {
	var kept [][]byte
	return func() { kept = append(kept, make([]byte, 4096)) }
}

// endlessSeat is a fake benchmark whose Loop never ends and which writes no
// call record, so a case calls the methods of a contract as often as a
// measurement does.
type endlessSeat struct {
	matchertest.Seat
}

// Loop reports that another iteration runs.
func (*endlessSeat) Loop() bool { return true }

// ReportMetric discards the metric.
func (*endlessSeat) ReportMetric(float64, string) {}

// started keeps the contract that a case starts, so the compiler moves it
// to the heap.
var started *bench.Contract

// contractCases returns a call of each function and method of a contract,
// with its allocation ceiling, measured.
func contractCases() []alloctest.Case {
	b := &endlessSeat{}
	stated := bench.Start(b)
	running := bench.Start(b)
	running.Loop()
	ended := bench.Start(b)
	for range iterations {
		ended.Loop()
	}
	return []alloctest.Case{
		{Name: "Start", Call: func(assert.TB) { started = bench.Start(b) }, Allocs: 1},
		{Name: "MaxLatency", Call: func(assert.TB) { stated.MaxLatency(time.Second) }},
		{Name: "MaxMean", Call: func(assert.TB) { stated.MaxMean(time.Second) }},
		{Name: "MaxAllocs", Call: func(assert.TB) { stated.MaxAllocs(1) }},
		{Name: "MaxBytes", Call: func(assert.TB) { stated.MaxBytes(1) }},
		{Name: "Warmup", Call: func(assert.TB) { stated.Warmup(1) }},
		{Name: "Loop", Call: func(assert.TB) { running.Loop() }},
		{Name: "Excluding", Call: func(assert.TB) { running.Excluding(noop) }},
		{Name: "End", Call: func(assert.TB) { ended.End() }, Allocs: 1},
	}
}

// TestContract checks the loop of a contract and the verdict of each ceiling
// that it states.
func TestContract(t *testing.T) {
	t.Parallel()

	unconstrained := func(c *bench.Contract) *bench.Contract { return c }

	t.Run("Loop", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the body once per iteration", func(t *testing.T) {
			t.Parallel()

			calls := 0
			run(iterations, unconstrained, func() { calls++ })

			assert.Equal(t, calls, iterations, "the body runs once per iteration")
		})
	})

	t.Run("End", func(t *testing.T) {
		t.Parallel()

		t.Run("reports no verdict when no iteration ran", func(t *testing.T) {
			t.Parallel()

			seat := run(0, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond)
			}, noop)

			assert.False(t, seat.Failed(), "a run of no iterations exceeds no ceiling")
			assert.Length(t, seat.verdicts(t), 0, "a run of no iterations checks no ceiling")
		})

		t.Run("publishes a metric under every unit", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, unconstrained, noop)

			for _, unit := range []string{"p99-ns/op", "mean-ns/op", "allocs/op", "bytes/op"} {
				_, published := seat.metric(unit)
				assert.True(t, published, "the contract publishes "+unit)
			}
		})

		t.Run("checks no ceiling that was not stated", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, unconstrained, allocating())

			assert.False(t, seat.Failed(), "a ceiling that was not stated is not checked")
			assert.Length(t, seat.verdicts(t), 0, "a ceiling that was not stated states no verdict")
		})

		t.Run("passes each stated ceiling that the benchmark meets", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(50 * time.Millisecond).MaxMean(50 * time.Millisecond)
			}, func() { time.Sleep(time.Millisecond) })

			assert.False(t, seat.Failed(), "a body that sleeps for a millisecond meets ceilings of 50 milliseconds")
			assert.Equal(t, seat.verdicts(t), []string{"bench-max-latency pass", "bench-max-mean pass"},
				"each stated ceiling passes")
		})

		t.Run("reports a p99 latency above its ceiling", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond)
			}, func() { time.Sleep(time.Millisecond) })

			records := seat.Records()
			assert.Length(t, records, 1, "a body slower than its ceiling fails the benchmark")
			assert.Equal(t, records[0].Contract, "the p99 latency per iteration is within its ceiling",
				"the contract names the ceiling that was exceeded")
		})

		t.Run("reports a record of bench-max-latency in the test file that ends the contract", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond)
			}, func() { time.Sleep(time.Millisecond) })

			records := seat.Records()
			assert.Length(t, records, 1, "the exceeded ceiling reports one record")
			assert.Equal(t, records[0].Assertion, "bench-max-latency", "the record names the ceiling")
			assert.Equal(t, records[0].Detail["want"], any(time.Nanosecond), "the record states the ceiling as want")
			got, _ := records[0].Detail["got"].(time.Duration)
			assert.True(t, got >= time.Millisecond, "the record states the p99 latency as got")
			assert.Equal(t, filepath.Base(records[0].Where.File), "contract_test.go",
				"the record names the test file that calls End")
		})

		t.Run("reports allocations above their ceiling in a build that counts them", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, func(c *bench.Contract) *bench.Contract {
				return c.MaxAllocs(0)
			}, allocating())

			assert.Equal(t, seat.Failed(), matcher.AllocationsCounted(),
				"a body that allocates exceeds a ceiling of zero allocations in a build that checks it")
			assert.Equal(t, seat.verdicts(t), []string{"bench-max-allocs " + verdictOf(matcher.AllocationsCounted())},
				"the ceiling fails in a build that checks it and passes in any other")
			if matcher.AllocationsCounted() {
				records := seat.Records()
				assert.Equal(t, records[0].Assertion, "bench-max-allocs", "the record names the ceiling")
				assert.Equal(t, records[0].Detail["want"], any(uint64(0)), "the record states the ceiling as want")
				got, _ := records[0].Detail["got"].(uint64)
				assert.True(t, got >= 1, "the record states the allocations per iteration as got")
			}
		})

		t.Run("reports bytes above their ceiling in a build that counts them", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, func(c *bench.Contract) *bench.Contract {
				return c.MaxBytes(0)
			}, allocating())

			assert.Equal(t, seat.Failed(), matcher.AllocationsCounted(),
				"a body that allocates exceeds a ceiling of zero bytes in a build that checks it")
			assert.Equal(t, seat.verdicts(t), []string{"bench-max-bytes " + verdictOf(matcher.AllocationsCounted())},
				"the ceiling fails in a build that checks it and passes in any other")
			if matcher.AllocationsCounted() {
				records := seat.Records()
				assert.Equal(t, records[0].Assertion, "bench-max-bytes", "the record names the ceiling")
				assert.Equal(t, records[0].Detail["want"], any(uint64(0)), "the record states the ceiling as want")
				got, _ := records[0].Detail["got"].(uint64)
				assert.True(t, got >= 4096, "the record states the bytes per iteration as got")
			}
		})

		t.Run("reports each exceeded ceiling", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond).MaxMean(time.Nanosecond)
			}, func() { time.Sleep(time.Millisecond) })

			assert.Length(t, seat.Errs(), 2, "the p99 ceiling and the mean ceiling both fail")
			var ids []string
			for _, record := range seat.Records() {
				ids = append(ids, record.Assertion)
			}
			assert.Equal(t, ids, []string{"bench-max-latency", "bench-max-mean"},
				"each record names the ceiling it exceeded")
		})
	})

	t.Run("MaxLatency", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the receiver", func(t *testing.T) {
			t.Parallel()

			c := bench.Start(newBenchSeat(1))
			assert.Equal(t, c.MaxLatency(time.Second), c, "MaxLatency returns the receiver")
		})
	})

	t.Run("MaxAllocs", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the receiver", func(t *testing.T) {
			t.Parallel()

			c := bench.Start(newBenchSeat(1))
			assert.Equal(t, c.MaxAllocs(1).MaxBytes(1).MaxMean(time.Second), c,
				"every ceiling returns the receiver")
		})
	})

	t.Run("Warmup", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the receiver", func(t *testing.T) {
			t.Parallel()

			c := bench.Start(newBenchSeat(1))
			assert.Equal(t, c.Warmup(3), c, "Warmup returns the receiver")
		})

		t.Run("runs the body once per warm-up iteration before the measured ones", func(t *testing.T) {
			t.Parallel()

			calls := 0
			seat := run(iterations, func(c *bench.Contract) *bench.Contract { return c.Warmup(5) }, func() { calls++ })

			assert.Equal(t, calls, iterations+5, "the body runs 5 warm-up iterations and the measured ones")
			assert.Equal(t, seat.remaining, 0, "the benchmark's own loop runs the measured iterations alone")
		})

		t.Run("leaves a warm-up iteration out of the latency ceilings", func(t *testing.T) {
			t.Parallel()

			first := true
			seat := run(4, func(c *bench.Contract) *bench.Contract {
				return c.Warmup(1).MaxLatency(10 * time.Millisecond)
			}, func() {
				if first {
					first = false
					time.Sleep(50 * time.Millisecond)
				}
			})

			assert.False(t, seat.Failed(), "a slow first iteration in the warm-up exceeds no ceiling")
		})

		t.Run("publishes nothing for a run that ends during the warm-up", func(t *testing.T) {
			t.Parallel()

			seat := newBenchSeat(iterations)
			c := bench.Start(seat).Warmup(3).MaxLatency(time.Nanosecond)
			for c.Loop() {
				break
			}
			c.End()

			_, published := seat.metric("p99-ns/op")
			assert.False(t, published, "a run that measured nothing publishes nothing")
			assert.Length(t, seat.verdicts(t), 0, "a run that measured nothing checks no ceiling")
		})

		t.Run("panics for a negative number of iterations", func(t *testing.T) {
			t.Parallel()

			c := bench.Start(newBenchSeat(1))
			raised := assert.Panics(t, func() { c.Warmup(-1) }, "a negative count is a usage error")
			assert.Equal(t, raised, any("bench: Warmup(-1) states a negative number of iterations"),
				"the message names the call")
		})

		t.Run("panics after Loop has run", func(t *testing.T) {
			t.Parallel()

			c := bench.Start(newBenchSeat(1))
			c.Loop()
			raised := assert.Panics(t, func() { c.Warmup(1) }, "a warm-up after the first iteration is a usage error")
			assert.Equal(t, raised, any("bench: Warmup after the contract has run its body"),
				"the message names the call")
		})
	})
}

// sink stores what a fixture built, so that escape analysis cannot
// remove the allocation.
var sink [][]int

// heavyFixture makes 17 heap allocations, so a run that counts them
// exceeds a ceiling of two allocations per iteration.
func heavyFixture() {
	held := make([][]int, 0, 16)
	for range 16 {
		held = append(held, make([]int, 32))
	}
	sink = held
}

// longRun is the number of iterations in a case that states a ceiling
// near zero. The runtime makes allocations of its own during a run, up to
// 5.5 KB measured in a fresh process, and over this many iterations they
// round to zero per iteration. It is a multiple of 16, so [every] makes
// exactly longRun/n allocations in a run of this length for an n of 2 or 16.
const longRun = 160_000

// every returns a body that makes one 24-byte allocation in every n calls.
// Over longRun iterations, every(16) measures 0.0625 allocations and 1.5
// bytes per iteration, and every(2) half an allocation and 12 bytes.
func every(n int) func() {
	calls := 0
	return func() {
		calls++
		if calls%n == 0 {
			sink = make([][]int, 1)
		}
	}
}

// runExcluding drives a contract over a body that receives the contract,
// so that the body can call Excluding.
func runExcluding(
	iterations int,
	state func(*bench.Contract) *bench.Contract,
	body func(*bench.Contract),
) *benchSeat {
	seat := newBenchSeat(iterations)

	c := state(bench.Start(seat))
	for c.Loop() {
		body(c)
	}
	c.End()

	return seat
}

// TestContractCounting checks how a contract counts the allocations of a
// body. It does not run in parallel: the allocation counter is
// process-wide, so an allocation by a parallel test would count against
// the ceilings that these cases state.
//
// Each Excluding case that takes work out has a twin that runs the same
// work in the body without Excluding. The twin fails, which proves that
// the work counts unless Excluding takes it out.
func TestContractCounting(t *testing.T) {
	nothing := func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(0).MaxBytes(0) }
	tightAllocs := func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(2) }

	t.Run("Loop", func(t *testing.T) {
		t.Run("leaves the contract's bookkeeping out of the allocation ceilings", func(t *testing.T) {
			seat := run(longRun, nothing, noop)

			assert.Equal(t, seat.First(), "", "a body that allocates nothing meets ceilings of zero")
		})

		t.Run("leaves the warm-up out of the allocation ceilings", func(t *testing.T) {
			warmed := false
			seat := run(1, func(c *bench.Contract) *bench.Contract { return c.Warmup(1).MaxAllocs(0) }, func() {
				if !warmed {
					warmed = true
					sink = make([][]int, 128)
				}
			})

			assert.Equal(t, seat.First(), "", "the allocation of the warm-up iteration counts against no ceiling")
		})

		t.Run("counts the first iteration of a run without a warm-up", func(t *testing.T) {
			warmed := false
			seat := run(1, func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(0) }, func() {
				if !warmed {
					warmed = true
					sink = make([][]int, 128)
				}
			})

			assert.Equal(t, seat.Failed(), matcher.AllocationsCounted(),
				"the allocation of the one measured iteration exceeds a ceiling of zero in a build that checks it")
		})

		t.Run("leaves the code after the loop out of the allocation ceilings", func(t *testing.T) {
			seat := newBenchSeat(iterations)

			c := bench.Start(seat).MaxBytes(1024)
			for c.Loop() {
				noop()
			}
			sink = make([][]int, 1<<16)
			c.End()

			assert.Equal(t, seat.First(), "",
				"1.5 MiB allocated after the loop does not count against a ceiling of 1,024 bytes per iteration")
		})
	})

	t.Run("End", func(t *testing.T) {
		t.Run("rounds the allocations per iteration to the nearest whole number", func(t *testing.T) {
			seat := run(longRun, func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(0) }, every(16))

			assert.Equal(t, seat.First(), "", "0.0625 allocations per iteration round to zero")
		})

		t.Run("rounds half an allocation per iteration up", func(t *testing.T) {
			seat := run(longRun, func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(0) }, every(2))

			want := []string{"bench-max-allocs " + verdictOf(matcher.AllocationsCounted())}
			assert.Equal(t, seat.verdicts(t), want, "half an allocation per iteration rounds up to one")
		})

		t.Run("rounds the bytes per iteration down before the comparison", func(t *testing.T) {
			seat := run(longRun, func(c *bench.Contract) *bench.Contract { return c.MaxBytes(1) }, every(16))

			assert.Equal(t, seat.First(), "", "1.5 bytes per iteration round down to one byte")
		})

		t.Run("reads the end counters itself when the body leaves the loop early", func(t *testing.T) {
			seat := newBenchSeat(longRun)

			c := nothing(bench.Start(seat))
			for calls := 1; c.Loop(); calls++ {
				if calls == longRun/2 {
					break
				}
			}
			c.End()

			assert.Equal(t, seat.First(), "", "a loop left early is counted up to the iteration it left")
		})

		t.Run("measures and checks the one iteration that the body leaves with break", func(t *testing.T) {
			seat := newBenchSeat(iterations)

			c := nothing(bench.Start(seat))
			for c.Loop() {
				sink = make([][]int, 128)
				break
			}
			c.End()

			assert.Equal(t, seat.verdicts(t), []string{
				"bench-max-allocs " + verdictOf(matcher.AllocationsCounted()),
				"bench-max-bytes " + verdictOf(matcher.AllocationsCounted()),
			}, "the allocation of the iteration exceeds both ceilings of zero")
			bytes, published := seat.metric("bytes/op")
			assert.True(t, published, "the contract publishes the bytes of the iteration")
			assert.InRange(t, bytes, 3072.0, 4096.0, "the 3 KiB of the iteration")
		})

		t.Run("counts the time of the iteration that the body leaves", func(t *testing.T) {
			seat := newBenchSeat(iterations)

			c := bench.Start(seat)
			for calls := 1; c.Loop(); calls++ {
				if calls == 2 {
					time.Sleep(10 * time.Millisecond)
					break
				}
			}
			c.End()

			mean, _ := seat.metric("mean-ns/op")
			assert.InRange(t, mean, float64(5*time.Millisecond), float64(time.Second),
				"the mean of a quick iteration and one of 10 ms")
		})

		t.Run("publishes each count per iteration before rounding", func(t *testing.T) {
			seat := run(longRun, func(c *bench.Contract) *bench.Contract { return c }, every(16))

			allocs, _ := seat.metric("allocs/op")
			assert.InRange(t, allocs, 0.0625, 0.5, "the contract publishes 0.0625 allocations per iteration")
			bytes, _ := seat.metric("bytes/op")
			assert.InRange(t, bytes, 1.5, 2, "the contract publishes 1.5 bytes per iteration")
		})
	})

	t.Run("Excluding", func(t *testing.T) {
		t.Run("takes the setup's allocations out of the count", func(t *testing.T) {
			seat := runExcluding(iterations, tightAllocs, func(c *bench.Contract) {
				c.Excluding(heavyFixture)
			})

			assert.False(t, seat.Failed(), "an excluded fixture does not count against the ceiling")
		})

		t.Run("leaves a fixture built outside it in the count", func(t *testing.T) {
			seat := runExcluding(iterations, tightAllocs, func(*bench.Contract) {
				heavyFixture()
			})

			assert.Equal(t, seat.Failed(), matcher.AllocationsCounted(),
				"a fixture built outside Excluding counts against the ceiling in a build that checks it")
		})

		t.Run("takes no allocations out before the loop", func(t *testing.T) {
			seat := newBenchSeat(iterations)

			c := tightAllocs(bench.Start(seat))
			c.Excluding(heavyFixture)
			for c.Loop() {
				noop()
			}
			c.End()

			assert.Equal(t, seat.First(), "", "an exclusion before the loop leaves the count of the loop unchanged")
		})

		t.Run("takes no allocations out after the loop", func(t *testing.T) {
			seat := newBenchSeat(iterations)

			c := tightAllocs(bench.Start(seat))
			for c.Loop() {
				noop()
			}
			c.Excluding(heavyFixture)
			c.End()

			assert.Equal(t, seat.First(), "", "an exclusion after the loop leaves the count of the loop unchanged")
		})

		t.Run("takes the setup's time out of the iteration", func(t *testing.T) {
			tight := func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(5 * time.Millisecond)
			}
			seat := runExcluding(4, tight, func(c *bench.Contract) {
				c.Excluding(func() { time.Sleep(20 * time.Millisecond) })
			})

			assert.False(t, seat.Failed(), "an excluded sleep is not timed")
		})

		t.Run("takes the time of every call out of the iteration", func(t *testing.T) {
			tight := func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(5 * time.Millisecond)
			}
			seat := runExcluding(4, tight, func(c *bench.Contract) {
				c.Excluding(func() { time.Sleep(10 * time.Millisecond) })
				c.Excluding(func() { time.Sleep(10 * time.Millisecond) })
			})

			assert.False(t, seat.Failed(), "neither of two excluded sleeps is timed")
		})

		t.Run("leaves a sleep outside it in the iteration", func(t *testing.T) {
			tight := func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(5 * time.Millisecond)
			}
			seat := runExcluding(4, tight, func(*bench.Contract) {
				time.Sleep(20 * time.Millisecond)
			})

			assert.True(t, seat.Failed(), "a sleep outside Excluding is timed")
		})

		t.Run("takes no time out of the first iteration before the loop", func(t *testing.T) {
			seat := newBenchSeat(4)

			c := bench.Start(seat).MaxMean(900 * time.Microsecond)
			c.Excluding(func() { time.Sleep(2 * time.Millisecond) })
			for c.Loop() {
				time.Sleep(time.Millisecond)
			}
			c.End()

			assert.True(t, seat.Failed(), "iterations of a millisecond exceed a mean of 0.9 milliseconds")
		})
	})
}

// TestContractAfterRunParallel checks the calls that a contract refuses once
// RunParallel has run its body, in benchmarks of testing that run in this
// process. It does not run in parallel, because it sets -benchtime, a flag
// of the process.
func TestContractAfterRunParallel(t *testing.T) {
	tests := []struct {
		name string
		give func(c *bench.Contract)
		want string
	}{
		{name: "Loop", give: func(c *bench.Contract) { c.Loop() }, want: "bench: Loop after RunParallel"},
		{
			name: "Warmup",
			give: func(c *bench.Contract) { c.Warmup(1) },
			want: "bench: Warmup after the contract has run its body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("panics after RunParallel has run", func(t *testing.T) {
				var recovered any
				benchmark(t, "1x", func(b bench.B, _ int) {
					c := bench.Start(b)
					c.RunParallel(spin)
					func() {
						defer func() { recovered = recover() }()
						tt.give(c)
					}()
				})

				assert.Equal(t, recovered, any(tt.want), "the message names the call")
			})
		})
	}
}

// TestContractAllocs checks the allocation ceiling of each function and
// method of a contract.
func TestContractAllocs(t *testing.T) {
	alloctest.Check(t, contractCases())
}

// BenchmarkContract measures each function and method of a contract.
func BenchmarkContract(b *testing.B) {
	for _, c := range contractCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
