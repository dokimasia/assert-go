// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import (
	"math"
	"runtime"
	"runtime/debug"
	"slices"
	"time"

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

// epoch is the instant at which the clock of every scheduler starts, which
// the definition fixes: midnight UTC 2000-01-01, the first instant of a
// testing/synctest bubble.
var epoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

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
	// parked is a turn that the task ended with a wait for a condition, or
	// with a sleep on the clock of its scheduler.
	parked ending = 4
)

// maxFrames is the most frames of a task's goroutine that a turn keeps of
// the task's panic.
const maxFrames = 64

// turn is how a task's turn ended, and where the task raised its panic.
type turn struct {
	// how is how the turn ended.
	how ending
	// value is the value of the task's panic.
	value any
	// pcs are the frames of the task's goroutine at the panic.
	pcs []uintptr
	// stack is the stack of the task's goroutine at the panic, as
	// runtime/debug.Stack formats it.
	stack []byte
}

// task is a spawned task.
type task struct {
	// deadline is the instant that a sleeping task sleeps until.
	deadline time.Time
	// f is the task's function.
	f func()
	// cond is the condition that a waiting task waits for, and nil for a task
	// that does not wait.
	cond func() bool
	// resume releases the task with true, and ends it with false.
	resume chan bool
	// priority is the task's priority under PCT.
	priority uint64
	// level is 0 until a change point lowers the task, and each lowering
	// gives it a level below every level before.
	level int
}

// Scheduler releases the tasks of one case, one at a time, in an order that
// the case's choices decide, so a schedule replays and shrinks as the
// case's values do.
//
// A task is a goroutine that [Scheduler.Spawn] starts, and that runs only
// while the scheduler releases it: until it calls [Scheduler.Yield], parks or
// ends. A task parks until a condition is true with [Scheduler.Wait], and
// until a deadline with the Sleep of [Scheduler.Clock]. A parked task is not
// ready, so [Scheduler.Run] returns while it waits or sleeps. Run makes the
// choice of each release, on the goroutine that calls it, and waits for each
// released task. A subject that runs as tasks starts its goroutines through
// Spawn, calls Yield at each point where another task may run, and waits and
// sleeps through the scheduler.
//
// The scheduler checks its parked tasks at the start of each Run and each
// [Scheduler.Advance], after each turn, and after each move of its clock. It
// makes ready, after every ready task, each sleeping task whose deadline the
// clock has passed, by deadline and then in the order of the sleeps, and then
// each waiting task whose condition returns true, in the order of the waits.
// A task keeps its priority under PCT through a wait and a sleep.
//
// The scheduler registers a cleanup with its case, which ends every task
// that waits for a release, waits for a condition or sleeps, so every
// goroutine of a case has ended once the case has. A task that panics as it
// ends, in a deferred call, fails the case with its panic.
//
// # Concurrency
//
// The goroutine that runs the body and the tasks call the methods one at a
// time, which the scheduler's releases ensure: one goroutine of the case
// runs at any moment.
type Scheduler struct {
	// now is the instant of the scheduler's clock.
	now time.Time
	// c is the case whose choices decide the releases.
	c *engine.Case
	// back receives the end of each turn and of each ended task.
	back chan turn
	// current is the released task, and nil while no task is released.
	current *task
	// ready are the ready tasks, in the order they became ready.
	ready []*task
	// waiting are the tasks that wait for a condition, in the order of their
	// waits.
	waiting []*task
	// sleeping are the tasks that sleep until a deadline, by deadline and then
	// in the order of their sleeps.
	sleeping []*task
	// strategy chooses the task to release.
	strategy Strategy
	// maxReleases is the most tasks that one Run or one Advance releases.
	maxReleases uint64
	// lowest is the level of the task that the last change point lowered.
	lowest int
	// running reports whether Run or Advance is releasing tasks.
	running bool
}

