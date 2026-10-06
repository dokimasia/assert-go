// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"

	"go.dokimi.dev/assert/internal/record"
)

// The fields of a detail that the accessors of a failure record read.
const (
	// wantField is what an assertion required.
	wantField = "want"
	// gotField is what an assertion observed.
	gotField = "got"
	// failureField is the failure record of a property's failing case.
	failureField = "failure"
)

// Failure is what a failing assertion reports.
//
// Assertion is the canonical id the definition names, Contract is the
// caller's message unchanged, and Detail contains exactly the fields
// that assertion declares. Where is the call site, and is absent when
// the frame could not be read.
type Failure struct {
	Assertion string
	Contract  string
	Detail    map[string]any
	Where     Where
}

// Want returns the field want of the record's detail, and whether the
// record's assertion declares want. A declared want can be nil.
//
// # Allocation contract
//
// Want allocates nothing.
func (f Failure) Want() (any, bool) {
	want, declared := f.Detail[wantField]
	return want, declared
}

// Got returns the field got of the record's detail, and whether the
// record's assertion declares got. A declared got can be nil.
//
// # Allocation contract
//
// Got allocates nothing.
func (f Failure) Got() (any, bool) {
	got, declared := f.Detail[gotField]
	return got, declared
}

// CaseFailure returns the failure record of the failing case, which the
// field failure of a property's record contains, and whether the record
// contains one: the minimal case of a counterexample, or the case that the
// replay of a flaky run contradicted. A record of an assertion that
// declares no failure, and a property's record whose failure is nil,
// contain none.
//
// # Allocation contract
//
// CaseFailure allocates nothing.
func (f Failure) CaseFailure() (Failure, bool) {
	failure, contained := f.Detail[failureField].(Failure)
	return failure, contained
}

// failureRecord is a failure record as the detail of a property's call
// record states it.
type failureRecord struct {
	Assertion string          `json:"assertion"`
	Contract  string          `json:"contract"`
	Detail    json.RawMessage `json:"detail"`
	Where     *record.Where   `json:"where,omitempty"`
}

// MarshalJSON returns the failure record as the detail of a property's call
// record states it: the assertion, the contract, the typed literal of each
// field of the detail, and the call site with the base name of its file,
// which a record without a site leaves out.
//
// # Allocation contract
//
// MarshalJSON allocates 26 times for a record of two int fields with a
// call site.
func (f Failure) MarshalJSON() ([]byte, error) {
	out := failureRecord{Assertion: f.Assertion, Contract: f.Contract, Detail: detailOf(f.Detail)}
	if f.Where != (Where{}) {
		out.Where = &record.Where{File: filepath.Base(f.Where.File), Line: f.Where.Line}
	}
	return json.Marshal(out)
}

// Where is the call site a failure came from. Line is zero when the
// frame could not be read.
type Where struct {
	File string
	Line int
}

// Reporter is a [Seat] that takes the record rather than the sentence.
//
// [Fail] passes the record to a Reporter, and the writer's text of it to
// any other seat, through Fatalf or Errorf. aborting is true for the
// aborting surface and false for the recording one.
type Reporter interface {
	Report(f Failure, aborting bool)
}

// The frames that are not the caller's code.
const (
	// modulePath is the import path of this module. A frame of a function
	// inside it is not the caller's code, unless its file is a test file.
	modulePath = "go.dokimi.dev/assert"
	// runtimePrefix starts the name of every function of the runtime.
	runtimePrefix = "runtime."
	// testSuffix ends the name of every Go test file.
	testSuffix = "_test.go"
	// maxFrames is the most frames that a location is searched in.
	maxFrames = 64
	// siteSkip is the number of frames that site skips: runtime.Callers,
	// site and its caller, which is a function of this module.
	siteSkip = 3
)

// site returns the innermost frame of the caller's code among the callers
// of the function that calls site. It reads the frames into an array on
// its own stack.
func site() Where {
	var pcs [maxFrames]uintptr
	n := runtime.Callers(siteSkip, pcs[:])
	return CallerWhere(pcs[:n])
}

// CallerWhere returns the innermost frame of the caller's code among pcs:
// the first frame whose file is a test file, or whose function is outside
// this module and the runtime. It returns the zero Where when no frame is,
// which a reader treats as absent.
//
// # Allocation contract
//
// CallerWhere allocates once: the iterator over the frames of pcs.
func CallerWhere(pcs []uintptr) Where {
	frames := runtime.CallersFrames(pcs)
	for more := true; more; {
		var frame runtime.Frame
		frame, more = frames.Next()
		if callers(frame) {
			return Where{File: frame.File, Line: frame.Line}
		}
	}
	return Where{}
}

// callers reports whether frame is the caller's code.
func callers(frame runtime.Frame) bool {
	if strings.HasSuffix(frame.File, testSuffix) {
		return true
	}
	inside := strings.HasPrefix(frame.Function, modulePath+".") || strings.HasPrefix(frame.Function, modulePath+"/")
	return !inside && !strings.HasPrefix(frame.Function, runtimePrefix)
}
