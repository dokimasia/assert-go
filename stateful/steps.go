// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import (
	"math"
	"time"

	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/prop"
)

// unlimited is the time that a concurrent section on threads waits for its
// clients: the largest duration, so the test's own deadline applies.
const unlimited = time.Duration(math.MaxInt64)

// Steps runs the steps of the machine m in the case c, in six parts, and
// records every decision as a choice of the case:
//
//  1. Swarm: one choice per action, in order, whether the case keeps it,
//     with odds of 1 in 2 conditioned on keeping one. A case keeps every
//     action under Swarm(false).
//  2. Setup: the check of the history, and the invariant.
//  3. Sequential steps on client 0. Before each step, the kept actions that
//     are enabled in every state that the last check left. With none, the
//     steps end. Otherwise a continue flag, with the mean of [Mean] and the
//     most steps of [Max], and an index into the list by weight. The step
//     is a span labelled with the action's name, from its flag to the end
//     of its run, which requests the step's input and runs the action. The
//     check and the invariant follow each step.
//  4. A concurrent section, for [Clients] of 2 or more. The section lists
//     its steps before any runs: each a flag with the most steps of
//     [Concurrent], a client from 0 to Clients, an index into the kept
//     actions without Enabled, and the step's input. Client 0 runs its steps
//     on the goroutine of the body, and then the other clients run theirs at
//     once. The check follows.
//  5. The drain: steps of the actions with Drain set, without a flag, until
//     none is enabled or Max steps ran, each followed by the check and the
//     invariant.
//  6. Settle, and the invariant.
//
// The check runs [history.Linearizable] on the case's history and the
// machine's spec, with every call in one partition, and the next state is
// the first of the states that the linearization it found leaves. Each check
// continues the search of the check before it through [history.Resume], so
// it searches the calls since that check, and reports what a search of the
// whole history reports. A check that fails, violated or undecided, ends
// the case with the record of linearizable, whose contract states that the
// history of the machine's steps is linearizable.
//
// The clients of a concurrent section run on threads, through
// [history.Concurrently], and the case then runs up to [Repeat] times,
// because threads do not replay. Under [Tasks], they run as tasks of a
// [Scheduler], one task per client that has a step, spawned in client
// order.
//
// The case of [prop.Draws] follows its trace: the swarm keeps the actions
// that its step entries name, and each step entry becomes the choices of
// one step. Steps refuses a step entry that it cannot take at its position,
// which ends the run with a fault at the entry's step before any other case:
// an action that the step's list lacks, a step past its part's most steps,
// a client outside the clients, a concurrent step of a machine with one
// client, and a drain step after the drain.
//
// Each request that the case makes while Steps runs, and each fingerprint
// that it observes, belongs to the part and the step that run, which the
// divergence of a flaky run names: a sequential, concurrent or drain step
// at its position in its part, from 0, with its action once its index has
// chosen it, and a swarm choice at the position of its action. Setup,
// settle and the run of a concurrent section's steps have no position.
//
// # Panics
//
// Steps panics for a machine that states no machine: an action with a
// negative weight or without Run, and two actions with one name. The panic
// fails the case.
//
// # Allocation contract
//
// Steps allocates the record of its run and the lists of its actions, and
// each step allocates its span, its input and what its check allocates.
func Steps[S any](c *prop.Case, m Machine[S], opts ...Option) {
	m.validate()
	r := &run[S]{c: c, e: (*engine.Case)(c), m: m, cfg: configure(opts), states: make([]S, 1)}
	r.seat.TB = c
	r.checks = []history.Option{history.Whole(), history.Final(&r.states), history.Resume(&r.checkpoint)}
	r.trace, r.traced = r.e.Trace()
	r.steps()
}

