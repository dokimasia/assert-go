// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"context"
	"errors"
	"sync"

	"go.dokimi.dev/assert"
)

// ErrOwn is what a subject answers when it fails on its own terms,
// unrelated to any handle it was given.
var ErrOwn = errors.New("conformance: the subject failed for its own reason")

// Subject is one built behaviour, in every shape an assertion asks for.
//
// A corpus case names a behaviour rather than stating a callable, and
// the assertions taking one differ in shape: one takes a context, one
// takes nothing, one takes a seat. Holding all of them means the
// per-assertion drivers do not each rebuild the behaviour.
type Subject struct {
	// Ctx is the shape honours-cancellation, honours-deadline and
	// nil-context-safe take.
	Ctx func(ctx context.Context) error
	// Bare is the shape panics, does-not-panic and pure take.
	Bare func()
	// Seated is the shape eventually takes.
	Seated func(tb assert.TB)
	// Observe reads the state pure compares across the call.
	Observe func() []int
}

// Subjects builds each named behaviour. A kind absent here is one this
// language cannot make, and the case is skipped.
var Subjects = map[string]func() *Subject{
	"returns-ok":          returnsOK,
	"reads-handle":        readsHandle,
	"ignores-handle":      returnsOK,
	"raises":              raises,
	"fails-otherwise":     failsOtherwise,
	"dereferences-handle": dereferencesHandle,
	"never-settles":       neverSettles,
	"settles-after":       settlesAfter,
	"accumulates":         func() *Subject { return observed(true) },
	"leaves-state-alone":  func() *Subject { return observed(false) },
}

// returnsOK does the work and answers success, whatever it was handed.
func returnsOK() *Subject {
	return &Subject{
		Ctx:  func(context.Context) error { return nil },
		Bare: func() {},
	}
}

// readsHandle answers the reason the handle gives, and success when it
// is still running.
func readsHandle() *Subject {
	return &Subject{
		Ctx: func(ctx context.Context) error { return ctx.Err() },
	}
}

// raises panics rather than answering.
func raises() *Subject {
	return &Subject{
		Bare: func() { panic("the subject raised") },
		Ctx:  func(context.Context) error { panic("the subject raised") },
	}
}

// failsOtherwise answers a failure of its own, which is not the reason
// a handle would give.
func failsOtherwise() *Subject {
	return &Subject{
		Ctx: func(context.Context) error { return ErrOwn },
	}
}

// dereferencesHandle reads a handle without checking it is there.
func dereferencesHandle() *Subject {
	return &Subject{
		Ctx: func(ctx context.Context) error {
			// A nil context panics here, which is the behaviour under
			// test: the assertion asks whether a subject survives one.
			return ctx.Err()
		},
	}
}

// neverSettles reports a failure on every attempt.
func neverSettles() *Subject {
	return &Subject{
		Seated: func(tb assert.TB) { tb.Errorf("never settles") },
	}
}

// settlesAfter reports a failure twice and succeeds on the third
// attempt. The count is per subject, so two cases cannot see each
// other's attempts.
func settlesAfter() *Subject {
	var mu sync.Mutex
	attempts := 0
	return &Subject{
		Seated: func(tb assert.TB) {
			mu.Lock()
			attempts++
			at := attempts
			mu.Unlock()
			if at < 3 {
				tb.Errorf("not yet")
			}
		},
	}
}

// observed answers a subject with state something outside it can read,
// which either changes on a call or does not.
func observed(changes bool) *Subject {
	held := []int{1, 2}
	return &Subject{
		Bare: func() {
			if changes {
				held = append(held, len(held))
			}
		},
		// A copy, so the projection does not share memory with the
		// subject and read the same value twice.
		Observe: func() []int { return append([]int(nil), held...) },
	}
}
