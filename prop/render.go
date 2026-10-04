// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// absent are the words for a side of a divergence that states nothing, by
// the [Difference] it belongs to: the run ended, observed no fingerprint,
// or passed.
var absent = [...]string{
	RequestDifference:     "no request",
	FingerprintDifference: "no fingerprint",
	VerdictDifference:     "a pass",
}

// The text writer takes the sentence of the record of every property
// assertion from this package, which registers it while it initialises.
func init() {
	matcher.RegisterSentence(sentence, append([]string{forAllID}, formIDs[:]...)...)
}

// sentence returns the sentence of a failing run's record f, which the text
// writer sends to a seat without a Report method. It states the contract,
// the outcome with the counts and the seed, and then a line for each part of
// the detail that the outcome uses: the counterexample's draws, its failure
// and how to replay it, each other failure, the divergence and the coverage
// requirement.
func sentence(f assert.Failure) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %v after %d valid and %d rejected cases, seed %s",
		f.Contract, f.Detail[outcomeField], f.Detail[casesField], f.Detail[rejectedField], f.Detail[seedField])
	if drawn, ok := f.Detail[counterexampleField].([]Drawn); ok {
		writeDraws(&b, drawn)
	}
	if failure, ok := f.Detail[failureField].(assert.Failure); ok {
		writeFailure(&b, "failure", failure)
	}
	if choices, ok := f.Detail[choicesField].(string); ok {
		writeReplay(&b, choices)
	}
	others, _ := f.Detail[othersField].([]Other)
	for _, other := range others {
		writeFailure(&b, "other failure", other.Failure)
		writeDraws(&b, other.Counterexample)
		writeReplay(&b, other.Choices)
	}
	if d, ok := f.Detail[divergenceField].(*Divergence); ok {
		fmt.Fprintf(&b, "\ndivergence: the %v at %d, recorded %s, replayed %s",
			d.What, d.Index, sideText(d.What, d.Recorded), sideText(d.What, d.Replayed))
	}
	if s, ok := f.Detail[coverageField].(*Shortfall); ok {
		fmt.Fprintf(&b, "\ncoverage: %v, %q counted %d of %d valid cases against a required share of %v",
			s.Verdict, s.Label, s.Counted, s.Valid, s.Share)
	}
	return b.String()
}

// writeDraws writes a line for each draw: its label and its value as a Go
// literal, and what the explain phase found.
func writeDraws(b *strings.Builder, drawn []Drawn) {
	for _, d := range drawn {
		fmt.Fprintf(b, "\n  %s: %#v", d.Label, d.Value)
		if d.Relevance == AnyValueFails {
			b.WriteString(", any value fails")
		}
		if d.NearestPassing != nil {
			fmt.Fprintf(b, ", %#v passes", d.NearestPassing)
		}
	}
}

// writeFailure writes a line about a failed case's record: what, the
// record's assertion, the base name of its file and its line where the
// record has them, and the record's sentence.
func writeFailure(b *strings.Builder, what string, f assert.Failure) {
	b.WriteString("\n" + what)
	if f.Assertion != "" {
		fmt.Fprintf(b, " of %s", f.Assertion)
	}
	if f.Where != (assert.Where{}) {
		fmt.Fprintf(b, " at %s:%d", filepath.Base(f.Where.File), f.Where.Line)
	}
	fmt.Fprintf(b, ": %s", matcher.Render(f))
}

// writeReplay writes a line that states the two ways to replay the case
// that tok records.
func writeReplay(b *strings.Builder, tok string) {
	fmt.Fprintf(b, "\nreplay: prop.Replay(%q) or %s=%s", tok, replayVariable, tok)
}

// sideText returns one side of a divergence of what as text, and the words
// for its absence when it is nil.
func sideText(what Difference, side any) string {
	if side == nil {
		return absent[what]
	}
	return fmt.Sprint(side)
}
