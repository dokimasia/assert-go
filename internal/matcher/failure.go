// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"fmt"
	"runtime"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
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

// Where is the call site a failure came from. Line is zero when the
// frame could not be read.
type Where struct {
	File string
	Line int
}

// Reporter is a [Seat] that takes the record rather than the sentence.
//
// [Fail] passes the record to a Reporter, and the rendered sentence to
// any other seat, through Fatalf or Errorf. aborting is true for the
// aborting surface and false for the recording one.
type Reporter interface {
	Report(f Failure, aborting bool)
}

// named is the set of the fields in order, built once so rendering a
// failure does not rebuild it.
var named = func() map[string]bool {
	out := make(map[string]bool, len(order))
	for _, name := range order {
		out[name] = true
	}
	return out
}()

// order is the sequence Go names detail fields in, which is want
// before got and the rest in a fixed reading order. A field not listed
// here sorts after these, alphabetically.
//
// The standard fixes the record, not the sentence. This is Go's
// phrasing of it, and it follows the want-then-got convention the
// standard library uses.
var order = []string{
	"want", "got", "length", "haystack", "needle", "index",
	"prefix", "suffix", "pattern", "tolerance", "low", "high",
	"first", "second", "attempts", "last", "leaked", "field",
}

// Render turns a record into the sentence a person reads.
//
// The contract leads, then the detail. Rendering is not standardised.
// Every implementation keeps the record in the same shape and phrases
// it in its own conventions.
func Render(f Failure) string {
	if len(f.Detail) == 0 {
		return f.Contract
	}
	if diff := equalDiff(f); diff != "" {
		return f.Contract + ": (-want +got)\n" + diff
	}

	rest := make([]string, 0, len(f.Detail))
	for name := range f.Detail {
		if !named[name] {
			rest = append(rest, name)
		}
	}
	slices.Sort(rest)

	var b strings.Builder
	b.WriteString(f.Contract)
	b.WriteString(": ")
	first := true
	write := func(name string) {
		value, ok := f.Detail[name]
		if !ok {
			return
		}
		if !first {
			b.WriteString(", ")
		}
		first = false
		fmt.Fprintf(&b, "%s %+v", name, value)
	}
	for _, name := range order {
		write(name)
	}
	for _, name := range rest {
		write(name)
	}
	return b.String()
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
	// failSkip is the number of frames that site skips before the caller
	// of Fail: runtime.Callers, site and Fail.
	failSkip = 3
)

// site returns the innermost frame of the caller's code among the callers
// of Fail. It reads the frames into an array on its own stack.
func site() Where {
	var pcs [maxFrames]uintptr
	n := runtime.Callers(failSkip, pcs[:])
	return CallerWhere(pcs[:n])
}

// CallerWhere returns the innermost frame of the caller's code among pcs:
// the first frame whose file is a test file, or whose function is outside
// this module and the runtime. It returns the zero Where when no frame is,
// which a reader treats as absent.
func CallerWhere(pcs []uintptr) Where {
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		if callers(frame) {
			return Where{File: frame.File, Line: frame.Line}
		}
		if !more {
			return Where{}
		}
	}
}

// callers reports whether frame is the caller's code.
func callers(frame runtime.Frame) bool {
	if strings.HasSuffix(frame.File, testSuffix) {
		return true
	}
	inside := strings.HasPrefix(frame.Function, modulePath+".") || strings.HasPrefix(frame.Function, modulePath+"/")
	return !inside && !strings.HasPrefix(frame.Function, runtimePrefix)
}

// diffed are the assertions whose want and got the sentence states as a
// diff: equal, and the three golden comparisons.
var diffed = map[string]bool{
	"equal":                   true,
	"golden-match":            true,
	"golden-match-at":         true,
	"golden-match-json-field": true,
}

// equalDiff returns the diff of a mismatch whose want and got the sentence
// states as a diff, and the empty string for anything else.
//
// A structural diff is what a Go reader wants from a mismatch, and
// printing two large structs side by side is not. The record contains
// want and got as the definition names them, and the diff is Go's
// sentence for the two.
func equalDiff(f Failure) (diff string) {
	if !diffed[f.Assertion] {
		return ""
	}
	want, hasWant := f.Detail["want"]
	got, hasGot := f.Detail["got"]
	if !hasWant || !hasGot {
		return ""
	}

	// A diff explains a failure and does not decide one. When cmp cannot
	// render it, the sentence states want and got instead. cmp panics on a
	// value it cannot walk, and that panic would arrive at the moment a
	// test first fails.
	defer func() {
		if recover() != nil {
			diff = ""
		}
	}()

	// The options the comparison itself used. Without the exporter cmp
	// refuses any value with an unexported field, which is most of them,
	// and this library states that unexported fields take part.
	//
	// The caller's relaxations are absent: a record contains what the
	// standard states, and an option is not part of it. Each relaxation
	// only widens what counts as equal, so a diff rendered without them
	// can name a difference the comparison forgave and cannot miss one
	// it did not.
	return cmp.Diff(want, got, Options()...)
}
