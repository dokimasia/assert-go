// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import (
	"fmt"
	"slices"
)

// followSequential serves the flag and the index of the trace's next entry
// when it is a sequential step entry. It refuses the entry once the
// sequential steps are at their maximum, and for an action that listed
// lacks.
func (r *run[S]) followSequential(listed []Action[S], count int) {
	step, ok := r.trace.Next()
	if !ok || step.Client >= 0 || step.Drain {
		return
	}
	if count == r.cfg.max {
		r.trace.Refuse(fmt.Sprintf("the sequential steps are at their maximum of %d", r.cfg.max))
	}
	index := indexOf(listed, step.Action)
	if index < 0 {
		r.trace.Refuse(fmt.Sprintf("%q is not among the actions that the step lists", step.Action))
	}
	r.trace.Take(1, uint64(index))
}

// refuseConcurrent refuses the trace's next entry when it is a concurrent
// step entry, for a machine that runs no concurrent section.
func (r *run[S]) refuseConcurrent() {
	if step, ok := r.trace.Next(); ok && step.Client >= 0 && !step.Drain {
		r.trace.Refuse("the machine runs no concurrent section")
	}
}

// followConcurrent serves the flag, the client and the index of the trace's
// next entry when it is a concurrent step entry. It refuses the entry once
// the section is at its maximum, for a client outside the clients, and for
// an action that eligible lacks.
func (r *run[S]) followConcurrent(eligible []Action[S], count int) {
	step, ok := r.trace.Next()
	if !ok || step.Client < 0 || step.Drain {
		return
	}
	if count == r.cfg.concurrent {
		r.trace.Refuse(fmt.Sprintf("the concurrent section is at its maximum of %d", r.cfg.concurrent))
	}
	if step.Client > r.cfg.clients {
		r.trace.Refuse(fmt.Sprintf("client %d is outside clients 0 to %d", step.Client, r.cfg.clients))
	}
	index := indexOf(eligible, step.Action)
	if index < 0 {
		r.trace.Refuse(fmt.Sprintf("%q is not among the actions that the step lists", step.Action))
	}
	r.trace.Take(1, uint64(step.Client), uint64(index))
}

// followDrain serves the index of the trace's next entry when it is a drain
// step entry. It refuses the entry for an action that available lacks, which
// is every action once the drain has ended.
func (r *run[S]) followDrain(available []Action[S]) {
	step, ok := r.trace.Next()
	if !ok || !step.Drain {
		return
	}
	index := indexOf(available, step.Action)
	if index < 0 {
		r.trace.Refuse(fmt.Sprintf("%q is not among the drain actions that are enabled", step.Action))
	}
	r.trace.Take(uint64(index))
}

// indexOf returns the index of the action named name in listed, and -1 for
// none.
func indexOf[S any](listed []Action[S], name string) int {
	return slices.IndexFunc(listed, func(a Action[S]) bool { return a.Name == name })
}
