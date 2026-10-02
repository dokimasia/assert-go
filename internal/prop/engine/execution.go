// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"errors"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/tree"
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
)

// Valid reports whether s is one of the five statuses.
func (s Status) Valid() bool {
	return s <= CaseDiverged
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
}

// Body is a property's body: it draws from the case it receives, and
// reports a failure to it or returns.
type Body func(*Case)

// Generate calls body once on case index of a run with seed, outside the
// case tree, with the cap of [MaxChoices].
func Generate(body Body, seed, index uint64, clock assert.Clock) Execution {
	return execute(body, newGenerating(random.ForCase(seed, index)), MaxChoices, clock)
}

// Replay calls body once on a case that replays choices, outside the case
// tree, with the cap of [MaxChoices].
func Replay(body Body, choices []choice.Choice, clock assert.Clock) Execution {
	return execute(body, replaying{choices: choices}, MaxChoices, clock)
}

// Bridge calls body once on a case decoded from a fuzzer's bytes, outside
// the case tree, with the cap of [MaxChoices].
func Bridge(body Body, data []byte, clock assert.Clock) Execution {
	return execute(body, &bridging{data: data}, MaxChoices, clock)
}

// execute calls body once, on a goroutine of its own, on a new case outside
// the case tree whose values come from p, and returns how the case ended.
func execute(body Body, p provider, maxChoices int, clock assert.Clock) Execution {
	return finish(newCase(p, maxChoices, nil, clock), body)
}

// finish calls body once on c, on a goroutine of its own, and returns how
// the case ended.
func finish(c *Case, body Body) Execution {
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
// ran.
func entered(t *tree.Tree, e Execution) (Execution, int) {
	c := e.Case
	c.mu.Lock()
	c.walker = t.Walk()
	steps := len(c.walk)
	for index, s := range c.walk {
		if stop := c.walked(c.walker.Step(s.bounds, s.value)); stop != running {
			c.stop, steps = stop, index+1
			break
		}
	}
	c.mu.Unlock()
	return executionOf(c), steps
}

// executionOf returns how the case ended, once its body's goroutine ended.
// A case that ended in a repeat, a divergence or past its cap leaves no
// leaf in the tree. Every other case ends its walk with a leaf.
func executionOf(c *Case) Execution {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stop == repeated {
		return Execution{Case: c, Status: CaseRepeated}
	}
	if c.stop == diverged {
		return Execution{Case: c, Status: CaseDiverged, Divergence: requestDivergence(c.divergence)}
	}
	if c.stop == overrun {
		return Execution{Case: c, Status: CaseRejected}
	}
	e := Execution{Case: c, Status: CasePassed}
	if records := c.recorder.Failures(); len(records) > 0 {
		e.Status, e.Identity = CaseFailed, identityOf(records[0])
	} else if c.panicked != nil {
		e.Status, e.Identity, e.Panic, e.Stack = CaseFailed, *c.panicked, c.panicValue, c.panicStack
	} else if c.stop == rejected {
		e.Status = CaseRejected
	}
	if c.walker == nil {
		return e
	}
	if d, ok := errors.AsType[*tree.DivergenceError](c.walker.End()); ok {
		return Execution{Case: c, Status: CaseDiverged, Divergence: requestDivergence(d)}
	}
	return e
}
