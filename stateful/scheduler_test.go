// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"runtime"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// The allocation ceilings of a whole replayed case that uses a scheduler. A
// case that starts a task allocates up to two more when the runtime makes
// the goroutine of the case or of the task rather than reusing one, and its
// ceiling includes the two.
const (
	// schedulerCaseAllocs is the ceiling of the allocations of a case that
	// makes a scheduler: the case's own three, the scheduler, its channel,
	// its cleanup and the growth of the case's cleanups.
	schedulerCaseAllocs = 9
	// spawnCaseAllocs is the ceiling of the allocations of a case that
	// spawns one task and never runs it: those of a scheduler case, the
	// task, its channel, its goroutine's start and the growth of the ready
	// tasks.
	spawnCaseAllocs = 17
	// runCaseAllocs is the ceiling of the allocations of a case that spawns
	// one task and runs it: those of a spawning case, and the release's
	// choice.
	runCaseAllocs = 19
)

// TestScheduler checks the releases of a scheduler: the order of the ready
// tasks, the choices of each strategy, and how a task's end ends Run.
func TestScheduler(t *testing.T) {
	t.Parallel()

	t.Run("Spawn", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []uint64
			want []string
		}{
			{
				name: "makes each task ready after every ready task",
				give: []uint64{0, 0, 0},
				want: []string{"a", "b", "c"},
			},
			{
				name: "lets the case choose among the ready tasks",
				give: []uint64{2, 1, 0},
				want: []string{"c", "b", "a"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				log := &runLog{}
				scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
					for _, name := range []string{"a", "b", "c"} {
						s.Spawn(func() { log.add(name, 0) })
					}
					s.Run()
				}, tt.give...)
				assert.Equal(t, log.all(), suffixed(tt.want...), "the order of the releases")
			})
		}

		t.Run("gives each task a priority by a choice under PCT, which orders the releases", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := scheduled(stateful.PCT(1), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(func() { log.add("a", 0) })
				s.Spawn(func() { log.add("b", 0) })
				s.Run()
			}, 5, 9)
			assert.Equal(t, log.all(), suffixed("b", "a"), "b of the higher priority first")
			assert.Equal(t, choicesOf(e), []uint64{5, 9}, "the priorities, and no choice of a release")
		})
	})

	t.Run("Yield", func(t *testing.T) {
		t.Parallel()

		t.Run("makes the running task ready again after every ready task", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(yielding(s, log, "a"))
				s.Spawn(func() { log.add("b", 0) })
				s.Run()
			}, 0, 0, 0)
			assert.Equal(t, log.all(), suffixed("a1", "b", "a2"), "b between the two turns of a")
		})

		t.Run("returns at once outside a task", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Yield()
				log.add("after", 0)
			})
			assert.Equal(t, log.all(), suffixed("after"), "the body runs on")
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("releases a task that a running task spawns", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(func() {
					log.add("a", 0)
					s.Spawn(func() { log.add("b", 0) })
				})
				s.Run()
			})
			assert.Equal(t, log.all(), suffixed("a", "b"), "the spawned task after its parent")
		})

		t.Run("records a release of one ready task as a choice that consumes nothing", func(t *testing.T) {
			t.Parallel()
			var got, want uint64
			engine.Generate(bodyOf(func(c *prop.Case) {
				s := stateful.NewScheduler(c, stateful.Uniform())
				s.Spawn(func() {})
				s.Run()
				got = c.Rand().Uint64()
			}), seed, 0, nil)
			engine.Generate(bodyOf(func(c *prop.Case) { want = c.Rand().Uint64() }), seed, 0, nil)
			assert.Equal(t, got, want, "the first value of the case's stream")
		})

		tests := []struct {
			name string
			give []uint64
			want []string
		}{
			{
				name: "releases the task of the highest priority again after it yields under PCT",
				give: []uint64{9, 5, 0},
				want: []string{"a1", "a2", "b"},
			},
			{
				name: "lowers the task that a change point releases below every other task under PCT",
				give: []uint64{9, 5, 1, 0},
				want: []string{"a1", "b", "a2"},
			},
			{
				name: "releases the earliest ready task among equal priorities under PCT",
				give: []uint64{0, 0, 0},
				want: []string{"a1", "b", "a2"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				log := &runLog{}
				scheduled(stateful.PCT(2), func(_ *prop.Case, s *stateful.Scheduler) {
					s.Spawn(yielding(s, log, "a"))
					s.Spawn(func() { log.add("b", 0) })
					s.Run()
				}, tt.give...)
				assert.Equal(t, log.all(), suffixed(tt.want...), "the order of the turns")
			})
		}

		t.Run("panics when a task calls Run of its own scheduler", func(t *testing.T) {
			t.Parallel()
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(s.Run)
				s.Run()
			})
			assert.Equal(t, e.Panic, any("stateful: a task called Run of its own scheduler"),
				"the panic fails the case")
		})

		t.Run("ends every other task, then its own goroutine, when a task ends its goroutine", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := scheduled(stateful.Uniform(), func(c *prop.Case, s *stateful.Scheduler) {
				s.Spawn(yielding(s, log, "b"))
				s.Spawn(func() { c.Fatalf("stop") })
				s.Run()
				log.add("after", 0)
			}, 0, 0)
			assert.Equal(t, log.all(), suffixed("b1", "b ended"), "b ends in its yield, and Run returns no more")
			assert.Equal(t, e.Case.Failures()[0].Contract, "stop", "the task's record")
		})

		t.Run("ends every other task, then raises a task's panic", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(yielding(s, log, "b"))
				s.Spawn(func() { panic("broken counter") })
				s.Run()
			}, 0, 0)
			assert.Equal(t, log.all(), suffixed("b1", "b ended"), "b ends in its yield")
			assert.Equal(t, e.Panic, any("broken counter"), "the task's panic fails the case")
		})

		t.Run("raises the panic of a task that panics as Run ends it", func(t *testing.T) {
			t.Parallel()
			e := scheduled(stateful.Uniform(), func(c *prop.Case, s *stateful.Scheduler) {
				s.Spawn(func() {
					defer func() { panic("teardown") }()
					s.Yield()
				})
				s.Spawn(func() { c.Assume(false) })
				s.Run()
			}, 0, 0)
			assert.Equal(t, []any{e.Status, e.Panic}, []any{engine.CaseFailed, "teardown"},
				"the panic of the ended task fails the case that the other task rejected")
		})

		t.Run("keeps the place where each task raised its panic", func(t *testing.T) {
			t.Parallel()
			raisedAt := func(task func()) assert.Where {
				return scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
					s.Spawn(task)
					s.Run()
				}).Identity.Where
			}
			_, file, line, _ := runtime.Caller(0)
			first := raisedAt(func() { panic("first") })
			second := raisedAt(func() { panic("second") })
			assert.Equal(
				t,
				[]assert.Where{first, second},
				[]assert.Where{
					{File: file, Line: line + 1},
					{File: file, Line: line + 2},
				},
				"the line of each task's panic",
			)
		})
	})

	t.Run("NewScheduler", func(t *testing.T) {
		t.Parallel()

		t.Run("raises the panic of a task that panics as the case ends it", func(t *testing.T) {
			t.Parallel()
			e := scheduled(stateful.Uniform(), func(c *prop.Case, s *stateful.Scheduler) {
				source := c.Rand()
				for range engine.MaxChoices - 1 {
					source.Uint64()
				}
				s.Spawn(func() {
					defer func() { panic("teardown") }()
					s.Yield()
				})
				s.Spawn(func() {})
				s.Run()
			})
			assert.Equal(t, []any{e.Status, e.Panic}, []any{engine.CaseFailed, "teardown"},
				"the release past the cap ends the case, and the cleanup raises the ended task's panic")
		})
	})
}

