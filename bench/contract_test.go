// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
)

// This package is a consumer of the library, so its tests are written
// with it.

// iterations is how many the stand-in runs. Enough that a p99 differs
// from the slowest sample, few enough to stay quick.
const iterations = 100

// run drives a contract over a body and answers the seat it reported
// through.
func run(iterations int, state func(*bench.Contract) *bench.Contract, body func()) *benchSeat {
	seat := newBenchSeat(iterations)

	c := state(bench.Start(seat))
	for c.Loop() {
		body()
	}
	c.End()

	return seat
}

// noop is a body that costs as little as a body can.
func noop() {}

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

		t.Run("a benchmark of no iterations reports nothing", func(t *testing.T) {
			t.Parallel()

			seat := run(0, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond)
			}, noop)

			assert.False(t, seat.Failed(),
				"nothing measured means no ceiling to exceed")
		})
	})

	t.Run("End", func(t *testing.T) {
		t.Parallel()

		t.Run("publishes what it measured", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, unconstrained, noop)

			for _, unit := range []string{"p99-ns/op", "mean-ns/op", "allocs/op", "bytes/op"} {
				_, published := seat.metric(unit)
				assert.True(t, published, "the contract publishes "+unit)
			}
		})

		t.Run("an unstated ceiling is not checked", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, unconstrained, func() { _ = make([]byte, 1024) })

			assert.False(t, seat.Failed(), "a ceiling nobody stated is not enforced")
		})

		t.Run("a ceiling that holds reports nothing", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Second).MaxMean(time.Second)
			}, noop)

			assert.False(t, seat.Failed(),
				"a second per iteration is not exceeded by an empty body")
		})

		t.Run("an exceeded latency ceiling reports", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond)
			}, func() { time.Sleep(time.Millisecond) })

			assert.True(t, seat.Failed(), "a body slower than its ceiling reports")
			assert.Contains(t, seat.First(), "p99",
				"the failure names the ceiling that was exceeded")
		})

		t.Run("an exceeded allocation ceiling reports", func(t *testing.T) {
			t.Parallel()

			seat := run(iterations, func(c *bench.Contract) *bench.Contract {
				return c.MaxAllocs(0)
			}, func() { _ = make([]byte, 4096) })

			assert.True(t, seat.Failed(),
				"a body that allocates exceeds a ceiling of none")
		})

		t.Run("every exceeded ceiling is reported, not only the first", func(t *testing.T) {
			t.Parallel()

			seat := run(10, func(c *bench.Contract) *bench.Contract {
				return c.MaxLatency(time.Nanosecond).MaxMean(time.Nanosecond)
			}, func() { time.Sleep(time.Millisecond) })

			assert.Length(t, seat.Errs(), 2,
				"both exceeded ceilings are reported, not only the first")
		})
	})

	t.Run("MaxLatency", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the receiver so ceilings chain", func(t *testing.T) {
			t.Parallel()

			c := bench.Start(newBenchSeat(1))
			assert.Equal(t, c.MaxLatency(time.Second), c, "MaxLatency returns the receiver")
		})
	})

	t.Run("MaxAllocs", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the receiver so ceilings chain", func(t *testing.T) {
			t.Parallel()

			c := bench.Start(newBenchSeat(1))
			assert.Equal(t, c.MaxAllocs(1).MaxBytes(1).MaxMean(time.Second), c,
				"every ceiling returns the receiver")
		})
	})
}

// sink holds what a fixture built, so escape analysis cannot decide the
// allocation never happened.
var sink [][]int

// heavyFixture allocates well past a ceiling of two, so a run that
// counts it cannot meet one.
func heavyFixture() {
	held := make([][]int, 0, 16)
	for range 16 {
		held = append(held, make([]int, 32))
	}
	sink = held
}

// runExcluding drives a contract whose body can reach it, which is what
// Excluding needs and what run does not offer.
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

// TestExcluding does not run in parallel. The allocation counter is
// process-wide, so a test reading it beside another that allocates
// reads the other's work as its own.
func TestExcluding(t *testing.T) {
	tightAllocs := func(c *bench.Contract) *bench.Contract { return c.MaxAllocs(2) }

	t.Run("takes the setup's allocations out of the ceiling", func(t *testing.T) {
		seat := runExcluding(iterations, tightAllocs, func(c *bench.Contract) {
			c.Excluding(heavyFixture)
		})

		assert.False(t, seat.Failed(),
			"an excluded fixture does not count against the ceiling")
	})

	t.Run("the same fixture unexcluded crosses the ceiling", func(t *testing.T) {
		// Without this the case above passes against an Excluding that
		// does nothing at all.
		seat := runExcluding(iterations, tightAllocs, func(*bench.Contract) {
			heavyFixture()
		})

		assert.True(t, seat.Failed(),
			"a fixture nobody excluded has to count against the ceiling")
	})

	t.Run("takes the setup's time out of the ceiling", func(t *testing.T) {
		tight := func(c *bench.Contract) *bench.Contract {
			return c.MaxLatency(5 * time.Millisecond)
		}
		seat := runExcluding(4, tight, func(c *bench.Contract) {
			c.Excluding(func() { time.Sleep(20 * time.Millisecond) })
		})

		assert.False(t, seat.Failed(), "an excluded sleep is not timed")
	})

	t.Run("the same sleep unexcluded crosses the ceiling", func(t *testing.T) {
		tight := func(c *bench.Contract) *bench.Contract {
			return c.MaxLatency(5 * time.Millisecond)
		}
		seat := runExcluding(4, tight, func(*bench.Contract) {
			time.Sleep(20 * time.Millisecond)
		})

		assert.True(t, seat.Failed(), "a sleep nobody excluded has to be timed")
	})
}
