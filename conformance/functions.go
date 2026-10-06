// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"context"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

// functions are the assertion functions of one surface, the package assert
// or the package expect, that the registry and the subject drivers call.
// Each field is the surface's function of the field's name, instantiated
// for the values that a corpus case decodes to and the subjects build.
// Permutation has a field for each element type that a case's slices have.
type functions struct {
	Equal               func(tb assert.TB, got, want any, msg string, opts ...assert.Option)
	NotEqual            func(tb assert.TB, got, want any, msg string, opts ...assert.Option)
	True                func(tb assert.TB, cond bool, msg string)
	False               func(tb assert.TB, cond bool, msg string)
	Nil                 func(tb assert.TB, got any, msg string)
	NotNil              func(tb assert.TB, got any, msg string)
	Length              func(tb assert.TB, got any, want int, msg string)
	Empty               func(tb assert.TB, got any, msg string)
	NotEmpty            func(tb assert.TB, got any, msg string)
	Contains            func(tb assert.TB, haystack, needle any, msg string, opts ...assert.Option)
	NotContains         func(tb assert.TB, haystack, needle any, msg string, opts ...assert.Option)
	ContainsInOrder     func(tb assert.TB, got any, needles []string, msg string)
	PermutationOfInts   func(tb assert.TB, got, want []int, msg string, opts ...assert.Option)
	PermutationOfFloats func(tb assert.TB, got, want []float64, msg string, opts ...assert.Option)
	PermutationOfItems  func(tb assert.TB, got, want []any, msg string, opts ...assert.Option)
	HasPrefix           func(tb assert.TB, got any, prefix, msg string)
	HasSuffix           func(tb assert.TB, got any, suffix, msg string)
	Matches             func(tb assert.TB, got any, pattern, msg string)
	CloseTo             func(tb assert.TB, got any, want, tolerance float64, msg string)
	InRange             func(tb assert.TB, got any, low, high float64, msg string)
	Panics              func(tb assert.TB, fn func(), msg string) any
	NotPanics           func(tb assert.TB, fn func(), msg string)
	HonoursCancellation func(tb assert.TB, fn func(ctx context.Context) error, msg string)
	HonoursDeadline     func(tb assert.TB, fn func(ctx context.Context) error, msg string)
	NilContextSafe      func(tb assert.TB, fn func(ctx context.Context) error, msg string)
	Pure                func(tb assert.TB, observe func() int, fn func(), msg string, opts ...assert.Option)
	NotPure             func(tb assert.TB, observe func() int, fn func(), msg string, opts ...assert.Option)
	Eventually          func(tb assert.TB, timeout, interval time.Duration, fn func(tb assert.TB), msg string)
	EventuallyTrue      func(tb assert.TB, timeout time.Duration, pred func() bool, msg string)
	Idempotent          func(
		tb assert.TB, call func(any) error, input any, observe func() int, msg string, opts ...assert.Option)
	Accumulates   func(tb assert.TB, call func(any) error, input any, observe func() int, msg string)
	Deterministic func(tb assert.TB, call func(int) (int, error), input int, msg string, opts ...assert.Option)
	Commutative   func(tb assert.TB, combine func(a, b int) int, a, b int, msg string, opts ...assert.Option)
	Associative   func(tb assert.TB, combine func(a, b int) int, a, b, c int, msg string, opts ...assert.Option)
	RoundTrip     func(tb assert.TB, forward func(int) (string, error), inverse func(string) (int, error), input int,
		msg string, opts ...assert.Option)
	StableOrder     func(tb assert.TB, iterate func() ([]any, error), msg string, opts ...assert.Option)
	NoDuplicates    func(tb assert.TB, iterate func() ([]any, error), msg string, opts ...assert.Option)
	Monotonic       func(tb assert.TB, observe func() int, advance func() error, steps int, msg string)
	Total           func(tb assert.TB, call func(int) error, domain []int, msg string)
	FailsAfterClose func(tb assert.TB, closer, call func() error, sentinel error, msg string)
	Poisoned        func(tb assert.TB, induce func(), observe func() error, msg string)
}

