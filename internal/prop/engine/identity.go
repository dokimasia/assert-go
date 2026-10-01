// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"cmp"
	"reflect"
	"runtime"
	"strings"

	"go.dokimi.dev/assert"
)

// The frames that are not the caller's code.
const (
	// modulePath is the import path of this module. A frame of a function
	// inside it is not the caller's code, unless its file is a test file.
	modulePath = "go.dokimi.dev/assert"
	// runtimePrefix starts the name of every function of the runtime.
	runtimePrefix = "runtime."
	// testSuffix ends the name of every Go test file.
	testSuffix = "_test.go"
	// maxFrames is the most frames a failure's location is searched in.
	maxFrames = 64
)

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
// innermost frame of the caller's code among pcs.
func panicIdentity(v any, pcs []uintptr) Identity {
	return Identity{Panic: reflect.TypeOf(v).String(), Where: callerWhere(pcs)}
}

// callerWhere returns the innermost frame of the caller's code among pcs:
// the first frame whose file is a test file, or whose function is outside
// this module and the runtime. It returns the zero Where when there is
// none.
func callerWhere(pcs []uintptr) assert.Where {
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		if callers(frame) {
			return assert.Where{File: frame.File, Line: frame.Line}
		}
		if !more {
			return assert.Where{}
		}
	}
}

// callers reports whether frame is the caller's code.
func callers(frame runtime.Frame) bool {
	if strings.HasSuffix(frame.File, testSuffix) {
		return true
	}
	inside := strings.HasPrefix(frame.Function, modulePath+".") || strings.HasPrefix(frame.Function, modulePath+"/")
	return !inside && !strings.HasPrefix(frame.Function, runtimePrefix) && frame.Function != ""
}
