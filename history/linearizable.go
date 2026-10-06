// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"errors"
	"reflect"
	"runtime"
	"sync"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
)

// linearizableOp is the operation of Linearizable, which names its faults.
const linearizableOp = "history.Linearizable"

// callerFrames is the most frames that the call site of a failing check is
// searched in.
const callerFrames = 64

// ErrSpec is the kind of the fault of a check whose spec's function panics,
// or ends the goroutine that runs it.
var ErrSpec = errors.New("history: a function of a spec panics or ends its goroutine")

// hashlessNote is the note of a check whose spec states Equal and no Hash.
const hashlessNote = linearizableOp + ": the spec states Equal and no Hash, so the search hashes every " +
	"state alike and compares each state with every state of the same calls. A Hash that agrees with Equal " +
	"lets the search compare the states of one hash alone."

// Linearizable checks that the calls of each partition of the history h have
// a linearization that the spec s accepts: an order that respects their
// real-time order and that s accepts as a sequence. It fails tb with one
// record of the assertion linearizable when the check does not pass. The
// record's contract is contract, and its location is the call of
// Linearizable.
//
// The check removes the calls that failed. A call that completed as [OK] is
// known, and takes effect between its invocation and its completion. A call
// whose outcome is unknown, and a pending call, takes effect at some point
// after its invocation, or never. Call a precedes call b in real time when
// a completed before b was invoked. Two calls that share a key are in one
// partition, and a call without keys puts every call into one partition.
// The check searches the partitions in the order of their first invocation,
// each from the spec's initial state, with the search that the definition
// fixes. It reports the first violated partition, and otherwise the first
// undecided one. The options state its limits and its workers, and [Whole]
// and [Final] search every call as one partition and store the states that
// a linearization leaves. [Resume] continues the search of an earlier check
// of the same history.
//
// The record's detail states the ten fields of the definition:
//
//   - outcome, a [Verdict]: [Violated] or [Undecided].
//   - partitions, steps, calls and concurrency, as int.
//   - partition, a []any of the keys of the reported partition, and empty
//     for a partition of every key.
//   - linearized and candidates, a []Span each, and states, a []S: the
//     frontier's order of calls, its states, and the calls that the spec
//     rejected there.
//   - limit, a [Limit] for an undecided check, and nil for a violated one.
//
// An [assert.Reporter] seat receives the record. Any other seat receives the
// record's sentence through Fatalf. The call record of a recorded run states
// the detail in the history's JSON form.
//
// A spec that states Equal and no Hash makes the search compare the states
// of every configuration of one set of calls, which is slower by orders of
// magnitude on a long history. The first check of a history against such a
// spec writes a note that states the cause into the log of a seat that has
// a log, whatever its verdict.
//
// # Errors
//
// The check ends the call with a fault for a nil history, for a spec without
// Initial or without Next, for [Final] and [Resume] of states of another
// type than the spec's, and for Resume without [Whole]. It ends the call
// with a fault of the kind [ErrSpec] when a function of the spec panics or
// ends the goroutine, and the fault names the call that the search stepped.
// A panic in a partition that one worker would not search ends nothing.
//
// # Allocation contract
//
// A check allocates its calls, its partitions and the memo of each search,
// and the states that the spec returns. A passing check of a register over
// two sequential calls allocates 38 times. A check that continues the search
// of the check before it allocates for the calls since that check alone: a
// write of the register that a history records and the check after it
// allocate 10 times together.
func Linearizable[S any](tb assert.TB, h *History, s Spec[S], contract string, opts ...Option) {
	tb.Helper()
	run := matcher.Begin(tb)
	c := configure(opts)
	if h == nil {
		run.Fault(matcher.Fatal, linearizableID, contract, fault.In(linearizableOp, fault.New("the history is nil")))
		return
	}
	if s.Initial == nil || s.Next == nil {
		run.Fault(matcher.Fatal, linearizableID, contract,
			fault.In(linearizableOp, fault.New("the spec states no Initial or no Next")))
		return
	}
	final, typed := c.final.(*[]S)
	if c.final != nil && !typed {
		run.Fault(matcher.Fatal, linearizableID, contract, fault.In(linearizableOp,
			fault.New("Final states %T for a spec whose states are of type %v", c.final, reflect.TypeFor[S]())))
		return
	}
	if _, resumable := c.resume.(*Checkpoint[S]); c.resume != nil && !resumable {
		run.Fault(matcher.Fatal, linearizableID, contract, fault.In(linearizableOp,
			fault.New("Resume states %T for a spec whose states are of type %v", c.resume, reflect.TypeFor[S]())))
		return
	}
	if c.resume != nil && !c.whole {
		run.Fault(matcher.Fatal, linearizableID, contract, fault.In(linearizableOp,
			fault.New("Resume continues the search of one partition, and the check states no Whole")))
		return
	}
	if s.Equal != nil && s.Hash == nil && h.noteHashless() {
		matcher.Note(tb, hashlessNote)
	}
	d, err := check(h, s.operations(), c)
	if err != nil {
		run.Fault(matcher.Fatal, linearizableID, contract, err)
		return
	}
	if typed {
		*final = d.final
	}
	if d.outcome == Passed {
		run.Pass(matcher.Fatal, linearizableID, contract)
		return
	}
	var pcs [callerFrames]uintptr
	where := matcher.CallerWhere(pcs[:runtime.Callers(1, pcs[:])])
	run.FailRun(matcher.Fatal, assert.Failure{
		Assertion: linearizableID, Contract: contract, Detail: d.fields(), Where: where,
	}, d)
}