// run is one run of a machine's steps in one case.
type run[S any] struct {
	// c is the case, and e the same case as the engine's.
	c *prop.Case
	e *engine.Case
	// m is the machine.
	m Machine[S]
	// cfg are the settings of the run.
	cfg config
	// trace is the trace that the case follows, when traced is set.
	trace  engine.Trace
	traced bool
	// states are the states that the last check left, and one zero S before
	// a check and for a machine without a spec.
	states []S
	// seat is the seat of the check, and checks are its options.
	seat   seat
	checks []history.Option
	// checkpoint keeps the search of the last check that passed, which the
	// next check continues.
	checkpoint history.Checkpoint[S]
	// weights are the weights of the last index, kept for the next.
	weights []uint64
}

// plannedStep is a step of a concurrent section, listed before any step
// runs: its action and its input.
type plannedStep[S any] struct {
	// action is the step's action.
	action Action[S]
	// input is the step's input.
	input any
}

// steps runs the six parts in order, and leaves the case outside the steps
// when they end, however they end.
func (r *run[S]) steps() {
	defer r.e.ClearPlace()
	kept := r.swarm()
	r.e.SetPlace(engine.Place{Part: engine.SetupPart})
	r.check()
	r.invariant()
	r.sequential(kept)
	r.concurrent(kept)
	r.drain()
	r.e.SetPlace(engine.Place{Part: engine.SettlePart})
	if r.m.Settle != nil {
		r.m.Settle(r.c, r.states[0])
	}
	r.invariant()
}

// at makes the case's place the step at position of part, before the step
// has chosen its action.
func (r *run[S]) at(part engine.Part, position int) {
	r.e.SetPlace(engine.Place{Part: part, Position: position, Positioned: true})
}

// acting makes the case's place the step at position of part, which takes
// the action a.
func (r *run[S]) acting(part engine.Part, position int, a Action[S]) {
	r.e.SetPlace(engine.Place{Part: part, Position: position, Positioned: true, Action: a.Name, Acting: true})
}

// swarm returns the actions that the case keeps, in order. A case that
// follows a trace keeps the actions that its step entries name.
func (r *run[S]) swarm() []Action[S] {
	actions := r.m.Actions
	if !r.cfg.swarm {
		return actions
	}
	var kept []Action[S]
	for i, a := range actions {
		r.acting(engine.SwarmPart, i, a)
		if r.traced {
			named := uint64(0)
			if r.trace.Names(a.Name) {
				named = 1
			}
			r.trace.Prepare(named)
		}
		if r.e.Keep(len(kept) > 0, len(actions)-i) {
			kept = append(kept, a)
		}
	}
	return kept
}

// sequential takes sequential steps until no kept action is enabled or a
// flag stops them.
func (r *run[S]) sequential(kept []Action[S]) {
	for count := 0; ; count++ {
		r.at(engine.SequentialPart, count)
		available := r.available(kept)
		start := r.e.Position()
		if r.traced {
			r.followSequential(available, count)
		}
		if len(available) == 0 || !r.e.Continue(count, r.cfg.max, r.cfg.mean) {
			return
		}
		r.take(start, available, engine.SequentialPart, count)
	}
}

// concurrent lists the steps of the concurrent section, runs them, and
// checks the history.
func (r *run[S]) concurrent(kept []Action[S]) {
	if r.cfg.clients == 1 {
		if r.traced {
			r.refuseConcurrent()
		}
		return
	}
	var eligible []Action[S]
	for _, a := range kept {
		if a.Enabled == nil {
			eligible = append(eligible, a)
		}
	}
	sizes, _ := choice.NewSizes(0, r.cfg.concurrent)
	average := random.Average(sizes)
	planned := make([][]plannedStep[S], r.cfg.clients+1)
	for count := 0; ; count++ {
		r.at(engine.ConcurrentPart, count)
		start := r.e.Position()
		if r.traced {
			r.followConcurrent(eligible, count)
		}
		if len(eligible) == 0 || !r.e.Continue(count, r.cfg.concurrent, average) {
			break
		}
		client := int(r.e.Uniform(uint64(r.cfg.clients)+1, 0))
		a := eligible[r.index(eligible)]
		r.acting(engine.ConcurrentPart, count, a)
		r.e.SpanFrom(start, a.Name, func() {
			r.e.Step(engine.MachineStep{Action: a.Name, Client: client})
			planned[client] = append(planned[client], plannedStep[S]{action: a, input: r.input(a)})
		})
	}
	r.e.SetPlace(engine.Place{Part: engine.ConcurrentPart})
	runSteps(r.c, 0, planned[0])
	r.section(planned)
	r.check()
}

