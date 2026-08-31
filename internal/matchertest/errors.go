// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"errors"
	"fmt"
	"testing"
)

// Sentinels the error cases match against. They are exported so a
// surface's own tests can reuse them rather than declaring their own,
// which would leave the two sets free to disagree.
var (
	// ErrSample is the error the cases expect to find.
	ErrSample = errors.New("matchertest: sample")
	// ErrOther is a distinct error, for the cases about telling two
	// sentinels apart.
	ErrOther = errors.New("matchertest: other")
)

// wrapped returns ErrSample wrapped twice, so a case proves the chain
// is walked rather than only the outermost error compared.
func wrapped() error {
	return fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrSample))
}

// NoErrorCases are the cases every surface's no-error assertion must
// produce. Drive them with [RunOne].
func NoErrorCases() []Case {
	return []Case{
		{Name: "a nil error passes", Args: []any{error(nil)}},
		{
			Name:      "an error reports its text",
			Args:      []any{ErrSample},
			Fails:     true,
			Assertion: "err-absent",
		},
	}
}

// HasErrorCases are the cases every surface's has-error assertion must
// produce. Drive them with [RunOne].
func HasErrorCases() []Case {
	return []Case{
		{Name: "an error passes", Args: []any{ErrSample}},
		{Name: "a wrapped error passes", Args: []any{wrapped()}},
		{
			Name:      "a nil error reports",
			Args:      []any{error(nil)},
			Fails:     true,
			Assertion: "err-present",
		},
	}
}

// ErrorIsCases are the cases every surface's error-identity assertion
// must produce. Drive them with [RunPair].
func ErrorIsCases() []Case {
	return []Case{
		{Name: "the same error matches", Args: []any{ErrSample, ErrSample}},
		{Name: "a wrapped error matches through the chain", Args: []any{wrapped(), ErrSample}},
		{
			Name:      "a different error reports both",
			Args:      []any{ErrOther, ErrSample},
			Fails:     true,
			Assertion: "err-is",
		},
		{
			Name:      "a nil error reports",
			Args:      []any{error(nil), ErrSample},
			Fails:     true,
			Assertion: "err-is",
		},
	}
}

// ErrorIsNotCases are the cases every surface's error-distinctness
// assertion must produce. Drive them with [RunPair].
func ErrorIsNotCases() []Case {
	return []Case{
		{Name: "two distinct errors pass", Args: []any{ErrOther, ErrSample}},
		{Name: "a nil error is distinct from a sentinel", Args: []any{error(nil), ErrSample}},
		{
			Name:      "the same error reports",
			Args:      []any{ErrSample, ErrSample},
			Fails:     true,
			Assertion: "err-is-not",
		},
		{
			Name:      "a wrapped error reports",
			Args:      []any{wrapped(), ErrSample},
			Fails:     true,
			Assertion: "err-is-not",
		},
	}
}

// AsError reads a case argument as an error, answering a nil error for
// a nil argument.
//
// A plain type assertion panics on that argument: a nil stored in an
// any holds no type, so it is not an error, and every adapter would
// otherwise need the same comma-ok dance.
func AsError(v any) error {
	err, _ := v.(error)
	return err
}

// TypedField is the value [WrappedTyped] carries, so a test can check
// that the error it got back is the one from the chain rather than a
// fresh zero value.
const TypedField = "typed"

// TypedError is a concrete error type, for the cases about pulling one
// out of a chain by type rather than by identity.
type TypedError struct {
	// Field distinguishes an unwrapped error from a zero value.
	Field string
}

// Error implements the error interface.
func (*TypedError) Error() string { return "matchertest: typed error" }

// WrappedTyped returns a [TypedError] wrapped twice, so finding it
// proves the chain was walked.
func WrappedTyped() error {
	return fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", &TypedError{Field: TypedField}))
}

// ErrorAsInvoke calls a surface's error-extraction assertion. The
// target type is fixed at [TypedError]: what varies between surfaces
// is the call, not the type parameter.
type ErrorAsInvoke func(seat *Seat, err error, msg string) *TypedError

// RunErrorAs drives invoke against every case an error-extraction
// assertion must produce.
func RunErrorAs(t *testing.T, invoke ErrorAsInvoke) {
	t.Helper()

	t.Run("finds a matching type in the chain", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		got := invoke(seat, WrappedTyped(), contractMsg)

		checkOutcome(t, seat, Case{})
		if got == nil || got.Field != TypedField {
			t.Fatalf("returned %+v, want the error from the chain", got)
		}
	})

	t.Run("finds an unwrapped error of the type", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		got := invoke(seat, &TypedError{Field: TypedField}, contractMsg)

		checkOutcome(t, seat, Case{})
		if got == nil {
			t.Fatal("returned nil for an error already of the target type")
		}
	})

	t.Run("reports when no error in the chain matches", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		_ = invoke(seat, ErrSample, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "err-as"})
	})

	t.Run("reports for a nil error", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		_ = invoke(seat, nil, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "err-as"})
	})

	t.Run("returns the zero value when nothing matches", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		got := invoke(seat, ErrSample, contractMsg)

		if got != nil {
			t.Fatalf("returned %+v on failure, want the zero value", got)
		}
	})
}
