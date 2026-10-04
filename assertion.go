// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Assertion applies several assertions to one value without repeating
// it. Every method returns the receiver, so calls join with a dot.
//
// The first failing method stops the test, so later methods in the
// chain never run. Where every property is worth reporting at once,
// use the recording surface instead.
//
// The zero value is not usable. Call [That]. An Assertion is not safe for
// concurrent use. Call its methods from one goroutine.
type Assertion[T any] struct {
	// tb is where a failing method reports.
	tb TB
	// got is the value every method compares against. For a reference
	// type, the chain keeps the reference, so a method sees a later change
	// of the referenced value.
	got T
}

// That starts an assertion chain on got.
//
//	assert.That(t, resp.StatusCode).
//	    NotEqual(0, "the status was set").
//	    Equal(200, "the request succeeded")
//
// Use it where several properties of one value are worth stating
// together. For a single property the function form reads better:
//
//	assert.Equal(t, resp.StatusCode, 200, "the request succeeded")
//
// # Allocation contract
//
// That allocates nothing for a chain that does not escape the caller's
// frame, such as a chain of one expression or one in a local variable. A
// chain that escapes to the heap, such as one in a package variable,
// allocates once.
func That[T any](tb TB, got T) *Assertion[T] {
	// That marks no helper frame, because it reports nothing. Its body is
	// small enough for the compiler to inline in a coverage build too, so a
	// chain that does not escape is allocated on the stack.
	return &Assertion[T]{tb: tb, got: got}
}

// Equal compares the chained value against want and stops the test
// when they differ. See [Equal] for the comparison rules and the
// failure shape.
//
// # Allocation contract
//
// A passing call on a chain of an int below 256 allocates nothing, as
// [Equal] does.
func (a *Assertion[T]) Equal(want T, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.Equal(a.tb, matcher.Fatal, a.got, want, msg, opts...)
	return a
}

// NotEqual compares the chained value against want and stops the test
// when they are equal. See [NotEqual] for the comparison rules and the
// failure shape.
//
// # Allocation contract
//
// A passing call on a chain of an int below 256 allocates nothing, as
// [NotEqual] does.
func (a *Assertion[T]) NotEqual(want T, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.NotEqual(a.tb, matcher.Fatal, a.got, want, msg, opts...)
	return a
}

// Nil stops the test when the chained value is not nil. See [Nil].
//
// # Allocation contract
//
// A passing call on a chain of a nil pointer allocates nothing.
func (a *Assertion[T]) Nil(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Nil(a.tb, matcher.Fatal, a.got, msg)
	return a
}

// NotNil stops the test when the chained value is nil. See [NotNil].
//
// # Allocation contract
//
// A passing call on a chain of a pointer allocates nothing.
func (a *Assertion[T]) NotNil(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.NotNil(a.tb, matcher.Fatal, a.got, msg)
	return a
}

// Length stops the test when the chained value does not have want items.
// See [Length].
//
// # Allocation contract
//
// A passing call on a chain of a slice allocates nothing.
func (a *Assertion[T]) Length(want int, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Length(a.tb, matcher.Fatal, a.got, want, msg)
	return a
}

// Empty stops the test when the chained value has any item. See [Empty].
//
// # Allocation contract
//
// A passing call on a chain of a slice allocates nothing.
func (a *Assertion[T]) Empty(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Empty(a.tb, matcher.Fatal, a.got, msg)
	return a
}

// NotEmpty stops the test when the chained value has no item. See
// [NotEmpty].
//
// # Allocation contract
//
// A passing call on a chain of a slice allocates nothing.
func (a *Assertion[T]) NotEmpty(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.NotEmpty(a.tb, matcher.Fatal, a.got, msg)
	return a
}

// Contains stops the test when the chained value does not contain
// needle. See [Contains].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) Contains(needle any, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.Contains(a.tb, matcher.Fatal, a.got, needle, msg, opts...)
	return a
}

// NotContains stops the test when the chained value contains needle. See
// [NotContains].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) NotContains(needle any, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.NotContains(a.tb, matcher.Fatal, a.got, needle, msg, opts...)
	return a
}

// ContainsInOrder stops the test when the chained value does not contain
// every needle in order. See [ContainsInOrder].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) ContainsInOrder(needles []string, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.ContainsInOrder(a.tb, matcher.Fatal, a.got, needles, msg)
	return a
}

// HasPrefix stops the test when the chained value does not start with
// prefix. See [HasPrefix].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) HasPrefix(prefix, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.HasPrefix(a.tb, matcher.Fatal, a.got, prefix, msg)
	return a
}

// HasSuffix stops the test when the chained value does not end with
// suffix. See [HasSuffix].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) HasSuffix(suffix, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.HasSuffix(a.tb, matcher.Fatal, a.got, suffix, msg)
	return a
}

// Matches stops the test when the chained value does not match
// pattern. See [Matches].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates 63 times: the chained
// string in an interface, and the compilation of pattern that [Matches]
// makes.
func (a *Assertion[T]) Matches(pattern, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Matches(a.tb, matcher.Fatal, a.got, pattern, msg)
	return a
}

// CloseTo stops the test when the chained value is further than
// tolerance from want. See [CloseTo].
//
// # Allocation contract
//
// A passing call on a chain of a float64 allocates once: the chained
// value in an interface.
func (a *Assertion[T]) CloseTo(want, tolerance float64, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.CloseTo(a.tb, matcher.Fatal, a.got, want, tolerance, msg)
	return a
}

// InRange stops the test when the chained value is outside the closed
// interval [low, high]. See [InRange].
//
// # Allocation contract
//
// A passing call on a chain of a float64 allocates once: the chained
// value in an interface.
func (a *Assertion[T]) InRange(low, high float64, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.InRange(a.tb, matcher.Fatal, a.got, low, high, msg)
	return a
}
