// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Assertion applies several assertions to one value without repeating
// it. Every method returns the receiver, so calls join with a dot.
//
// Every method runs, whatever the ones before it reported, so a chain
// reports each failing property rather than only the first. Where the
// first failure should stop the test, use
// [go.dokimi.dev/assert.That].
//
// The zero value is not usable. Call [That]. An Assertion is not safe for
// concurrent use. Call its methods from one goroutine.
type Assertion[T any] struct {
	// tb is where a failing method reports.
	tb assert.TB
	// got is the value every method compares against. For a reference
	// type, the chain keeps the reference, so a method sees a later change
	// of the referenced value.
	got T
}

// That starts an assertion chain on got.
//
//	expect.That(t, user).
//	    NotNil("the user was found").
//	    HasPrefix("usr_", "the id starts with its prefix")
//
// Every method runs, so one run reports every failing property.
//
// # Allocation contract
//
// That allocates nothing for a chain that does not escape the caller's
// frame, such as a chain of one expression or one in a local variable. A
// chain that escapes to the heap, such as one in a package variable,
// allocates once.
func That[T any](tb assert.TB, got T) *Assertion[T] {
	// That marks no helper frame, because it reports nothing. Its body is
	// small enough for the compiler to inline in a coverage build too, so a
	// chain that does not escape is allocated on the stack.
	return &Assertion[T]{tb: tb, got: got}
}

// Equal compares the chained value against want and records a failure
// when they differ. The chain continues either way. See
// [go.dokimi.dev/assert.Equal] for the comparison rules and the
// failure shape.
//
// # Allocation contract
//
// A passing call on a chain of an int below 256 allocates nothing, as
// [Equal] does.
func (a *Assertion[T]) Equal(want T, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.Equal(a.tb, matcher.Soft, a.got, want, msg, opts...)
	return a
}

// NotEqual compares the chained value against want and records a
// failure when they are equal. The chain continues either way. See
// [go.dokimi.dev/assert.NotEqual] for the comparison rules and the
// failure shape.
//
// # Allocation contract
//
// A passing call on a chain of an int below 256 allocates nothing, as
// [NotEqual] does.
func (a *Assertion[T]) NotEqual(want T, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.NotEqual(a.tb, matcher.Soft, a.got, want, msg, opts...)
	return a
}

// Nil records a failure when the chained value is not nil. See
// [go.dokimi.dev/assert.Nil].
//
// # Allocation contract
//
// A passing call on a chain of a nil pointer allocates nothing.
func (a *Assertion[T]) Nil(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Nil(a.tb, matcher.Soft, a.got, msg)
	return a
}

// NotNil records a failure when the chained value is nil. See
// [go.dokimi.dev/assert.NotNil].
//
// # Allocation contract
//
// A passing call on a chain of a pointer allocates nothing.
func (a *Assertion[T]) NotNil(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.NotNil(a.tb, matcher.Soft, a.got, msg)
	return a
}

// Length records a failure when the chained value does not have want
// items. See [go.dokimi.dev/assert.Length].
//
// # Allocation contract
//
// A passing call on a chain of a slice allocates nothing.
func (a *Assertion[T]) Length(want int, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Length(a.tb, matcher.Soft, a.got, want, msg)
	return a
}

// Empty records a failure when the chained value has any item. See
// [Empty].
//
// # Allocation contract
//
// A passing call on a chain of a slice allocates once: the interface of
// the slice, which a failure states as got.
func (a *Assertion[T]) Empty(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Empty(a.tb, matcher.Soft, a.got, msg)
	return a
}

// NotEmpty records a failure when the chained value has no item. See
// [NotEmpty].
//
// # Allocation contract
//
// A passing call on a chain of a slice allocates once: the interface of
// the slice, which a failure states as got.
func (a *Assertion[T]) NotEmpty(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.NotEmpty(a.tb, matcher.Soft, a.got, msg)
	return a
}

// Contains records a failure when the chained value does not contain
// needle. See [go.dokimi.dev/assert.Contains].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) Contains(needle any, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.Contains(a.tb, matcher.Soft, a.got, needle, msg, opts...)
	return a
}

