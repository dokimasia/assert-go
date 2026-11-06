// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"runtime"
	"testing"
	"time"
	"weak"

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

	t.Run("Wait", func(t *testing.T) {
		t.Parallel()

		t.Run("parks the running task until another task makes its condition true", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				set := false
				s.Spawn(waiting(s, log, "a", func() bool { return set }))
				s.Spawn(func() {
					log.add("b", 0)
					set = true
				})
				s.Run()
			})
			assert.Equal(t, log.all(), suffixed("a1", "b", "a2"), "a runs again after the turn of b")
		})

		t.Run("returns at once for a condition that is true, without a release", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(waiting(s, log, "a", func() bool { return true }))
				s.Run()
			})
			assert.Equal(t, log.all(), suffixed("a1", "a2"), "the task runs through its wait")
			assert.Length(t, choicesOf(e), 1, "one release")
		})

		t.Run("lets Run return while a task waits, and wakes the task at the next Run", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			var first []string
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				set := false
				s.Spawn(waiting(s, log, "a", func() bool { return set }))
				s.Run()
				first = log.all()
				set = true
				s.Run()
			})
			assert.Equal(t, first, suffixed("a1"), "the first Run returns while a waits")
			assert.Equal(t, log.all(), suffixed("a1", "a2"), "the second Run wakes a")
		})

		t.Run("makes the woken tasks ready after the ready tasks, in the order of their waits", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				set := false
				s.Spawn(waiting(s, log, "w", func() bool { return set }))
				s.Spawn(waiting(s, log, "v", func() bool { return set }))
				s.Spawn(func() {
					log.add("s1", 0)
					set = true
					s.Yield()
					log.add("s2", 0)
				})
				s.Run()
			})
			assert.Equal(t, log.all(), suffixed("w1", "v1", "s1", "s2", "w2", "v2"),
				"s yields before the waits end, so it runs before w and v")
		})

		t.Run("keeps the priority of a task through its wait under PCT", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.PCT(1), func(_ *prop.Case, s *stateful.Scheduler) {
				set := false
				s.Spawn(waiting(s, log, "a", func() bool { return set }))
				s.Spawn(func() {
					log.add("b1", 0)
					set = true
					s.Yield()
					log.add("b2", 0)
				})
				s.Spawn(func() { log.add("c", 0) })
				s.Run()
			}, 9, 1, 1)
			assert.Equal(t, log.all(), suffixed("a1", "b1", "a2", "c", "b2"),
				"a of the highest priority runs once its wait ends, and c became ready before b yielded")
		})

		t.Run("clears the condition of a woken task, so its next sleep lasts until its deadline", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			var first []string
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				set := false
				s.Spawn(func() {
					s.Wait(func() bool { return set })
					s.Clock().Sleep(4 * time.Millisecond)
					log.add("slept", 0)
				})
				s.Run()
				set = true
				s.Run()
				first = log.all()
				s.Advance(4 * time.Millisecond)
			})
			assert.Equal(t, first, []string(nil), "the second Run wakes the task, which then sleeps")
			assert.Equal(t, log.all(), suffixed("slept"), "the advance ends the sleep")
		})

		t.Run("keeps no reference to a woken task once it has returned", func(t *testing.T) {
			t.Parallel()
			var collected bool
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				set := false
				s.Spawn(func() { s.Wait(func() bool { return false }) })
				s.Run()
				payload := holding(s, func() bool { return set })
				s.Run()
				set = true
				s.Run()
				runtime.GC()
				collected = payload.Value() == nil
			})
			assert.True(t, collected, "the collector frees the value of the task that returned")
		})

		t.Run("returns at once outside a task for a condition that is true", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Wait(func() bool { return true })
				log.add("after", 0)
			})
			assert.Equal(t, log.all(), suffixed("after"), "the body runs on")
		})

		t.Run("panics outside a task for a condition that is false", func(t *testing.T) {
			t.Parallel()
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Wait(func() bool { return false })
			})
			assert.Equal(t, e.Panic, any("stateful: Wait outside a task waits for a condition that is false"),
				"the panic fails the case")
		})

		t.Run("raises the panic of a condition that the scheduler calls between turns", func(t *testing.T) {
			t.Parallel()
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				calls := 0
				s.Spawn(waiting(s, &runLog{}, "a", func() bool {
					if calls++; calls == 2 {
						panic("broken condition")
					}
					return false
				}))
				s.Spawn(func() {})
				s.Run()
			})
			assert.Equal(t, e.Panic, any("broken condition"), "the panic fails the case")
		})
	})

	t.Run("Advance", func(t *testing.T) {
		t.Parallel()

		t.Run("runs each sleeping task at its own deadline, up to the end time", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			var first []string
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(ticking(s, log, "t", 4*time.Millisecond, 3))
				s.Advance(10 * time.Millisecond)
				first = log.all()
				s.Advance(2 * time.Millisecond)
			})
			assert.Equal(t, first, suffixed("t@4ms", "t@8ms"), "two ticks within 10 ms")
			assert.Equal(t, log.all(), suffixed("t@4ms", "t@8ms", "t@12ms"), "the third tick within 2 ms more")
		})

		t.Run("makes ready at its start each waiting task whose condition returns true", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				set := false
				s.Spawn(waiting(s, log, "w", func() bool { return set }))
				s.Run()
				set = true
				s.Spawn(func() {
					log.add("r", 0)
					set = false
				})
				s.Advance(0)
			})
			assert.Equal(t, log.all(), suffixed("w1", "r", "w2"),
				"w became ready before r made its condition false again")
		})

		t.Run("ends a sleep whose deadline is the end time within the advance", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(ticking(s, log, "t", 4*time.Millisecond, 1))
				s.Advance(4 * time.Millisecond)
			})
			assert.Equal(t, log.all(), suffixed("t@4ms"), "the tick at the end time")
		})

		t.Run("wakes the tasks of a deadline in the order of their sleeps, after an earlier deadline",
			func(t *testing.T) {
				t.Parallel()
				log := &runLog{}
				scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
					s.Spawn(ticking(s, log, "c", 6*time.Millisecond, 1))
					s.Spawn(ticking(s, log, "a", 2*time.Millisecond, 1))
					s.Spawn(ticking(s, log, "b", 6*time.Millisecond, 1))
					s.Advance(6 * time.Millisecond)
				})
				assert.Equal(t, log.all(), suffixed("a@2ms", "c@6ms", "b@6ms"), "a first, then c and b in sleep order")
			})

		t.Run("releases the ready tasks as Run does for no time, and moves the clock by nothing", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			var now time.Time
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(yielding(s, log, "a"))
				s.Advance(-time.Second)
				now = s.Clock().Now()
			}, 0, 0)
			assert.Equal(t, log.all(), suffixed("a1", "a2"), "both turns of a")
			assert.Equal(t, now, epoch, "the clock reads its start")
		})

		t.Run("makes the change points of one run under PCT, over every deadline", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := scheduled(stateful.PCT(2), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(ticking(s, log, "t", 4*time.Millisecond, 2))
				s.Advance(10 * time.Millisecond)
			}, 0, 0)
			assert.Equal(t, log.all(), suffixed("t@4ms", "t@8ms"), "both ticks")
			assert.Equal(t, choicesOf(e), []uint64{0, 0}, "the priority, and the presence of one change point")
		})

		t.Run("panics when a task calls Advance of its own scheduler", func(t *testing.T) {
			t.Parallel()
			e := scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(func() { s.Advance(time.Second) })
				s.Run()
			})
			assert.Equal(t, e.Panic, any("stateful: a task called Advance of its own scheduler"),
				"the panic fails the case")
		})

		t.Run("fails the case once it has released the limit of tasks while a task is still ready",
			func(t *testing.T) {
				t.Parallel()
				e := limited(2, func(_ *prop.Case, s *stateful.Scheduler) {
					s.Spawn(ticking(s, &runLog{}, "t", time.Millisecond, 3))
					s.Advance(5 * time.Millisecond)
				})
				assert.Equal(t, e.Case.Failures()[0].Contract,
					"the task scheduler released 2 tasks in one run, and a task is still ready",
					"the third tick passes the limit of two releases")
			})
	})

	t.Run("NewScheduler", func(t *testing.T) {
		t.Parallel()

		t.Run("fails the case once a Run has released the limit of tasks while a task is still ready",
			func(t *testing.T) {
				t.Parallel()
				log := &runLog{}
				e := limited(5, func(_ *prop.Case, s *stateful.Scheduler) {
					s.Spawn(spinning(s, log, "a"))
					s.Run()
					log.add("after", 0)
				})
				assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
				assert.Equal(t, e.Case.Failures()[0], assert.Failure{
					Contract: "the task scheduler released 5 tasks in one run, and a task is still ready",
					Where:    e.Case.Failures()[0].Where,
				}, "a record without an assertion")
				assert.Equal(t, log.all(), suffixed("a ended"), "the spinning task ends, and Run returns no more")
				assert.Length(t, choicesOf(e), 5, "five releases")
			})

		t.Run("ends every task before it fails the case at the limit", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			var atFailure []string
			limited(2, func(c *prop.Case, s *stateful.Scheduler) {
				s.Spawn(spinning(s, log, "a"))
				c.Cleanup(func() { atFailure = log.all() })
				s.Run()
			})
			assert.Equal(t, atFailure, suffixed("a ended"),
				"the task has ended when the cleanups of the case start")
		})

		t.Run("counts the releases of each Run apart", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := limited(2, func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(yielding(s, log, "a"))
				s.Run()
				s.Spawn(yielding(s, log, "b"))
				s.Run()
			}, 0, 0, 0, 0)
			assert.Equal(t, e.Status, engine.CasePassed, "two Runs of two releases each")
			assert.Equal(t, log.all(), suffixed("a1", "a2", "b1", "b2"), "every turn")
		})

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

		t.Run("ends the tasks that wait for a condition or sleep when the case ends", func(t *testing.T) {
			check := assert.NoGoroutineLeaks(t, "every task has ended")
			log := &runLog{}
			scheduled(stateful.Uniform(), func(_ *prop.Case, s *stateful.Scheduler) {
				s.Spawn(waiting(s, log, "w", func() bool { return false }))
				s.Spawn(ticking(s, log, "t", time.Hour, 1))
				s.Run()
			})
			check()
			assert.Equal(t, log.all(), suffixed("w1"), "neither task returns from its wait or its sleep")
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

// limited runs body with a uniform scheduler whose limit of releases is n, in
// a case that replays the integer choices, and returns how the case ended.
func limited(n int, body func(c *prop.Case, s *stateful.Scheduler), choices ...uint64) engine.Execution {
	return replayed(func(c *prop.Case) {
		body(c, stateful.NewScheduler(c, stateful.Uniform(), stateful.MaxReleases(n)))
	}, choices...)
}

// waiting returns a task named name that logs name1, waits until cond
// returns true, and logs name2.
func waiting(s *stateful.Scheduler, log *runLog, name string, cond func() bool) func() {
	return func() {
		log.add(name+"1", 0)
		s.Wait(cond)
		log.add(name+"2", 0)
	}
}

// holding spawns a task on s that waits until cond returns true and then
// writes a value of its own, and returns a weak pointer to the value. Only
// the task refers to the value.
func holding(s *stateful.Scheduler, cond func() bool) weak.Pointer[[64]byte] {
	value := new([64]byte)
	s.Spawn(func() {
		s.Wait(cond)
		value[0] = 1
	})
	return weak.Make(value)
}

// spinning returns a task named name that yields without an end, and logs
// "name ended" when the scheduler ends it.
func spinning(s *stateful.Scheduler, log *runLog, name string) func() {
	return func() {
		defer log.add(name+" ended", 0)
		for {
			s.Yield()
		}
	}
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
	// waitingOutside waits outside a task for a condition that is true.
	waitingOutside = bodyOf(func(c *prop.Case) {
		stateful.NewScheduler(c, stateful.Uniform()).Wait(func() bool { return true })
	})
	// advancing advances the clock of a scheduler without a task.
	advancing = bodyOf(func(c *prop.Case) { stateful.NewScheduler(c, stateful.Uniform()).Advance(time.Second) })
)

// schedulerAllocs are the cases of the allocation ceilings of the
// scheduler, each a whole replayed case.
var schedulerAllocs = []alloctest.Case{
	{Name: "NewScheduler", Call: func(assert.TB) { engine.Replay(scheduling, nil, nil) }, Allocs: schedulerCaseAllocs},
	{Name: "Spawn", Call: func(assert.TB) { engine.Replay(spawning, nil, nil) }, Allocs: spawnCaseAllocs},
	{Name: "Run", Call: func(assert.TB) { engine.Replay(running, nil, nil) }, Allocs: runCaseAllocs},
	{Name: "Yield", Call: func(assert.TB) { engine.Replay(yieldingOutside, nil, nil) }, Allocs: schedulerCaseAllocs},
	{Name: "Wait", Call: func(assert.TB) { engine.Replay(waitingOutside, nil, nil) }, Allocs: schedulerCaseAllocs},
	{Name: "Advance", Call: func(assert.TB) { engine.Replay(advancing, nil, nil) }, Allocs: schedulerCaseAllocs},
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
