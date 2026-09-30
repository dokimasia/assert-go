// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/matcher"
)

// This package consumes the library, so its tests use the library's own
// assertions.

// iterations is the number of iterations the stand-in runs. At 100
// iterations the p99 is a different sample from the slowest one.
const iterations = 100

// run drives a contract over body and returns the benchmark stand-in,
// which records what the contract reported.
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

// allocating returns a body that makes one 4,096-byte heap allocation per
// call. The body appends each slice to a slice of its own, so the slice
// escapes to the heap and parallel tests do not share a variable.
func allocating() func() {
	var kept [][]byte
	return func() { kept = append(kept, make([]byte, 4096)) }
}

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

		t.Run("reports nothing when no iteration ran", func(t *testing.T) {
			t.Parallel()

			seat := run(0, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond)
			}, noop)

			assert.False(t, seat.Failed(), "a run of no iterations exceeds no ceiling")
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
		})

		t.Run("reports nothing when every ceiling is met", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(50 * time.Millisecond).MaxMean(50 * time.Millisecond)
			}, func() { time.Sleep(time.Millisecond) })

			assert.False(t, seat.Failed(), "a body that sleeps for a millisecond meets ceilings of 50 milliseconds")
		})

		t.Run("reports a p99 latency above its ceiling", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond)
			}, func() { time.Sleep(time.Millisecond) })

			assert.True(t, seat.Failed(), "a body slower than its ceiling fails the benchmark")
			assert.Contains(t, seat.First(), "p99",
				"the failure names the ceiling that was exceeded")
		})

		t.Run("reports allocations above their ceiling in a build that counts them", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, func(c *bench.Contract) *bench.Contract {
				return c.MaxAllocs(0)
			}, allocating())

			assert.Equal(t, seat.Failed(), matcher.AllocationsCounted(),
				"a body that allocates exceeds a ceiling of zero allocations in a build that checks it")
		})

		t.Run("reports bytes above their ceiling in a build that counts them", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, func(c *bench.Contract) *bench.Contract {
				return c.MaxBytes(0)
			}, allocating())

			assert.Equal(t, seat.Failed(), matcher.AllocationsCounted(),
				"a body that allocates exceeds a ceiling of zero bytes in a build that checks it")
		})

		t.Run("reports each exceeded ceiling", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond).MaxMean(time.Nanosecond)
			}, func() { time.Sleep(time.Millisecond) })

			assert.Length(t, seat.Errs(), 2, "the p99 ceiling and the mean ceiling both fail")
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
// round down to zero per iteration. It is a multiple of 16, so [sparse]
// makes exactly longRun/16 allocations in a run of this length.
const longRun = 160_000

// sparse returns a body that makes one 24-byte allocation in every 16
// calls. Over longRun iterations it measures 0.0625 allocations and 1.5
// bytes per iteration.
func sparse() func() {
	calls := 0
	return func() {
		calls++
		if calls%16 == 0 {
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

// TestContractAllocs does not run in parallel. The allocation counter is
// process-wide, so an allocation by a parallel test would count against
// the ceilings that these cases state.
//
// Each Excluding case that takes work out has a twin that runs the same
// work in the body without Excluding. The twin fails, which proves that
// the work counts unless Excluding takes it out.
func TestContractAllocs(t *testing.T) {
	nothing := func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(0).MaxBytes(0) }
	tightAllocs := func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(2) }

	t.Run("Loop", func(t *testing.T) {
		t.Run("leaves the contract's bookkeeping out of the allocation ceilings", func(t *testing.T) {
			seat := run(longRun, nothing, noop)

			assert.Equal(t, seat.First(), "", "a body that allocates nothing meets ceilings of zero")
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
		t.Run("rounds each count per iteration down before the comparison", func(t *testing.T) {
			seat := run(longRun, func(c *bench.Contract) *bench.Contract {
				return c.MaxAllocs(0).MaxBytes(1)
			}, sparse())

			assert.Equal(t, seat.First(), "",
				"0.0625 allocations and 1.5 bytes per iteration meet ceilings of zero allocations and one byte")
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

		t.Run("publishes each count per iteration before rounding", func(t *testing.T) {
			seat := run(longRun, func(c *bench.Contract) *bench.Contract { return c }, sparse())

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
