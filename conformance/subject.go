// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"

	"go.dokimi.dev/assert"
)

// ErrOwn is what a subject returns when it fails on its own terms,
// unrelated to any handle it was given.
var ErrOwn = errors.New("conformance: the subject failed for its own reason")

// ErrClosed is what a closed subject of after-close fails with.
var ErrClosed = errors.New("conformance: the subject is closed")

// Subject is one built behaviour, in every shape that an assertion takes.
//
// A corpus case names a behaviour instead of stating a callable, and the
// assertions that take one differ in shape: one takes a context, one takes
// nothing, one takes a seat. One Subject with every shape means that the
// per-assertion drivers do not each rebuild the behaviour.
type Subject struct {
	// Ctx is the shape honours-cancellation, honours-deadline and
	// nil-context-safe take.
	Ctx func(ctx context.Context) error
	// Bare is the shape panics, does-not-panic, pure and not-pure take.
	Bare func()
	// Seated is the shape eventually takes.
	Seated func(tb assert.TB)
	// Observe reads the integer that pure, not-pure, idempotent,
	// accumulates and monotonic compare.
	Observe func() int
	// Call is the operation that idempotent and accumulates call with
	// Input, and total calls with each element of Domain.
	Call func(int) error
	// Input is what idempotent, accumulates, deterministic and round-trip
	// hand their subject.
	Input int
	// Compute is the computation that deterministic calls with Input.
	Compute func(int) (int, error)
	// Combine is the operation that commutative applies to A and B, and
	// associative to A, B and C.
	Combine func(a, b int) int
	// A, B and C are the operands of Combine.
	A, B, C int
	// Forward renders Input as text, and Inverse parses the text back, for
	// round-trip.
	Forward func(int) (string, error)
	// Inverse parses what Forward renders.
	Inverse func(string) (int, error)
	// Iterate yields the sequence that stable-order and no-duplicates read.
	Iterate func() ([]int, error)
	// Advance moves the integer that monotonic reads through Observe, Steps
	// times.
	Advance func() error
	// Steps is how many times monotonic calls Advance.
	Steps int
	// Domain is the inputs that total calls Call with.
	Domain []int
	// Closer closes the subject of after-close, and Use calls it after.
	Closer func() error
	// Use is the call that after-close makes once Closer has closed the
	// subject.
	Use func() error
	// Sentinel is the failure that Use returns once the subject is closed.
	Sentinel error
	// Induce induces the failure that poisoned then reads through Read.
	Induce func()
	// Read is one reading of the subject of poisoned.
	Read func() error
}

// Subjects builds each named behaviour. A kind absent here is one this
// language cannot make, and the corpus runner fails its cases.
var Subjects = map[string]func() *Subject{
	"returns-ok":          returnsOK,
	"reads-handle":        readsHandle,
	"ignores-handle":      returnsOK,
	"raises":              raises,
	"fails-otherwise":     failsOtherwise,
	"dereferences-handle": dereferencesHandle,
	"never-settles":       neverSettles,
	"settles-after":       settlesAfter,
	"accumulates":         func() *Subject { return counter(1) },
	"leaves-state-alone":  func() *Subject { return counter(0) },
	"sets-value":          setsValue,
	"counts-calls":        countsCalls,
	"adds":                func() *Subject { return combines(func(a, b int) int { return a + b }) },
	"subtracts":           func() *Subject { return combines(func(a, b int) int { return a - b }) },
	"renders-decimal":     func() *Subject { return renders(false) },
	"drops-the-sign":      func() *Subject { return renders(true) },
	"yields-in-order":     func() *Subject { return yields(1, 2, 3, 4, 5) },
	"rotates":             rotates,
	"repeats-an-element":  func() *Subject { return yields(1, 2, 2, 3) },
	"wraps-around":        wrapsAround,
	"refuses-after-close": func() *Subject { return closes(true) },
	"serves-after-close":  func() *Subject { return closes(false) },
}

// returnsOK returns success, whatever it was handed. Its computation
// returns its input, 1, and its domain is 1, 2 and 3.
func returnsOK() *Subject {
	return &Subject{
		Ctx:     func(context.Context) error { return nil },
		Bare:    func() {},
		Call:    func(int) error { return nil },
		Input:   1,
		Compute: func(input int) (int, error) { return input, nil },
		Domain:  []int{1, 2, 3},
	}
}

// readsHandle returns the reason the handle gives, and success when it
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

