// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// Writer turns the records and the faults of this module into the text that
// a seat receives. Every text that the module sends to a seat comes from
// the one writer, so a writer of another form changes the text of the whole
// module and no code that reports.
type Writer interface {
	// Failure returns the text of a failure's record.
	Failure(f Failure) string
	// Fault returns the text of a fault.
	Fault(err error) string
}

// writer is the writer of this module: the text writer.
var writer Writer = text{}

// sentences are the sentences of the records of the assertions whose
// package writes its own, by assertion. [RegisterSentence] writes the table
// only while a package initialises, which completes before any test
// starts, so its readers take no lock.
var sentences = map[string]func(Failure) string{}

// RegisterSentence makes sentence the text writer's text of the records of
// each of assertions. prop calls it from its init function for the property
// assertions, whose sentence states a run, because this package does not
// import prop. Call it only while a package initialises.
//
// # Panics
//
// It panics for an assertion whose sentence is registered already, and
// registers nothing then.
func RegisterSentence(sentence func(Failure) string, assertions ...string) {
	for _, assertion := range assertions {
		if _, ok := sentences[assertion]; ok {
			panic(fmt.Sprintf("matcher: RegisterSentence registers the sentence of %s a second time", assertion))
		}
	}
	for _, assertion := range assertions {
		sentences[assertion] = sentence
	}
}

// text is the writer of the sentence that a Go reader expects: the
// contract, then the detail, and a diff labelled -want +got for a mismatch.
type text struct{}

// Failure returns the registered sentence of the record's assertion, when
// [RegisterSentence] registered one. For any other record without detail,
// it returns the contract alone. For a mismatch of equal or of a golden
// comparison, it returns the contract and a diff labelled -want +got.
// Otherwise it returns the contract and each field of the detail with its
// value, want before got and the rest in a fixed reading order, with a
// field that the order does not name after them, alphabetically.
func (text) Failure(f Failure) string {
	if sentence, ok := sentences[f.Assertion]; ok {
		return sentence(f)
	}
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

// Fault returns the text of err that its Error method writes.
func (text) Fault(err error) string {
	return err.Error()
}

// Render returns the writer's text of a failure's record.
//
// The standard fixes the record and not the sentence. Every implementation
// reports the same record, and writes its text by the conventions of its
// language.
//
// # Allocation contract
//
// Render allocates 5 times for a record of two int fields.
func Render(f Failure) string {
	return writer.Failure(f)
}

// RenderFault returns the writer's text of a fault.
//
// # Allocation contract
//
// RenderFault allocates what the Error method of err allocates: twice for
// a fault of an operation, a field and a reason.
func RenderFault(err error) string {
	return writer.Fault(err)
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

// order is the order in which the text states the fields of a detail: want
// before got, as the standard library writes them, and the rest in a fixed
// reading order. A field that order does not list follows them,
// alphabetically.
var order = []string{
	"want", "got", "length", "haystack", "needle", "index",
	"prefix", "suffix", "pattern", "tolerance", "low", "high",
	"first", "second", "attempts", "last", "leaked", "field",
}

// diffed are the assertions whose text states want and got as a diff:
// equal, and the three golden comparisons.
var diffed = map[string]bool{
	"equal":                   true,
	"golden-match":            true,
	"golden-match-at":         true,
	"golden-match-json-field": true,
}

// equalDiff returns the diff of want and got for an assertion in diffed,
// and the empty string for anything else.
//
// The record contains want and got under the definition's names, and the
// text states the two as a structural diff, which a Go reader reads more
// easily than two large structs side by side.
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
	// render it, the text states want and got as fields instead. cmp panics
	// on a value it cannot walk, and that panic would end the test at its
	// first failure.
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
	// can show a difference that the comparison ignored, and cannot miss
	// one that it counted.
	return cmp.Diff(want, got, Options()...)
}
