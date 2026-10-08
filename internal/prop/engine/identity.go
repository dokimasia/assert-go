// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"cmp"
	"reflect"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// maxFrames is the most frames a failure's location is searched in.
const maxFrames = 64

// Identity is what a failure is kept by: the assertion that reported it
// and where, or the type of a panic's value and where it was raised. An
// assertion's record without a location is kept by its assertion and its
// contract, which are the same for every case that fails at one call site.
// Two failures are one failure when their identities are equal. A message
// is no part of an identity.
type Identity struct {
	// Assertion is the assertion that reported the failure, and empty for
	// a plain message and for a panic.
	Assertion string
	// Contract is the contract of an assertion's record without a
	// location, and empty otherwise.
	Contract string
	// Panic is the type of a panic's value, and empty otherwise.
	Panic string
	// Where is the innermost frame of the caller's code.
	Where assert.Where
}

// Compare orders identities by assertion, contract, panic type, file and
// line, so a run shrinks failures of equal size in one order.
func (i Identity) Compare(j Identity) int {
	return cmp.Or(
		cmp.Compare(i.Assertion, j.Assertion),
		cmp.Compare(i.Contract, j.Contract),
		cmp.Compare(i.Panic, j.Panic),
		cmp.Compare(i.Where.File, j.Where.File),
		cmp.Compare(i.Where.Line, j.Where.Line),
	)
}

// identityOf returns the identity of an assertion's record, or of a
// message that the body passed to the case.
func identityOf(f assert.Failure) Identity {
	id := Identity{Assertion: f.Assertion, Where: f.Where}
	if f.Assertion != "" && f.Where == (assert.Where{}) {
		id.Contract = f.Contract
	}
	return id
}

// panicIdentity returns the identity of a panic with value v, raised at the
// innermost frame of the caller's code among pcs, which
// [matcher.CallerWhere] finds.
func panicIdentity(v any, pcs []uintptr) Identity {
	return Identity{Panic: reflect.TypeOf(v).String(), Where: matcher.CallerWhere(pcs)}
}
