// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import (
	"math"
	"runtime"
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// The bounds of the scheduler's choices.
var (
	// unsigned are the bounds of a task's priority under PCT, and of the
	// count of a change point.
	unsigned = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(math.MaxUint64))
	// presence are the bounds of the presence choice of a change point.
	presence = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1))
)

// ending is how a task's turn ended.
type ending uint8

const (
	// yielded is a turn that the task ended with Yield.
	yielded ending = 0
	// returned is a turn whose task returned.
	returned ending = 1
	// exited is a turn whose task ended its goroutine, and the end of a task
	// that the scheduler ended.
	exited ending = 2
	// panicked is a turn whose task panicked.
	panicked ending = 3
)

// turn is how a task's turn ended, and the value of a task's panic.
type turn struct {
	// how is how the turn ended.
	how ending
	// value is the value of the task's panic.
	value any
}

// task is a spawned task.
type task struct {
	// f is the task's function.
	f func()
	// priority is the task's priority under PCT.
	priority uint64
	// level is 0 until a change point lowers the task, and each lowering
	// gives it a level below every level before.
	level int
	// resume releases the task with true, and ends it with false.
	resume chan bool
}

// Scheduler releases the tasks of one case, one at a time, in an order that
// the case's choices decide, so a schedule replays and shrinks as the
// case's values do.
//
// A task is a goroutine that [Scheduler.Spawn] starts, and that runs only
// while the scheduler releases it: until it calls [Scheduler.Yield] or
// ends. [Scheduler.Run] makes the choice of each release, on the goroutine
// that calls it, and waits for each released task. A subject that runs as
// tasks starts its goroutines through Spawn, and calls Yield at each point
// where another task may run.
//
// The scheduler registers a cleanup with its case, which ends every task
// that waits for a release, so every goroutine of a case has ended once the
// case has.
//
// # Concurrency
//
// The goroutine that runs the body and the tasks call the methods one at a
// time, which the scheduler's releases ensure: one goroutine of the case
// runs at any moment.
type Scheduler struct {
	// c is the case whose choices decide the releases.
	c *engine.Case
	// strategy chooses the task to release.
	strategy Strategy
	// back receives the end of each turn and of each ended task.
	back chan turn
	// ready are the ready tasks, in the order they became ready.
	ready []*task
	// lowest is the level of the task that the last change point lowered.
	lowest int
	// running reports whether Run is releasing tasks.
	running bool
	// current is the released task, and nil while no task is released.
	current *task
}

// NewScheduler returns a scheduler of the case c that releases tasks by
// strategy.
//
// # Allocation contract
//
// NewScheduler allocates the scheduler, its channel and the cleanup that
// it registers with the case.
func NewScheduler(c *prop.Case, strategy Strategy) *Scheduler {
	s := &Scheduler{c: (*engine.Case)(c), strategy: strategy, back: make(chan turn)}
	c.Cleanup(s.end)
	return s
}

// Spawn starts the goroutine of the task f, which waits for its release,
// and makes the task ready after every ready task. Under PCT, the task
// first receives its priority, a choice of the case on the goroutine that
// calls Spawn.
//
// # Allocation contract
//
// Spawn allocates the task, its channel and its goroutine, and grows the
// ready tasks.
func (s *Scheduler) Spawn(f func()) {
	t := &task{f: f, resume: make(chan bool)}
	if s.strategy.depth > 0 {
		t.priority = s.c.Integer(unsigned).Magnitude()
	}
	go s.start(t)
	s.ready = append(s.ready, t)
}

// Yield ends the turn of the running task, which becomes ready again after
// every ready task and waits for its next release. Outside a task, on the
// goroutine that runs the body, it returns at once, so a step on client 0
// runs to its end.
//
// # Allocation contract
//
// Yield allocates nothing.
func (s *Scheduler) Yield() {
	t := s.current
	if t == nil {
		return
	}
	s.back <- turn{how: yielded}
	if !<-t.resume {
		runtime.Goexit()
	}
}

// Run makes the change points of PCT, and then releases one ready task at
// a time until none is ready. A task that returns lets Run release the
// next. A task that ends its goroutine, as a fatal record, a rejection or a
// draw that ends the case does, makes Run end every other task and then end
// its own goroutine the same way. A task that panics makes Run end every
// other task and then raise the panic again.
//
// # Panics
//
// Run panics when a task calls it on its own scheduler, and raises the
// panic of a task.
//
// # Allocation contract
//
// Run allocates the counts of the change points, and what the case's
// choices of the releases allocate.
func (s *Scheduler) Run() {
	if s.running {
		panic("stateful: a task called Run of its own scheduler")
	}
	s.running = true
	defer func() { s.running = false }()
	points := s.changePoints()
	for release := uint64(0); len(s.ready) > 0; release++ {
		t := s.next()
		for _, p := range points {
			if p == release {
				s.lowest--
				t.level = s.lowest
			}
		}
		switch r := s.release(t); r.how {
		case yielded:
			s.ready = append(s.ready, t)
		case exited:
			s.end()
			runtime.Goexit()
		case panicked:
			s.end()
			panic(r.value)
		}
	}
}

// changePoints returns the counts of the change points of a run: up to
// depth - 1 under PCT, each present by a choice with edge 1, and none under
// Uniform.
func (s *Scheduler) changePoints() []uint64 {
	var points []uint64
	for range s.strategy.depth - 1 {
		if s.c.Structure(presence, 1).Magnitude() == 1 {
			points = append(points, s.c.Integer(unsigned).Magnitude())
		}
	}
	return points
}

// next removes the task to release from the ready tasks and returns it:
// under PCT the task of the highest level and then the highest priority,
// the earliest ready among equals, and under Uniform the case's choice
// among the ready tasks.
func (s *Scheduler) next() *task {
	var i int
	if s.strategy.depth > 0 {
		i = s.highest()
	} else {
		i = int(s.c.Uniform(uint64(len(s.ready)), 0))
	}
	t := s.ready[i]
	s.ready = slices.Delete(s.ready, i, i+1)
	return t
}

// highest returns the index of the ready task of the highest level and then
// the highest priority, the earliest ready among equals.
func (s *Scheduler) highest() int {
	i := 0
	for j, t := range s.ready {
		if best := s.ready[i]; t.level > best.level || t.level == best.level && t.priority > best.priority {
			i = j
		}
	}
	return i
}

// release releases t and returns once its turn has ended.
func (s *Scheduler) release(t *task) turn {
	s.current = t
	t.resume <- true
	r := <-s.back
	s.current = nil
	return r
}

// start runs the task t on its goroutine once the scheduler releases it,
// and sends how its last turn ended. A task that the scheduler ends before
// its first release runs nothing.
func (s *Scheduler) start(t *task) {
	r := turn{how: exited}
	defer func() {
		if v := recover(); v != nil {
			r = turn{how: panicked, value: v}
		}
		s.back <- r
	}()
	if <-t.resume {
		t.f()
		r.how = returned
	}
}

// end ends every task that waits for a release, a task that an ending task
// spawns included, and returns once each has ended.
func (s *Scheduler) end() {
	for len(s.ready) > 0 {
		t := s.ready[0]
		s.ready = s.ready[1:]
		t.resume <- false
		<-s.back
	}
}