// NewScheduler returns a scheduler of the case c that releases tasks by
// strategy, under the options opts. Its clock reads midnight UTC 2000-01-01.
//
// # Allocation contract
//
// NewScheduler allocates the scheduler, its channel and the cleanup that
// it registers with the case.
func NewScheduler(c *prop.Case, strategy Strategy, opts ...SchedulerOption) *Scheduler {
	s := &Scheduler{
		c: (*engine.Case)(c), strategy: strategy, back: make(chan turn), now: epoch,
		maxReleases: defaultMaxReleases,
	}
	for _, o := range opts {
		if o.maxReleases > 0 {
			s.maxReleases = uint64(o.maxReleases)
		}
	}
	c.Cleanup(s.cleanUp)
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
	t.await()
}

// Wait parks the running task until cond returns true. It returns at once
// when cond returns true at the call. The scheduler calls cond again between
// two turns, on the goroutine that calls Run or Advance, and makes the task
// ready once cond returns true. Outside a task, on the goroutine that runs
// the body, Wait returns at once when cond returns true.
//
// # Panics
//
// Wait panics outside a task when cond returns false, because no task runs
// while the body waits.
//
// # Allocation contract
//
// Wait allocates nothing.
func (s *Scheduler) Wait(cond func() bool) {
	if cond() {
		return
	}
	t := s.current
	if t == nil {
		panic("stateful: Wait outside a task waits for a condition that is false")
	}
	t.cond = cond
	s.park(t)
}

// Run makes the change points of PCT, and then releases one ready task at
// a time until none is ready. A task that returns lets Run release the
// next. A task that ends its goroutine, as a fatal record, a rejection or a
// draw that ends the case does, makes Run end every other task and then end
// its own goroutine the same way. A task that panics makes Run end every
// other task and then raise the panic again. The case keeps a panic that
// Run raises again with the place where the task raised it.
//
// A Run that has released the limit of [MaxReleases] while a task is still
// ready ends every task, and fails the case with a record without an
// assertion whose contract is "the task scheduler released n tasks in one
// run, and a task is still ready".
//
// # Panics
//
// Run panics when a task calls it on its own scheduler, and raises the
// panic of a task: of the task that panicked in its turn, or of the first
// task that panicked as Run ended it. It raises the panic of a condition
// that it calls.
//
// # Allocation contract
//
// Run allocates the counts of the change points, and what the case's
// choices of the releases allocate.
func (s *Scheduler) Run() {
	s.begin("stateful: a task called Run of its own scheduler")
	defer func() { s.running = false }()
	s.wake()
	s.releaseAll(s.changePoints(), 0)
}

// Advance releases the ready tasks as [Scheduler.Run] does, and moves the
// clock to each deadline of a sleeping task up to d from its reading at the
// call, in the order of the deadlines. The tasks of each deadline run at the
// time of that deadline. Advance returns once the clock reads its end time
// and no task is ready. A duration of zero or less moves the clock by
// nothing. Under PCT, Advance makes the change points of one run, and counts
// its releases from 0 over every deadline. It fails the case at the limit of
// [MaxReleases] as Run does.
//
// # Panics
//
// Advance panics when a task calls it on its own scheduler, and raises the
// panic of a task and of a condition as Run does.
//
// # Allocation contract
//
// Advance allocates the counts of the change points, and what the case's
// choices of the releases allocate.
func (s *Scheduler) Advance(d time.Duration) {
	s.begin("stateful: a task called Advance of its own scheduler")
	defer func() { s.running = false }()
	end := s.now.Add(max(d, 0))
	s.wake()
	points := s.changePoints()
	for release := uint64(0); ; {
		release = s.releaseAll(points, release)
		s.now = end
		if len(s.sleeping) > 0 && s.sleeping[0].deadline.Before(end) {
			s.now = s.sleeping[0].deadline
		}
		s.wake()
		// A move to a deadline before end makes the task of that deadline
		// ready, so no task is ready only once the clock reads end.
		if len(s.ready) == 0 {
			return
		}
	}
}

// begin starts a Run or an Advance, and panics with message when a task of
// the scheduler calls it.
func (s *Scheduler) begin(message string) {
	if s.running {
		panic(message)
	}
	s.running = true
}