// NotContains records a failure when the chained value contains needle.
// See [NotContains].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) NotContains(needle any, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.NotContains(a.tb, matcher.Soft, a.got, needle, msg, opts...)
	return a
}

// ContainsInOrder records a failure when the chained value does not
// contain every needle in order. See [go.dokimi.dev/assert.ContainsInOrder].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) ContainsInOrder(needles []string, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.ContainsInOrder(a.tb, matcher.Soft, a.got, needles, msg)
	return a
}

// HasPrefix records a failure when the chained value does not start with
// prefix. See [go.dokimi.dev/assert.HasPrefix].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) HasPrefix(prefix, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.HasPrefix(a.tb, matcher.Soft, a.got, prefix, msg)
	return a
}

// HasSuffix records a failure when the chained value does not end with
// suffix. See [go.dokimi.dev/assert.HasSuffix].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates once: the chained
// string in an interface.
func (a *Assertion[T]) HasSuffix(suffix, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.HasSuffix(a.tb, matcher.Soft, a.got, suffix, msg)
	return a
}

// Matches records a failure when the chained value does not match
// pattern. See [go.dokimi.dev/assert.Matches].
//
// # Allocation contract
//
// A passing call on a chain of a string allocates 63 times: the chained
// string in an interface, and the compilation of pattern that [Matches]
// makes.
func (a *Assertion[T]) Matches(pattern, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Matches(a.tb, matcher.Soft, a.got, pattern, msg)
	return a
}

// CloseTo records a failure when the chained value is further than
// tolerance from want. See [go.dokimi.dev/assert.CloseTo].
//
// # Allocation contract
//
// A passing call on a chain of a float64 allocates once: the chained
// value in an interface.
func (a *Assertion[T]) CloseTo(want, tolerance float64, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.CloseTo(a.tb, matcher.Soft, a.got, want, tolerance, msg)
	return a
}

// InRange records a failure when the chained value is outside the closed
// interval [low, high]. See [go.dokimi.dev/assert.InRange].
//
// # Allocation contract
//
// A passing call on a chain of a float64 allocates once: the chained
// value in an interface.
func (a *Assertion[T]) InRange(low, high float64, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.InRange(a.tb, matcher.Soft, a.got, low, high, msg)
	return a
}

// NoError records a failure when the chained value is an error, or a value
// of another type that is not nil. See [go.dokimi.dev/assert.NoError].
//
// # Allocation contract
//
// A passing call on a chain of a nil error allocates nothing.
func (a *Assertion[T]) NoError(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.NoError(a.tb, matcher.Soft, a.got, msg)
	return a
}

// HasError records a failure when the chained value is no error: nil, or a
// value of another type. See [go.dokimi.dev/assert.HasError].
//
// # Allocation contract
//
// A passing call on a chain of an error allocates nothing.
func (a *Assertion[T]) HasError(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.HasError(a.tb, matcher.Soft, a.got, msg)
	return a
}

// ErrorIs records a failure when the chained value is no error that
// matches target under [errors.Is]. See [go.dokimi.dev/assert.ErrorIs].
//
//	expect.That(t, err).
//	    ErrorIs(store.ErrMalformed, "Decode refuses the encoding").
//	    ErrorIs(codec.ErrVersion, "and wraps the codec's error")
//
// # Allocation contract
//
// A passing call on a chain of a sentinel wrapped twice allocates nothing.
func (a *Assertion[T]) ErrorIs(target error, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.ErrorIs(a.tb, matcher.Soft, a.got, target, msg)
	return a
}

// ErrorIsNot records a failure when the chained value matches target under
// [errors.Is], or is no error and not nil. See
// [go.dokimi.dev/assert.ErrorIsNot].
//
// # Allocation contract
//
// A passing call on a chain of a sentinel wrapped twice allocates nothing.
func (a *Assertion[T]) ErrorIsNot(target error, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.ErrorIsNot(a.tb, matcher.Soft, a.got, target, msg)
	return a
}
