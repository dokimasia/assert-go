// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// TestClock checks the clock of a scheduler: its start, what moves it, and
// how a task sleeps on it.
func TestClock(t *testing.T) {
	t.Parallel()

	t.Run("Clock", func(t *testing.T) {
		t.Parallel()

		t.Run("reads midnight UTC 2000-01-01 until an advance moves it", func(t *testing.T) {
			t.Parallel()
			var start, later time.Time
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				start = s.Clock().Now()
				s.Advance(time.Minute)
				later = s.Clock().Now()
			})
			assert.Equal(t, []time.Time{start, later}, []time.Time{epoch, epoch.Add(time.Minute)},
				"the start, and a minute later")
		})

		t.Run("parks a sleeping task, and Run moves the clock by nothing", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			var now time.Time
			var first []string
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(ticking(s, log, "t", 4*time.Millisecond, 1))
				s.Run()
				first, now = log.all(), s.Clock().Now()
				s.Advance(4 * time.Millisecond)
			})
			assert.Equal(t, []any{first, now}, []any{[]string(nil), epoch}, "Run leaves the task asleep")
			assert.Equal(t, log.all(), suffixed("t@4ms"), "the advance wakes it")
		})

		t.Run("returns at once from a sleep of no time, without a release", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(ticking(s, log, "t", 0, 2))
				s.Run()
			})
			assert.Equal(t, log.all(), suffixed("t@0s", "t@0s"), "both ticks within one turn")
			assert.Length(t, choicesOf(e), 1, "one release")
		})

		t.Run("returns at once outside a task from a sleep of no time", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Clock().Sleep(-time.Second)
				log.add("after", 0)
			})
			assert.Equal(t, log.all(), suffixed("after"), "the body runs on")
		})

		t.Run("panics outside a task for a sleep of a positive duration", func(t *testing.T) {
			t.Parallel()
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Clock().Sleep(time.Millisecond)
			})
			assert.Equal(t, e.Panic, any("stateful: Sleep outside a task waits for a clock that only the body moves"),
				"the panic fails the case")
		})
	})
}

// The bodies of the measured cases of the clock.
var (
	// reading reads the clock of a scheduler.
	reading = bodyOf(func(c *prop.Case) { stateful.NewScheduler(c, stateful.Uniform()).Clock().Now() })
	// sleepingOutside sleeps on the clock of a scheduler for no time, outside
	// a task.
	sleepingOutside = bodyOf(func(c *prop.Case) { stateful.NewScheduler(c, stateful.Uniform()).Clock().Sleep(0) })
)

// clockAllocs are the cases of the allocation ceilings of the clock, each a
// whole replayed case.
var clockAllocs = []alloctest.Case{
	{Name: "Clock", Call: func(assert.TB) { engine.Replay(reading, nil, nil) }, Allocs: schedulerCaseAllocs},
	{Name: "Sleep", Call: func(assert.TB) { engine.Replay(sleepingOutside, nil, nil) }, Allocs: schedulerCaseAllocs},
}

// TestClockAllocs checks the allocation ceilings of the clock.
func TestClockAllocs(t *testing.T) {
	alloctest.Check(t, clockAllocs)
}

// BenchmarkClock measures the clock, each on a whole replayed case.
func BenchmarkClock(b *testing.B) {
	for _, c := range clockAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