// noteHashless reports whether no check of h has noted a spec that states
// Equal and no Hash, and records that one has.
func (h *History) noteHashless() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	first := !h.hashless
	h.hashless = true
	return first
}

// check returns the detail of the check of h through the spec's functions
// ops under c: the first violated partition, else the first undecided one,
// else a pass. A pass states its final states when c asks for them and the
// check searched one partition or none. It returns the fault of a spec's
// function that panicked in a partition that one worker would search.
func check[S any](h *History, ops operations[S], c config) (detail[S], error) {
	d := &deadline{}
	if c.timeLimit > 0 {
		d.end = time.Now().Add(c.timeLimit)
	}
	if cp, resumable := c.resume.(*Checkpoint[S]); resumable {
		return resumed(h, ops, c, d, cp)
	}
	events, ids := h.recordedFrom(0)
	calls, _ := callsOf(events, ids, 0)
	parts := partitionsOf(calls)
	if c.whole && len(calls) > 0 {
		parts = []partition{{keys: []any{}, calls: calls}}
	}
	if c.final != nil && len(parts) == 0 {
		var e ending[S]
		newSearch(ops, partition{}, c, d).run(func(end ending[S]) { e = end })
		return detail[S]{outcome: Passed, final: e.states}, e.err
	}
	endingOf, stop := searches(ops, parts, c, d)
	defer stop()
	steps := 0
	var undecided *detail[S]
	var final []S
	for i, p := range parts {
		e := endingOf(i)
		if e.err != nil {
			return detail[S]{}, e.err
		}
		steps += e.steps
		if e.verdict == Passed {
			if len(parts) == 1 {
				final = e.states
			}
			continue
		}
		r := reported(p, e, len(parts), steps)
		if e.verdict == Violated {
			return r, nil
		}
		if undecided == nil {
			undecided = &r
		}
	}
	if undecided != nil {
		return *undecided, nil
	}
	return detail[S]{outcome: Passed, partitions: len(parts), steps: steps, final: final}, nil
}

// searches starts the searches of parts on c.workers goroutines, in
// partition order, and returns endingOf, which returns how the search of the
// partition i ended once it has, and stop, which cancels the searches still
// running, takes the partitions that no worker started, and waits for the
// workers. On one worker, endingOf runs the search on the caller's
// goroutine.
func searches[S any](ops operations[S], parts []partition, c config, d *deadline) (func(int) ending[S], func()) {
	if c.workers == 1 {
		return func(i int) ending[S] {
			var e ending[S]
			newSearch(ops, parts[i], c, d).run(func(end ending[S]) { e = end })
			return e
		}, func() {}
	}
	type finished struct {
		// i is the index of the partition.
		i int
		// e is how its search ended.
		e ending[S]
	}
	work := make(chan int, len(parts))
	for i := range parts {
		work <- i
	}
	close(work)
	done := make(chan finished, len(parts))
	var workers sync.WaitGroup
	for range min(c.workers, len(parts)) {
		workers.Go(func() {
			for i := range work {
				newSearch(ops, parts[i], c, d).run(func(e ending[S]) { done <- finished{i: i, e: e} })
			}
		})
	}
	endings := make([]ending[S], len(parts))
	ended := make([]bool, len(parts))
	endingOf := func(i int) ending[S] {
		for !ended[i] {
			f := <-done
			endings[f.i], ended[f.i] = f.e, true
		}
		return endings[i]
	}
	stop := func() {
		d.cancelled.Store(true)
		for range work {
		}
		workers.Wait()
	}
	return endingOf, stop
}
