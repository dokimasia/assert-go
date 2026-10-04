// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.dokimi.dev/assert/internal/align"
	"go.dokimi.dev/assert/internal/equality"
	"go.dokimi.dev/assert/internal/text"
)

// maxDifferences is the most places of a difference that the text of a
// failure states.
const maxDifferences = 64

// contextLines is the number of unchanged lines that a line diff states
// before and after each change.
const contextLines = 3

// gapMark replaces the places past maxDifferences, and the unchanged lines
// of a line diff that no change is near.
const gapMark = "…"

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
var writer Writer = textWriter{}

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

// textWriter is the writer of the sentence that a Go reader expects: the
// contract, then the detail, and a diff labelled -want +got for a mismatch.
type textWriter struct{}

// Failure returns the registered sentence of the record's assertion, when
// [RegisterSentence] registered one. For any other record without detail,
// it returns the contract alone. For a mismatch of equal or of a golden
// comparison, it returns the contract and a diff labelled -want +got: each
// place where want and got differ on a line of its own, at most 64 of
// them. Otherwise it returns the contract and each field of the detail
// with its value, want before got and the rest in a fixed reading order,
// with a field that the order does not name after them, alphabetically. A
// value that contains itself, or that has more than 65,536 parts, states
// its bounded text as the package text writes it.
func (textWriter) Failure(f Failure) string {
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
		text.Fprintf(&b, "%s %+v", name, value)
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
func (textWriter) Fault(err error) string {
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
// and the empty string for anything else, or for two values that differ
// in no place.
//
// The record contains want and got under the definition's names, and the
// text states each place where they differ, which a Go reader reads more
// easily than two large structs side by side. The places are those of the
// default comparison: a record contains what the definition states, and an
// option is no part of it. A relaxation only widens what counts as equal,
// so the text can show a difference that the comparison ignored, and never
// misses one that it counted.
func equalDiff(f Failure) string {
	if !diffed[f.Assertion] {
		return ""
	}
	want, hasWant := f.Detail["want"]
	got, hasGot := f.Detail["got"]
	if !hasWant || !hasGot {
		return ""
	}

	x, y := reflect.ValueOf(&want).Elem(), reflect.ValueOf(&got).Elem()
	var b strings.Builder
	for i, d := range equality.Diff(x, y, equality.Rules{}, maxDifferences+1) {
		if i == maxDifferences {
			b.WriteString("\t" + gapMark + "\n")
			break
		}
		writeDifference(&b, d)
	}
	return b.String()
}

// writeDifference writes the line of one place where want and got differ:
// its path, then the value of want after a - and the value of got after a
// +, each of a side that has the place. Two values whose texts are equal
// state their types. Two strings of one type of which one has more than
// one line are a line diff instead.
func writeDifference(b *strings.Builder, d equality.Difference) {
	x, y := unwrapped(d.X), unwrapped(d.Y)
	path := pathText(d.Path)
	if multiline(x, y) {
		writeLines(b, path, x.String(), y.String())
		return
	}

	var xText, yText string
	if x.IsValid() {
		xText = valueText(x)
	}
	if y.IsValid() {
		yText = valueText(y)
	}
	if x.IsValid() && y.IsValid() && xText == yText {
		xText, yText = x.Type().String()+"("+xText+")", y.Type().String()+"("+yText+")"
	}

	b.WriteString("\t" + path)
	if path != "" {
		b.WriteString(": ")
	}
	if x.IsValid() {
		b.WriteString("-" + xText)
	}
	if x.IsValid() && y.IsValid() {
		b.WriteString(" ")
	}
	if y.IsValid() {
		b.WriteString("+" + yText)
	}
	b.WriteString("\n")
}

// pathText returns the text of a path: a field as .Name, an element as [2],
// and a key as its value's text in brackets, as ["gift"] or [7].
func pathText(path []equality.Step) string {
	var b strings.Builder
	for _, s := range path {
		if s.Field != "" {
			b.WriteString("." + s.Field)
			continue
		}
		if s.Key.IsValid() {
			b.WriteString("[" + valueText(unwrapped(s.Key)) + "]")
			continue
		}
		b.WriteString("[" + strconv.Itoa(s.Index) + "]")
	}
	return b.String()
}

// valueText returns the text of one value of a difference: a string
// quoted, and any other value as the package text writes it under %+v, so
// a value of an unexported field states no text of its methods.
func valueText(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return strconv.Quote(v.String())
	}
	return text.Sprintf("%+v", v)
}

// unwrapped returns the value inside v when v is an interface that is not
// nil, and v otherwise.
func unwrapped(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface && !v.IsNil() {
		return v.Elem()
	}
	return v
}

// multiline reports whether x and y are two strings of one type, of which
// one has more than one line.
func multiline(x, y reflect.Value) bool {
	return x.Kind() == reflect.String && y.Kind() == reflect.String && x.Type() == y.Type() &&
		(strings.Contains(x.String(), "\n") || strings.Contains(y.String(), "\n"))
}

// writeLines writes the line diff of the texts x and y at path: the path
// on a line of its own when it has a step, then each line that only x has
// after -, each line that only y has after +, and each unchanged line
// within three lines of a change after two spaces, with … in place of the
// unchanged lines between.
func writeLines(b *strings.Builder, path, x, y string) {
	if path != "" {
		b.WriteString("\t" + path + ":\n")
	}
	xs, ys := strings.Split(x, "\n"), strings.Split(y, "\n")
	edits := align.Edits(len(xs), len(ys), func(i, j int) bool { return xs[i] == ys[j] })
	near := make([]bool, len(edits))
	for k, e := range edits {
		if e.Op != align.Keep {
			for c := max(k-contextLines, 0); c <= min(k+contextLines, len(edits)-1); c++ {
				near[c] = true
			}
		}
	}
	skipped := false
	for k, e := range edits {
		if !near[k] {
			skipped = true
			continue
		}
		if skipped {
			b.WriteString("\t  " + gapMark + "\n")
			skipped = false
		}
		switch e.Op {
		case align.Delete:
			b.WriteString("\t- " + xs[e.X] + "\n")
		case align.Insert:
			b.WriteString("\t+ " + ys[e.Y] + "\n")
		default:
			b.WriteString("\t  " + xs[e.X] + "\n")
		}
	}
	if skipped {
		b.WriteString("\t  " + gapMark + "\n")
	}
}
