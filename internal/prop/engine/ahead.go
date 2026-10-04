// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"sync"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/tree"
	"go.dokimi.dev/assert/internal/record"
)

// flight is one case that a worker of a run on more than one worker runs.
type flight struct {
	// c is the case.
	c *Case
	// trail is the provider of a random case, and nil for any other case.
	trail *trailing
	// done is closed once the case has ended.
	done chan struct{}
	// e is how the case ended, set before done is closed.
	e Execution
}

// newFlight returns a case whose values come from p, not yet started. The
// case keeps its walk, which the runner enters into the case tree once the
// case has ended. Each call record that the case keeps states the steps
// that its walk had made, so the runner keeps only the calls that a run on
// one worker makes.
func newFlight(p provider, s Settings) *flight {
	c := newCase(p, s, nil)
	c.keepsWalk = true
	if s.Slot != nil {
		record.Run(&c.calls, s.Slot, c.steps)
	}
	return &flight{c: c, done: make(chan struct{})}
}

// ahead is the executor of more than one worker.
//
// Its workers run the random and edge cases ahead of the runner, outside
// the case tree, in the order one worker runs them: random case 0, edge
// case 0, random case 1, edge case 1, and on, with random cases alone after
// the fourth edge case. A worker starts a case at most Workers positions of
// that order past the furthest case that the runner called for. A call for
// a case waits until the case has ended, and enters it into the tree then.
// A replayed case, the simplest case or a prefix case, runs on the next
// free worker before any case ahead.
//
// At close, every case that the runner did not take is cancelled, and ends
// at its next choice.
type ahead struct {
	// body is the property's body.
	body Body
	// s are the settings of the run.
	s Settings
	// tree is the case tree, which only the runner's goroutine touches.
	tree *tree.Tree
	// mu guards the fields below.
	mu sync.Mutex
	// wake wakes the workers when a case to run arrives or the run closes.
	wake *sync.Cond
	// flights are the cases ahead that a worker started or the runner called
	// for, by their position, until the runner takes them.
	flights map[int]*flight
	// next is the position of the next case ahead to start.
	next int
	// taken is one past the furthest position the runner called for.
	taken int
	// replays are the replayed cases that wait for a worker, in order.
	replays []*flight
	// closed reports whether the run has ended.
	closed bool
	// workers waits for every worker to return.
	workers sync.WaitGroup
}

// newAhead returns the executor of s.Workers workers, which start at once.
func newAhead(body Body, s Settings, t *tree.Tree) *ahead {
	a := &ahead{body: body, s: s, tree: t, flights: make(map[int]*flight)}
	a.wake = sync.NewCond(&a.mu)
	for range s.Workers {
		a.workers.Go(a.work)
	}
	return a
}

// replay runs the case that replays choices on the next free worker, and
// enters it into the tree once it has ended.
func (a *ahead) replay(choices []choice.Choice) Execution {
	f := newFlight(replaying{choices: choices}, a.s)
	a.mu.Lock()
	a.replays = append(a.replays, f)
	a.wake.Signal()
	a.mu.Unlock()
	<-f.done
	e, _ := entered(a.tree, f.e)
	return e
}

// random takes random case index and enters it into the tree. For a case
// that the tree ends as repeated, it returns the record and the source of
// the case at the step of the repeat, where a run on one worker stops it.
func (a *ahead) random(index uint64) (Execution, []choice.Choice, random.Source) {
	f := a.take(randomPosition(index))
	e, steps := entered(a.tree, f.e)
	if e.Status == CaseRepeated {
		return e, f.c.recordAt(steps), f.trail.after(steps)
	}
	return e, f.c.record(), f.trail.after(len(f.trail.states) - 1)
}

// edge takes the edge case of the boundary at and enters it into the tree.
func (a *ahead) edge(at boundary) Execution {
	e, _ := entered(a.tree, a.take(edgePosition(at)).e)
	return e
}

// close cancels every case that the runner did not take, and returns once
// every worker has returned.
func (a *ahead) close() {
	a.mu.Lock()
	a.closed = true
	for _, f := range a.flights {
		f.c.cancel()
	}
	a.wake.Broadcast()
	a.mu.Unlock()
	a.workers.Wait()
}

// take returns the case at position once it has ended, and lets the
// workers run up to Workers positions past it.
func (a *ahead) take(position int) *flight {
	a.mu.Lock()
	f := a.flightAt(position)
	a.taken = max(a.taken, position+1)
	a.wake.Broadcast()
	a.mu.Unlock()
	<-f.done
	a.mu.Lock()
	delete(a.flights, position)
	a.mu.Unlock()
	return f
}

// work runs cases until the run closes.
func (a *ahead) work() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for f := a.claim(); f != nil; f = a.claim() {
		a.mu.Unlock()
		f.e = finish(f.c, a.body)
		close(f.done)
		a.mu.Lock()
	}
}

// claim returns the next case for a worker. The caller has locked a.mu.
// claim returns a replayed case first, and otherwise the next case ahead
// while it is at most Workers positions past the runner. It waits while
// there is none, and returns nil once the run has closed.
func (a *ahead) claim() *flight {
	for !a.closed {
		if len(a.replays) > 0 {
			f := a.replays[0]
			a.replays = a.replays[1:]
			return f
		}
		if a.next < a.taken+a.s.Workers {
			f := a.flightAt(a.next)
			a.next++
			return f
		}
		a.wake.Wait()
	}
	return nil
}

// flightAt returns the case at position, and makes it when neither the
// runner nor a worker has made it yet. The caller has locked a.mu.
func (a *ahead) flightAt(position int) *flight {
	if f, ok := a.flights[position]; ok {
		return f
	}
	var f *flight
	if index, ok := randomAt(position); ok {
		trail := newTrailing(random.ForCase(a.s.Seed, index))
		f = newFlight(trail, a.s)
		f.trail = trail
	} else {
		f = newFlight(edge{at: boundaries[position/2]}, a.s)
	}
	a.flights[position] = f
	return f
}

// randomPosition returns the position of random case index in the order
// one worker runs the cases ahead: 2*index while edge cases alternate with
// the random cases, and index + len(boundaries) after the last edge case.
// The smaller of the two is the position at every index.
func randomPosition(index uint64) int {
	return int(min(2*index, index+uint64(len(boundaries))))
}

// edgePosition returns the position of the edge case of the boundary at,
// whose value is its index in boundaries.
func edgePosition(at boundary) int {
	return 2*int(at) + 1
}

// randomAt returns the index of the random case at position, and false for
// the position of an edge case, an odd position below 2*len(boundaries).
// The index is position/2 while edge cases alternate with the random
// cases, and position - len(boundaries) after the last edge case. The
// larger of the two is the index at every position.
func randomAt(position int) (uint64, bool) {
	if position%2 == 1 && position/2 < len(boundaries) {
		return 0, false
	}
	return uint64(max(position/2, position-len(boundaries))), true
}
