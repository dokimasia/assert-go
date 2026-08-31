// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"fmt"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// Failure is what a failing assertion reports.
//
// Assertion is the canonical id the definition names, Contract is the
// caller's message unchanged, and Detail carries exactly the fields
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
// A seat satisfying it receives the record; a seat that does not
// receives the rendered sentence through Fatalf or Errorf. aborting is
// true for the aborting surface and false for the recording one.
type Reporter interface {
	Report(f Failure, aborting bool)
}

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
// The contract leads, then the detail. Rendering is not standardised:
// every implementation holds the record in the same shape and phrases
// it in its own conventions.
func Render(f Failure) string {
	if len(f.Detail) == 0 {
		return f.Contract
	}
	if diff := equalDiff(f); diff != "" {
		return f.Contract + ": (-want +got)\n" + diff
	}

	named := make(map[string]bool, len(order))
	for _, name := range order {
		named[name] = true
	}
	rest := make([]string, 0, len(f.Detail))
	for name := range f.Detail {
		if !named[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)

	var b strings.Builder
	b.WriteString(f.Contract)
	b.WriteString(": ")
	first := true
	for _, name := range append(slices.Clone(order), rest...) {
		value, ok := f.Detail[name]
		if !ok {
			continue
		}
		if !first {
			b.WriteString(", ")
		}
		first = false
		fmt.Fprintf(&b, "%s %+v", name, value)
	}
	return b.String()
}

// site reads the call site skip frames above the caller of site.
//
// It returns the zero Where when the frame cannot be read, which a
// reader treats as absent.
func site(skip int) Where {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return Where{}
	}
	return Where{File: file, Line: line}
}

// equalDiff answers the diff for an equality mismatch, and the empty
// string for anything else.
//
// A structural diff is what a Go reader wants from a mismatch, and
// printing two large structs side by side is not. The record carries
// want and got as the definition states; this is how Go says them.
func equalDiff(f Failure) string {
	if f.Assertion != "equal" {
		return ""
	}
	want, hasWant := f.Detail["want"]
	got, hasGot := f.Detail["got"]
	if !hasWant || !hasGot {
		return ""
	}
	return cmp.Diff(want, got)
}