// TestSchedulerGoroutinesProcess checks that a case's tasks end with the case. It
// reads every goroutine of the process, so it does not run in parallel.
func TestSchedulerGoroutinesProcess(t *testing.T) {
	t.Run("NewScheduler", func(t *testing.T) {
		t.Run("ends the tasks that wait for a release when the case ends", func(t *testing.T) {
			check := assert.NoGoroutineLeaks(t, "every task has ended")
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(func() {})
				s.Spawn(func() {})
			})
			check()
		})
	})
}

// scheduled runs body with a scheduler of strategy in a case that replays
// the integer choices, and returns how the case ended.
func scheduled(strategy stateful.Strategy, body func(c *prop.Case, s *stateful.Scheduler),
	choices ...uint64,
) engine.Execution {
	return replayed(func(c *prop.Case) { body(c, stateful.NewScheduler(c, strategy)) }, choices...)
}

// yielding returns a task named name that logs name1, yields, and logs
// name2, and logs "name ended" when the scheduler ends it in its yield.
func yielding(s *stateful.Scheduler, log *runLog, name string) func() {
	return func() {
		ended := true
		defer func() {
			if ended {
				log.add(name+" ended", 0)
			}
		}()
		log.add(name+"1", 0)
		s.Yield()
		log.add(name+"2", 0)
		ended = false
	}
}

// suffixed returns the log lines of names, each a run on client 0.
func suffixed(names ...string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = name + "/0"
	}
	return out
}

// The bodies of the measured cases.
var (
	// scheduling makes a scheduler.
	scheduling = bodyOf(func(c *prop.Case) { stateful.NewScheduler(c, stateful.Uniform()) })
	// spawning spawns one task and never runs it.
	spawning = bodyOf(func(c *prop.Case) { stateful.NewScheduler(c, stateful.Uniform()).Spawn(nothing) })
	// running spawns one task and runs it.
	running = bodyOf(func(c *prop.Case) {
		s := stateful.NewScheduler(c, stateful.Uniform())
		s.Spawn(nothing)
		s.Run()
	})
	// yieldingOutside yields outside a task.
	yieldingOutside = bodyOf(func(c *prop.Case) { stateful.NewScheduler(c, stateful.Uniform()).Yield() })
)

// schedulerAllocs are the cases of the allocation ceilings of the
// scheduler, each a whole replayed case.
var schedulerAllocs = []alloctest.Case{
	{Name: "NewScheduler", Call: func(assert.TB) { engine.Replay(scheduling, nil, nil) }, Allocs: schedulerCaseAllocs},
	{Name: "Spawn", Call: func(assert.TB) { engine.Replay(spawning, nil, nil) }, Allocs: spawnCaseAllocs},
	{Name: "Run", Call: func(assert.TB) { engine.Replay(running, nil, nil) }, Allocs: runCaseAllocs},
	{Name: "Yield", Call: func(assert.TB) { engine.Replay(yieldingOutside, nil, nil) }, Allocs: schedulerCaseAllocs},
}

// nothing is a task that does nothing.
func nothing() {}

// TestSchedulerAllocs checks the allocation ceilings of the scheduler.
func TestSchedulerAllocs(t *testing.T) {
	alloctest.Check(t, schedulerAllocs)
}

// BenchmarkScheduler measures the scheduler's functions, each on a whole
// replayed case.
func BenchmarkScheduler(b *testing.B) {
	for _, c := range schedulerAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