// section runs the steps of the clients 1 to Clients that have one, at
// once: as tasks of the scheduler, spawned in client order, or on threads,
// for which the case asks to repeat.
func (r *run[S]) section(planned [][]plannedStep[S]) {
	var active []int
	for client := 1; client < len(planned); client++ {
		if len(planned[client]) > 0 {
			active = append(active, client)
		}
	}
	if s := r.cfg.scheduler; s != nil {
		for _, client := range active {
			s.Spawn(func() { runSteps(r.c, client, planned[client]) })
		}
		s.Run()
		return
	}
	if len(active) == 0 {
		return
	}
	r.e.Repeat(r.cfg.repeat)
	history.Concurrently(len(active), unlimited, func(i int) (any, error) {
		runSteps(r.c, active[i], planned[active[i]])
		return nil, nil //nolint:nilnil // a client's steps have no output for the history
	})
}

// drain takes drain steps until no drain action is enabled or Max steps
// ran.
func (r *run[S]) drain() {
	var drains []Action[S]
	for _, a := range r.m.Actions {
		if a.Drain {
			drains = append(drains, a)
		}
	}
	for count := 0; ; count++ {
		r.at(engine.DrainPart, count)
		var available []Action[S]
		if count < r.cfg.max {
			available = r.available(drains)
		}
		if r.traced {
			r.followDrain(available)
		}
		if len(available) == 0 {
			return
		}
		r.take(r.e.Position(), available, engine.DrainPart, count)
	}
}

// take takes the step at position of part, sequential or drain, on client 0,
// of the action of listed that the case's index chooses: a span labelled
// with the action's name from start, which records the step, requests its
// input and runs it, and then the check and the invariant.
func (r *run[S]) take(start int, listed []Action[S], part engine.Part, position int) {
	a := listed[r.index(listed)]
	r.acting(part, position, a)
	r.e.SpanFrom(start, a.Name, func() {
		r.e.Step(engine.MachineStep{Action: a.Name, Client: -1, Drain: part == engine.DrainPart})
		a.Run(r.c, 0, r.input(a))
	})
	r.check()
	r.invariant()
}

// available returns the actions of listed that are enabled in every state
// that the last check left. It calls each Enabled once per state.
func (r *run[S]) available(listed []Action[S]) []Action[S] {
	var out []Action[S]
	for _, a := range listed {
		enabled := true
		if a.Enabled != nil {
			for _, s := range r.states {
				enabled = a.Enabled(s) && enabled
			}
		}
		if enabled {
			out = append(out, a)
		}
	}
	return out
}

// index returns the case's choice of an action of listed, by weight.
func (r *run[S]) index(listed []Action[S]) int {
	r.weights = r.weights[:0]
	for _, a := range listed {
		r.weights = append(r.weights, uint64(max(a.Weight, 1)))
	}
	return r.e.Weighted(r.weights)
}

// input returns the input that a requests from the case, and nil for an
// action without Input.
func (r *run[S]) input(a Action[S]) any {
	if a.Input == nil {
		return nil
	}
	return a.Input(r.c, r.states[0])
}

// invariant runs the machine's invariant, when it has one.
func (r *run[S]) invariant() {
	if r.m.Invariant != nil {
		r.m.Invariant(r.c, r.states[0])
	}
}

// runSteps runs the planned steps of client in order.
func runSteps[S any](c *prop.Case, client int, planned []plannedStep[S]) {
	for _, p := range planned {
		p.action.Run(c, client, p.input)
	}
}