// failsOtherwise returns a failure of its own, which is not the reason a
// handle would give, for every call. Its domain is 1, 2 and 3.
func failsOtherwise() *Subject {
	return &Subject{
		Ctx:    func(context.Context) error { return ErrOwn },
		Call:   func(int) error { return ErrOwn },
		Domain: []int{1, 2, 3},
	}
}

// dereferencesHandle reads a handle without checking it is there.
func dereferencesHandle() *Subject {
	return &Subject{
		Ctx: func(ctx context.Context) error {
			// A nil context panics here, which is the behaviour under
			// test: the assertion checks whether a subject handed one panics.
			return ctx.Err()
		},
	}
}

// neverSettles reports a failure on every attempt and every reading.
// Inducing changes nothing.
func neverSettles() *Subject {
	return &Subject{
		Seated: func(tb assert.TB) { tb.Errorf("never settles") },
		Induce: func() {},
		Read:   func() error { return ErrOwn },
	}
}

// settlesAfter reports a failure twice and succeeds on the third attempt
// or reading. Inducing changes nothing. The count is per subject, so two
// cases cannot see each other's attempts.
func settlesAfter() *Subject {
	var mu sync.Mutex
	attempts := 0
	settled := func() bool {
		mu.Lock()
		defer mu.Unlock()
		attempts++
		return attempts >= 3
	}
	return &Subject{
		Seated: func(tb assert.TB) {
			if !settled() {
				tb.Errorf("not yet")
			}
		},
		Induce: func() {},
		Read: func() error {
			if !settled() {
				return ErrOwn
			}
			return nil
		},
	}
}

// counter returns a subject whose state is an integer that starts at 0 and
// rises by step on each call and each advance. It advances 5 steps.
func counter(step int) *Subject {
	count := 0
	rise := func() { count += step }
	return &Subject{
		Bare: rise,
		Call: func(int) error {
			rise()
			return nil
		},
		Advance: func() error {
			rise()
			return nil
		},
		Steps:   5,
		Observe: func() int { return count },
	}
}

// setsValue returns a cell that starts at 0, which a call sets to its
// input, 7.
func setsValue() *Subject {
	cell := 0
	return &Subject{
		Call: func(v int) error {
			cell = v
			return nil
		},
		Input:   7,
		Observe: func() int { return cell },
	}
}

// countsCalls returns how many times it has been called, starting at 1,
// whatever its input, 1.
func countsCalls() *Subject {
	calls := 0
	return &Subject{
		Compute: func(int) (int, error) {
			calls++
			return calls, nil
		},
		Input: 1,
	}
}

// combines returns a subject that combines integers with op, over the
// operands 2, 3 and 5.
func combines(op func(a, b int) int) *Subject {
	return &Subject{Combine: op, A: 2, B: 3, C: 5}
}

// renders returns a subject that renders an integer as decimal text and
// parses the text back, over the input -42. One that drops the sign
// renders the integer's absolute value.
func renders(dropSign bool) *Subject {
	return &Subject{
		Forward: func(v int) (string, error) {
			text := strconv.Itoa(v)
			if dropSign {
				text = strings.TrimPrefix(text, "-")
			}
			return text, nil
		},
		Inverse: strconv.Atoi,
		Input:   -42,
	}
}

// yields returns a subject that yields items on every iteration.
func yields(items ...int) *Subject {
	return &Subject{Iterate: func() ([]int, error) { return slices.Clone(items), nil }}
}

// rotates yields the integers 1 to 5, rotated one place further on each
// iteration.
func rotates() *Subject {
	items := []int{1, 2, 3, 4, 5}
	return &Subject{Iterate: func() ([]int, error) {
		out := items
		items = slices.Concat(items[1:], items[:1])
		return out, nil
	}}
}

// wrapsAround returns a counter that starts at 0, rises by one per advance
// and returns to 0 after 3. It advances 5 steps.
func wrapsAround() *Subject {
	count := 0
	return &Subject{
		Observe: func() int { return count },
		Advance: func() error {
			count = (count + 1) % 4
			return nil
		},
		Steps: 5,
	}
}

// closes returns a subject whose calls succeed until it closes. After it
// closes, a subject that refuses fails every call with ErrClosed, and one
// that does not still succeeds.
func closes(refuses bool) *Subject {
	closed := false
	return &Subject{
		Closer: func() error {
			closed = true
			return nil
		},
		Use: func() error {
			if closed && refuses {
				return ErrClosed
			}
			return nil
		},
		Sentinel: ErrClosed,
	}
}