// releaseAll releases one ready task at a time until none is ready, and
// returns the count of the releases of the run after the last one, which it
// counts from release. It fails the case once the count reaches the limit
// while a task is still ready.
func (s *Scheduler) releaseAll(points []uint64, release uint64) uint64 {
	for ; len(s.ready) > 0; release++ {
		if release == s.maxReleases {
			s.raise(s.end())
			s.c.Fatalf("the task scheduler released %d tasks in one run, and a task is still ready", release)
		}
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
		case parked:
			s.keep(t)
		case exited:
			s.raise(s.end())
			runtime.Goexit()
		case panicked:
			s.end()
			kept := r
			s.raise(&kept)
		case returned:
			// A task that returned has ended, so it leaves the ready tasks.
		}
		s.wake()
	}
	return release
}

// keep keeps the parked task t: among the waiting tasks after every one of
// them, or among the sleeping tasks after every task whose deadline is not
// after its own.
func (s *Scheduler) keep(t *task) {
	if t.cond != nil {
		s.waiting = append(s.waiting, t)
		return
	}
	at := len(s.sleeping)
	for i, other := range s.sleeping {
		if other.deadline.After(t.deadline) {
			at = i
			break
		}
	}
	s.sleeping = slices.Insert(s.sleeping, at, t)
}

// wake makes ready, after every ready task, each sleeping task whose
// deadline the clock has passed, in the order of the sleeping tasks, and
// then each waiting task whose condition returns true, in the order of the
// waits.
func (s *Scheduler) wake() {
	due := 0
	for due < len(s.sleeping) && !s.sleeping[due].deadline.After(s.now) {
		due++
	}
	s.ready = append(s.ready, s.sleeping[:due]...)
	s.sleeping = slices.Delete(s.sleeping, 0, due)
	still := s.waiting[:0]
	for _, t := range s.waiting {
		if t.cond() {
			t.cond = nil
			s.ready = append(s.ready, t)
			continue
		}
		still = append(still, t)
	}
	clear(s.waiting[len(still):])
	s.waiting = still
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

// park ends the turn of the running task t, which the scheduler keeps as a
// waiting or a sleeping task, and returns at its next release.
func (s *Scheduler) park(t *task) {
	s.back <- turn{how: parked}
	t.await()
}

// await waits for the next release of the task, and ends its goroutine when
// the scheduler ends the task instead.
func (t *task) await() {
	if !<-t.resume {
		runtime.Goexit()
	}
}

// start runs the task t on its goroutine once the scheduler releases it,
// and sends how its last turn ended, with where the task raised a panic. A
// task that the scheduler ends before its first release runs nothing.
func (s *Scheduler) start(t *task) {
	r := turn{how: exited}
	defer func() {
		if v := recover(); v != nil {
			var pcs [maxFrames]uintptr
			n := runtime.Callers(1, pcs[:])
			r = turn{how: panicked, value: v, pcs: slices.Clone(pcs[:n]), stack: debug.Stack()}
		}
		s.back <- r
	}()
	if <-t.resume {
		t.f()
		r.how = returned
	}
}

// end ends every task that waits for a release, waits for a condition or
// sleeps, a task that an ending task spawns included, and returns once each
// has ended. It returns the turn of the first task that panicked as it
// ended, in a deferred call, and nil when none did.
func (s *Scheduler) end() *turn {
	var first *turn
	for t := s.take(); t != nil; t = s.take() {
		t.resume <- false
		if r := <-s.back; r.how == panicked && first == nil {
			kept := r
			first = &kept
		}
	}
	return first
}

// take removes and returns a task that has not ended: the first ready task,
// then the first waiting task, then the first sleeping task, and nil when no
// task is left.
func (s *Scheduler) take() *task {
	for _, tasks := range [...]*[]*task{&s.ready, &s.waiting, &s.sleeping} {
		if len(*tasks) > 0 {
			t := (*tasks)[0]
			*tasks = (*tasks)[1:]
			return t
		}
	}
	return nil
}

// cleanUp ends every task that has not ended when the case ends, and raises
// again the panic of the first task that panicked as it ended.
func (s *Scheduler) cleanUp() {
	s.raise(s.end())
}

// raise raises the panic of the turn r again, on the calling goroutine,
// with the place where the task raised it, and does nothing for nil.
func (s *Scheduler) raise(r *turn) {
	if r != nil {
		s.c.Raise(r.value, r.pcs, r.stack)
	}
}