// abortingFunctions are the functions of the surface that stops the test at
// the first failure.
var abortingFunctions = functions{
	Equal:               assert.Equal[any],
	NotEqual:            assert.NotEqual[any],
	True:                assert.True,
	False:               assert.False,
	Nil:                 assert.Nil,
	NotNil:              assert.NotNil,
	Length:              assert.Length,
	Empty:               assert.Empty,
	NotEmpty:            assert.NotEmpty,
	Contains:            assert.Contains,
	NotContains:         assert.NotContains,
	ContainsInOrder:     assert.ContainsInOrder,
	PermutationOfInts:   assert.Permutation[int],
	PermutationOfFloats: assert.Permutation[float64],
	PermutationOfItems:  assert.Permutation[any],
	HasPrefix:           assert.HasPrefix,
	HasSuffix:           assert.HasSuffix,
	Matches:             assert.Matches,
	CloseTo:             assert.CloseTo,
	InRange:             assert.InRange,
	Panics:              assert.Panics,
	NotPanics:           assert.NotPanics,
	HonoursCancellation: assert.HonoursCancellation,
	HonoursDeadline:     assert.HonoursDeadline,
	NilContextSafe:      assert.NilContextSafe,
	Pure:                assert.Pure[int],
	NotPure:             assert.NotPure[int],
	Eventually:          assert.Eventually,
	EventuallyTrue:      assert.EventuallyTrue,
	Idempotent:          assert.Idempotent[any, int],
	Accumulates:         assert.Accumulates[any],
	Deterministic:       assert.Deterministic[int, int],
	Commutative:         assert.Commutative[int, int],
	Associative:         assert.Associative[int],
	RoundTrip:           assert.RoundTrip[int, string],
	StableOrder:         assert.StableOrder[any],
	NoDuplicates:        assert.NoDuplicates[any],
	Monotonic:           assert.Monotonic[int],
	Total:               assert.Total[int],
	FailsAfterClose:     assert.FailsAfterClose,
	Poisoned:            assert.Poisoned,
}

// recordingFunctions are the functions of the surface that records a
// failure and lets the test continue.
var recordingFunctions = functions{
	Equal:               expect.Equal[any],
	NotEqual:            expect.NotEqual[any],
	True:                expect.True,
	False:               expect.False,
	Nil:                 expect.Nil,
	NotNil:              expect.NotNil,
	Length:              expect.Length,
	Empty:               expect.Empty,
	NotEmpty:            expect.NotEmpty,
	Contains:            expect.Contains,
	NotContains:         expect.NotContains,
	ContainsInOrder:     expect.ContainsInOrder,
	PermutationOfInts:   expect.Permutation[int],
	PermutationOfFloats: expect.Permutation[float64],
	PermutationOfItems:  expect.Permutation[any],
	HasPrefix:           expect.HasPrefix,
	HasSuffix:           expect.HasSuffix,
	Matches:             expect.Matches,
	CloseTo:             expect.CloseTo,
	InRange:             expect.InRange,
	Panics:              expect.Panics,
	NotPanics:           expect.NotPanics,
	HonoursCancellation: expect.HonoursCancellation,
	HonoursDeadline:     expect.HonoursDeadline,
	NilContextSafe:      expect.NilContextSafe,
	Pure:                expect.Pure[int],
	NotPure:             expect.NotPure[int],
	Eventually:          expect.Eventually,
	EventuallyTrue:      expect.EventuallyTrue,
	Idempotent:          expect.Idempotent[any, int],
	Accumulates:         expect.Accumulates[any],
	Deterministic:       expect.Deterministic[int, int],
	Commutative:         expect.Commutative[int, int],
	Associative:         expect.Associative[int],
	RoundTrip:           expect.RoundTrip[int, string],
	StableOrder:         expect.StableOrder[any],
	NoDuplicates:        expect.NoDuplicates[any],
	Monotonic:           expect.Monotonic[int],
	Total:               expect.Total[int],
	FailsAfterClose:     expect.FailsAfterClose,
	Poisoned:            expect.Poisoned,
}
