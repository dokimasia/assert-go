// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"cmp"
	"context"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"sync"

	"go.dokimi.dev/assert"
)

// ErrOwn is what a subject returns when it fails on its own terms,
// unrelated to any handle it was given.
var ErrOwn = errors.New("conformance: the subject failed for its own reason")

// ErrClosed is what a closed subject of after-close fails with.
var ErrClosed = errors.New("conformance: the subject is closed")

// Subject is one built behaviour of the definition's subjects table, as a
// function of each signature that an assertion or a property form calls.
// The field of a signature that the behaviour does not have is nil.
//
// A subject takes its input as any, so one subject serves an assertion,
// which hands it an int, and a property form, which hands it the generated
// input. An integer that a subject returns is an int64, and its caller
// converts it to the Go type of the integers that it compares.
type Subject struct {
	// Ctx is the call of an input x that takes a cancellation handle, as
	// honours-cancellation, honours-deadline and nil-context-safe call it.
	Ctx func(ctx context.Context, x any) error
	// Raise is the call of an input x that raises or returns, as throws
	// and not-throws call it.
	Raise func(x any)
	// Seated is the function that the assertion eventually calls.
	Seated func(tb assert.TB)
	// Call is the operation of an input x: it changes the integer that
	// Observe reads, or returns a failure of its own. Idempotent,
	// accumulates, total, pure and not-pure call it, and so do the
	// property forms of errors.
	Call func(x any) error
	// Observe reads the integer that Call and Advance change.
	Observe func() int
	// Input is the value that an assertion passes to the subject.
	Input int
	// Function returns a value of the input x, as the property form of a
	// function of its input calls it.
	Function func(x any) any
	// Ordered reports whether two adjacent items are in order, as pairwise
	// calls it.
	Ordered func(first, second any) bool
	// Compute is the computation that deterministic calls.
	Compute func(x any) any
	// Combine is the operation that commutative applies to A and B, and
	// associative to A, B and C.
	Combine func(a, b any) any
	// A, B and C are the operands that an assertion passes to Combine.
	A, B, C int
	// Render renders x as decimal text, which round-trip parses back as
	// decimal text.
	Render func(x any) string
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

// Subjects builds each behaviour of the definition's subjects table, by
// its kind. Each call builds a subject with state of its own. A kind
// absent here is one this language cannot make, and the corpus runner and
// the forms runner fail its cases.
var Subjects = map[string]func() *Subject{
	"returns-ok":          returnsOK,
	"reads-handle":        readsHandle,
	"ignores-handle":      ignoresHandle,
	"raises":              raises,
	"raises-on-negative":  raisesOnNegative,
	"fails-otherwise":     failsOtherwise,
	"fails-on-negative":   failsOnNegative,
	"dereferences-handle": dereferencesHandle,
	"never-settles":       neverSettles,
	"settles-after":       settlesAfter,
	"accumulates":         func() *Subject { return counter(1) },
	"leaves-state-alone":  func() *Subject { return counter(0) },
	"sets-value":          setsValue,
	"counts-calls":        countsCalls,
	"adds":                func() *Subject { return combines(func(a, b int64) int64 { return a + b }) },
	"subtracts":           func() *Subject { return combines(func(a, b int64) int64 { return a - b }) },
	"renders-decimal":     func() *Subject { return renders(false) },
	"drops-the-sign":      func() *Subject { return renders(true) },
	"yields-in-order":     func() *Subject { return yields(1, 2, 3, 4, 5) },
	"rotates":             rotates,
	"repeats-an-element":  func() *Subject { return yields(1, 2, 2, 3) },
	"wraps-around":        wrapsAround,
	"refuses-after-close": func() *Subject { return closes(true) },
	"serves-after-close":  func() *Subject { return closes(false) },
	"identity":            function(func(x any) any { return x }),
	"is-non-negative":     function(func(x any) any { return signedOf(x) >= 0 }),
	"returns-null":        function(func(any) any { return nil }),
	"drops-the-first":     function(dropsTheFirst),
	"prepends-zero":       function(func(x any) any { return slices.Concat([]any{int64(0)}, x.([]any)) }),
	"sorts":               function(sorted),
	"wraps-in-a-and-b":    function(func(x any) any { return "a" + x.(string) + "b" }),
	"ascending":           ascending,
}

// signedOf returns x, a value of a signed integer type, as an int64.
func signedOf(x any) int64 { return reflect.ValueOf(x).Int() }

// returnsOK returns success, whatever it was handed. Its computation
// returns its input, 1, and its domain is 1, 2 and 3.
func returnsOK() *Subject {
	return &Subject{
		Ctx:     func(context.Context, any) error { return nil },
		Raise:   func(any) {},
		Call:    func(any) error { return nil },
		Input:   1,
		Compute: func(x any) any { return x },
		Domain:  []int{1, 2, 3},
	}
}

// readsHandle returns the reason that the handle gives, and success for an
// absent handle or one still running.
func readsHandle() *Subject {
	return &Subject{Ctx: func(ctx context.Context, _ any) error {
		if ctx == nil {
			return nil
		}
		return ctx.Err()
	}}
}

// ignoresHandle returns success without reading the handle.
func ignoresHandle() *Subject {
	return &Subject{Ctx: func(context.Context, any) error { return nil }}
}

// raises panics on every call.
func raises() *Subject {
	return &Subject{Raise: func(any) { panic("the subject raised") }}
}

// raisesOnNegative panics for a negative integer input, and returns
// otherwise.
func raisesOnNegative() *Subject {
	return &Subject{Raise: func(x any) {
		if signedOf(x) < 0 {
			panic("the subject raised")
		}
	}}
}

// failsOtherwise returns a failure of its own, which is not the reason a
// handle would give, for every call. Its domain is 1, 2 and 3.
func failsOtherwise() *Subject {
	return &Subject{
		Ctx:    func(context.Context, any) error { return ErrOwn },
		Call:   func(any) error { return ErrOwn },
		Domain: []int{1, 2, 3},
	}
}

// failsOnNegative returns a failure of its own for a negative integer
// input, and success otherwise.
func failsOnNegative() *Subject {
	return &Subject{Call: func(x any) error {
		if signedOf(x) < 0 {
			return ErrOwn
		}
		return nil
	}}
}

// dereferencesHandle reads a handle without checking it is there.
func dereferencesHandle() *Subject {
	return &Subject{
		Ctx: func(ctx context.Context, _ any) error {
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
// rises by step on each call and each advance, whatever the input. It
// advances 5 steps.
func counter(step int) *Subject {
	count := 0
	return &Subject{
		Call: func(any) error {
			count += step
			return nil
		},
		Advance: func() error {
			count += step
			return nil
		},
		Steps:   5,
		Observe: func() int { return count },
	}
}

// setsValue returns a cell that starts at 0, which a call sets to its
// integer input. An assertion's input is 7.
func setsValue() *Subject {
	cell := 0
	return &Subject{
		Call: func(x any) error {
			cell = int(signedOf(x))
			return nil
		},
		Input:   7,
		Observe: func() int { return cell },
	}
}

// countsCalls returns how many times it has been called, starting at 1,
// whatever its input. An assertion's input is 1.
func countsCalls() *Subject {
	var calls int64
	return &Subject{
		Compute: func(any) any {
			calls++
			return calls
		},
		Input: 1,
	}
}

// combines returns a subject that combines two integers with op. An
// assertion's operands are 2, 3 and 5.
func combines(op func(a, b int64) int64) *Subject {
	return &Subject{
		Combine: func(a, b any) any { return op(signedOf(a), signedOf(b)) },
		A:       2,
		B:       3,
		C:       5,
	}
}

// renders returns a subject that renders an integer as decimal text, and
// one that drops the sign renders the integer's absolute value. An
// assertion's input is -42.
func renders(dropSign bool) *Subject {
	return &Subject{
		Input: -42,
		Render: func(x any) string {
			n := signedOf(x)
			if dropSign {
				n = max(n, -n)
			}
			return strconv.FormatInt(n, 10)
		},
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

// function returns the builder of a subject that is the function f of its
// input.
func function(f func(x any) any) func() *Subject {
	return func() *Subject { return &Subject{Function: f} }
}

// dropsTheFirst returns x, a list, without its first item, and the empty
// list as it is.
func dropsTheFirst(x any) any {
	items := x.([]any)
	return items[min(1, len(items)):]
}

// sorted returns x, a list of integers, in ascending order.
func sorted(x any) any {
	items := slices.Clone(x.([]any))
	slices.SortStableFunc(items, func(a, b any) int { return cmp.Compare(signedOf(a), signedOf(b)) })
	return items
}

// ascending returns the predicate over two integers that is true when the
// first is not greater than the second.
func ascending() *Subject {
	return &Subject{Ordered: func(first, second any) bool { return signedOf(first) <= signedOf(second) }}
}
