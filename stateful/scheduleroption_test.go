// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// TestSchedulerOption checks the options of a scheduler.
func TestSchedulerOption(t *testing.T) {
	t.Parallel()

	t.Run("MaxReleases", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the most releases of one Run", func(t *testing.T) {
			t.Parallel()
			e := limited(1, func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(spinning(s, &runLog{}, "a"))
				s.Run()
			})
			assert.Equal(t, e.Case.Failures()[0].Contract,
				"the task scheduler released 1 tasks in one run, and a task is still ready",
				"the second release passes the limit of one")
		})

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.MaxReleases(0) }, "no release")
			assert.Equal(t, got, any("stateful: MaxReleases(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("SchedulerOption", func(t *testing.T) {
		t.Parallel()

		t.Run("changes nothing as the zero option", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) {
				s := stateful.NewScheduler(c, stateful.Uniform(), stateful.SchedulerOption{})
				s.Spawn(yielding(s, log, "a"))
				s.Run()
			}, 0, 0)
			assert.Equal(t, []any{e.Status, log.all()}, []any{engine.CasePassed, suffixed("a1", "a2")},
				"the default limit")
		})
	})
}

// TestSchedulerOptionAllocs checks that an option of a scheduler allocates
// nothing.
func TestSchedulerOptionAllocs(t *testing.T) {
	var kept stateful.SchedulerOption
	assert.MaxAllocs(t, func() { kept = stateful.MaxReleases(10) }, 0, "MaxReleases returns a value")
	assert.NotEqual(t, kept, stateful.SchedulerOption{}, "the kept option states a limit")
}

// BenchmarkSchedulerOption measures each option of a scheduler.
func BenchmarkSchedulerOption(b *testing.B) {
	b.Run("MaxReleases", func(b *testing.B) {
		var got stateful.SchedulerOption
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = stateful.MaxReleases(10)
		}
		assert.NotEqual(b, got, stateful.SchedulerOption{}, "the option states a limit")
	})
}
