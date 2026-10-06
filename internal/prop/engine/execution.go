// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"cmp"
	"context"
	"errors"
	"math"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/tree"
	"go.dokimi.dev/assert/internal/record"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Status -linecomment -output=execution.string_gen.go

// Status is how one case ended.
type Status uint8

const (
	// CasePassed is a case whose body returned without a failure.
	CasePassed Status = 0 // passed
	// CaseFailed is a case whose body reported a failure or panicked.
	CaseFailed Status = 1 // failed
	// CaseRejected is a case that the body rejected, that a filter or a
	// unique collection gave up on, or that went past its cap on choices.
	CaseRejected Status = 2 // rejected
	// CaseRepeated is a case whose choices repeat a tested case.
	CaseRepeated Status = 3 // repeated
	// CaseDiverged is a case whose body made other requests after the same
	// values than an earlier case made.
	CaseDiverged Status = 4 // diverged
	// CaseRefused is a case of [Settings.Draws] that refused the entry of a
	// draw.
	CaseRefused Status = 5 // refused
)

// Valid reports whether s is one of the six statuses.
func (s Status) Valid() bool {
	return s <= CaseRefused
}

// Execution is one call of a body: its case and how it ended.
type Execution struct {
	// Case is the record of the call.
	Case *Case
	// Status is how the case ended.
	Status Status
	// Identity is the identity of a failed case's failure.
	Identity Identity
	// Divergence is the difference that made a case diverge.
	Divergence *Divergence
	// Panic is the value that a failed case's body panicked with, and nil
	// for a case that failed through a record or did not fail.
	Panic any
	// Stack is the stack of the body's goroutine where it panicked, as
	// runtime/debug.Stack formats it, and nil when Panic is nil.
	Stack []byte
	// Refusal is the refusal of a refused case, a fault at the entry of
	// Settings.Draws that the case refused, and nil for any other.
	Refusal error
	// ran are the cases of the runs of a case that [Case.Repeat] ran more
	// than once, in the order they ran, and nil for a case that ran once.
	ran []*Case
}

// runs returns the number of runs of the body that e made.
func (e Execution) runs() int {
	return max(len(e.ran), 1)
}

// take hands the calls of every run of e to slot under phase, in the order
// the runs ran. Each run takes a number of its own.
func (e Execution) take(slot *record.Slot, phase record.Phase) {
	if e.ran == nil {
		slot.Take(&e.Case.calls, phase)
		return
	}
	for _, c := range e.ran {
		slot.Take(&c.calls, phase)
	}
}

// Body is a property's body: it draws from the case it receives, and
// reports a failure to it or returns.
type Body func(*Case)

// WithContext returns a body that runs body on a case whose context,
// which [Case.Context] returns, derives from ctx. Pass the context of the
// test that runs the property, such as the one a *testing.T returns.
func WithContext(ctx context.Context, body Body) Body {
	return func(c *Case) {
		c.derive(ctx)
		body(c)
	}
}

// Generate calls body once on case index of a run with seed, outside the
// case tree, with the cap of [MaxChoices].
func Generate(body Body, seed, index uint64, clock assert.Clock) Execution {
	return execute(body, newGenerating(random.ForCase(seed, index)), Settings{MaxChoices: MaxChoices, Clock: clock})
}

// Replay calls body once on a case that replays choices, outside the case
// tree, with the cap of [MaxChoices].
func Replay(body Body, choices []choice.Choice, clock assert.Clock) Execution {
	return execute(body, replaying{choices: choices}, Settings{MaxChoices: MaxChoices, Clock: clock})
}

// Bridge calls body once on a case decoded from a fuzzer's bytes, outside
// the case tree, with the cap of s.MaxChoices, or of [MaxChoices] when s
// states none, and reads the clock of s. The slot of s takes the calls of
// the case under the phase fuzz.
func Bridge(body Body, data []byte, s Settings) Execution {
	limits := Settings{MaxChoices: cmp.Or(s.MaxChoices, MaxChoices), Clock: s.Clock, Slot: s.Slot}
	e := execute(body, &bridging{data: data}, limits)
	e.take(s.Slot, record.Fuzz)
	return e
}

// execute calls body once, on a goroutine of its own, on a new case outside
// the case tree whose values come from p, capped at s.MaxChoices, and
// returns how the case ended.
func execute(body Body, p provider, s Settings) Execution {
	return finish(newCase(p, s, nil), body)
}

