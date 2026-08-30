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
// The zero value is not usable; call [That]. An Assertion is not safe
// for concurrent use, and a chain is a single expression in practice,
// so this costs nothing.
type Assertion[T any] struct {
	// tb is where a failing method reports.
	tb assert.TB
	// got is the value every method compares against. It is held
	// rather than copied, so a method sees later mutations of a
	// reference type.
	got T
}

// That starts an assertion chain on got.
//
//	expect.That(t, user).
//	    NotNil("the user was found").
//	    HasPrefix("usr_", "the id carries its prefix")
//
// Every method runs, so one run reports every failing property.
func That[T any](tb assert.TB, got T) *Assertion[T] {
	tb.Helper()
	return &Assertion[T]{tb: tb, got: got}
}

// Equal compares the chained value against want and records a failure
// when they differ. The chain carries on either way. See
// [go.dokimi.dev/assert.Equal] for the comparison rules and the
// failure shape.
func (a *Assertion[T]) Equal(want T, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.Equal(a.tb, matcher.Soft, a.got, want, msg, opts...)
	return a
}

// NotEqual compares the chained value against want and records a
// failure when they are equal. The chain carries on either way. See
// [go.dokimi.dev/assert.NotEqual] for the comparison rules and the
// failure shape.
func (a *Assertion[T]) NotEqual(want T, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.NotEqual(a.tb, matcher.Soft, a.got, want, msg, opts...)
	return a
}

// Nil records a failure when the chained value is not nil. See [go.dokimi.dev/assert.Nil].
func (a *Assertion[T]) Nil(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Nil(a.tb, matcher.Soft, a.got, msg)
	return a
}

// NotNil records a failure when the chained value is nil. See [go.dokimi.dev/assert.NotNil].
func (a *Assertion[T]) NotNil(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.NotNil(a.tb, matcher.Soft, a.got, msg)
	return a
}

// Length records a failure when the chained value does not hold want
// items. See [go.dokimi.dev/assert.Length].
func (a *Assertion[T]) Length(want int, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Length(a.tb, matcher.Soft, a.got, want, msg)
	return a
}

// Empty records a failure when the chained value holds anything. See
// [Empty].
func (a *Assertion[T]) Empty(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Empty(a.tb, matcher.Soft, a.got, msg)
	return a
}

// NotEmpty records a failure when the chained value holds nothing. See
// [NotEmpty].
func (a *Assertion[T]) NotEmpty(msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.NotEmpty(a.tb, matcher.Soft, a.got, msg)
	return a
}

// Contains records a failure when the chained value does not hold needle.
// See [go.dokimi.dev/assert.Contains].
func (a *Assertion[T]) Contains(needle any, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.Contains(a.tb, matcher.Soft, a.got, needle, msg, opts...)
	return a
}

// NotContains records a failure when the chained value holds needle. See
// [NotContains].
func (a *Assertion[T]) NotContains(needle any, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.NotContains(a.tb, matcher.Soft, a.got, needle, msg, opts...)
	return a
}

// ContainsInOrder records a failure when the chained value does not hold
// every needle in order. See [go.dokimi.dev/assert.ContainsInOrder].
func (a *Assertion[T]) ContainsInOrder(needles []string, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.ContainsInOrder(a.tb, matcher.Soft, a.got, needles, msg)
	return a
}

// HasPrefix records a failure when the chained value does not start with
// prefix. See [go.dokimi.dev/assert.HasPrefix].
func (a *Assertion[T]) HasPrefix(prefix, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.HasPrefix(a.tb, matcher.Soft, a.got, prefix, msg)
	return a
}

// HasSuffix records a failure when the chained value does not end with
// suffix. See [go.dokimi.dev/assert.HasSuffix].
func (a *Assertion[T]) HasSuffix(suffix, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.HasSuffix(a.tb, matcher.Soft, a.got, suffix, msg)
	return a
}

// Matches records a failure when the chained value does not match
// pattern. See [go.dokimi.dev/assert.Matches].
func (a *Assertion[T]) Matches(pattern, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.Matches(a.tb, matcher.Soft, a.got, pattern, msg)
	return a
}

// CloseTo records a failure when the chained value is further than
// tolerance from want. See [go.dokimi.dev/assert.CloseTo].
func (a *Assertion[T]) CloseTo(want, tolerance float64, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.CloseTo(a.tb, matcher.Soft, a.got, want, tolerance, msg)
	return a
}

// InRange records a failure when the chained value falls outside the
// closed interval [low, high]. See [go.dokimi.dev/assert.InRange].
func (a *Assertion[T]) InRange(low, high float64, msg string) *Assertion[T] {
	a.tb.Helper()
	matcher.InRange(a.tb, matcher.Soft, a.got, low, high, msg)
	return a
}