// finish calls body on c, on a goroutine of its own, and returns how the
// case ended, with every run that [Case.Repeat] asks for, as repeat runs
// them.
func finish(c *Case, body Body) Execution {
	return repeat(runOnce(c, body), body, math.MaxInt)
}

// repeat runs the case of e, which ran once, again on its choices when it
// passed and asked for more runs with [Case.Repeat]. Each run is a case
// outside the case tree that keeps its walk and its wheres when the first
// does. The runs go on until one fails, the runs the case asked for have
// passed, or limit runs in all have run. The first run that fails is the
// case's end, with its record. The execution keeps the case of every run,
// whose calls [Execution.take] hands on.
func repeat(e Execution, body Body, limit int) Execution {
	c := e.Case
	runs := min(c.repeats(), limit)
	if e.Status != CasePassed || runs == 1 {
		return e
	}
	ran := make([]*Case, 1, runs)
	ran[0] = c
	for n := 1; n < runs; n++ {
		again := newCase(replaying{choices: c.Choices()}, c.settings, nil)
		again.keepsWheres = c.keepsWheres
		if c.keepsWalk {
			again.keepWalk()
		}
		r := runOnce(again, body)
		ran = append(ran, again)
		if r.Status == CaseFailed {
			r.ran = ran
			return r
		}
	}
	e.ran = ran
	return e
}

// runOnce calls body once on c, on a goroutine of its own, and returns how
// the case ended.
func runOnce(c *Case, body Body) Execution {
	c.done.Add(1)
	go c.run(body)
	c.done.Wait()
	return executionOf(c)
}

// entered walks the case of e, which ran outside the case tree and kept its
// walk, into t, and returns how it ends there with the number of steps it
// made by then.
//
// The walk is every choice the case made, the ones a rewind removed from
// its record included, as a case that walked t as it ran stepped them. It
// stops where such a case would have stopped: at a choice that repeats a
// tested case, or at a request that differs from the recorded one.
// [executionOf] then ends the case as it ends a case that walked t as it
// ran. The case keeps the calls made before the step that stops it, which
// are the calls of such a case, and keeps no other run that [Case.Repeat]
// asked for. A case that the walk does not stop keeps its runs.
func entered(t *tree.Tree, e Execution) (Execution, int) {
	c := e.Case
	c.mu.Lock()
	c.walker = t.Walk()
	steps := len(c.walk)
	stopped := false
	for index, s := range c.walk {
		if stop := c.walked(c.walker.Step(s.bounds, s.value)); stop != running {
			c.stop, steps, stopped = stop, index+1, true
			break
		}
	}
	c.mu.Unlock()
	out := executionOf(c)
	if stopped {
		record.Cut(&c.calls, steps)
		return out, steps
	}
	out.ran = e.ran
	return out, steps
}

// executionOf returns how the case ended, once its body's goroutine ended.
// A failure or a panic of the body or of a cleanup fails a case that the
// body rejected or that went past its cap, as it fails any other. A case
// that ended in a repeat, a divergence or past its cap leaves no leaf in the
// tree. Every other case ends its walk with a leaf.
func executionOf(c *Case) Execution {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stop == refused {
		return Execution{Case: c, Status: CaseRefused, Refusal: c.refusal}
	}
	if c.stop == repeated {
		return Execution{Case: c, Status: CaseRepeated}
	}
	if c.stop == diverged {
		return Execution{Case: c, Status: CaseDiverged, Divergence: requestDivergence(c, c.divergence)}
	}
	e := Execution{Case: c, Status: CasePassed}
	if records := c.recorder.Failures(); len(records) > 0 {
		e.Status, e.Identity = CaseFailed, identityOf(records[0])
	} else if c.panicked != nil {
		e.Status, e.Identity, e.Panic, e.Stack = CaseFailed, *c.panicked, c.panicValue, c.panicStack
	} else if c.stop == rejected || c.stop == overrun {
		e.Status = CaseRejected
	}
	if c.walker == nil || c.stop == overrun {
		return e
	}
	if d, ok := errors.AsType[*tree.DivergenceError](c.walker.End()); ok {
		return Execution{Case: c, Status: CaseDiverged, Divergence: requestDivergence(c, d)}
	}
	return e
}
