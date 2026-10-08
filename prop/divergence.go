// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"path/filepath"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Difference -linecomment -output=divergence.string_gen.go

// The words of a failure identity as text.
const (
	// panicWords lead the identity of a panic, before the type of its value.
	panicWords = "panic of "
	// messageWords are the identity of a message that the body passed to
	// [Case.Fatalf] or [Case.Errorf].
	messageWords = "message"
)

// Difference is what differed first in a flaky run. Each value converts
// from the engine's difference of the same spelling.
type Difference uint8

const (
	// RequestDifference is a request with other bounds than the recorded
	// request at the same position, or an end where the recorded run made a
	// request.
	RequestDifference Difference = 0 // request
	// FingerprintDifference is another fingerprint than the recorded one at
	// the same position, or another number of fingerprints.
	FingerprintDifference Difference = 1 // fingerprint
	// VerdictDifference is a replay of a failing case that passed or failed
	// another way.
	VerdictDifference Difference = 2 // verdict
)

// Valid reports whether d is one of the three differences.
func (d Difference) Valid() bool {
	return d <= VerdictDifference
}

// MarshalText returns the difference's spelling, as the record of a run
// states it.
func (d Difference) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// Divergence is what differed first in a flaky run, with the recorded and
// the replayed side of the difference. Each side has the form that What
// states:
//
//   - A request states its bounds as text: "integer in [0, 9]", "float in
//     [0, 1] of width 64", the same with " or NaN" when NaN is a value, and
//     "sequence of 0 to 8 values below 256" or "sequence of 2 or more values
//     below 256".
//   - A fingerprint is the uint64 that [Case.Observe] recorded.
//   - A verdict states the failure's identity as text: the assertion and
//     its location, such as "equal at codec_test.go:18", the assertion and
//     its contract for a record without a location, "message at
//     codec_test.go:18", or "panic of *errors.errorString at
//     codec_test.go:18".
//
// A side is nil where its run ended, observed no fingerprint, or passed.
//
// Label and Step state where the replayed run made the request or
// observed the fingerprint, so a reader finds the code that read what
// differed: the draw that ran, and the part and the step of a machine. A
// request of a sequential step's index whose bounds differ, for example,
// lists other actions than the recorded run listed, because an Enabled of
// the machine read a result that the subject returned differently.
type Divergence struct {
	// What is what differed.
	What Difference
	// Index is the position of the request or the fingerprint that
	// differs. For a verdict it is the number of choices that the recorded
	// case made, the position where it ended.
	Index int
	// Recorded is the recorded run's side of the difference.
	Recorded any
	// Replayed is the replayed run's side of the difference.
	Replayed any
	// Label is the label of the draw that ran where the replayed run made
	// the request or observed the fingerprint. It is nil where no draw ran,
	// where the replayed run made no request or observed no fingerprint at
	// the position, and for a verdict.
	Label *string
	// Step is the part and the step of a machine that ran there. It is nil
	// outside a machine's steps, where the replayed run made no request or
	// observed no fingerprint at the position, and for a verdict.
	Step *Place
}

// divergenceOf returns the engine's divergence d with each side in the
// form that [Divergence] states.
func divergenceOf(d engine.Divergence) *Divergence {
	out := &Divergence{
		What:     Difference(d.What),
		Index:    d.Index,
		Recorded: versionOf(d.Recorded),
		Replayed: versionOf(d.Replayed),
		Step:     placeOf(d.Where),
	}
	if d.Where.Drawing {
		out.Label = new(d.Where.Label)
	}
	return out
}

// versionOf returns one side of an engine's divergence in the form that
// [Divergence] states: bounds and an identity as text, and a fingerprint
// or nil as it is.
func versionOf(v any) any {
	switch v := v.(type) {
	case choice.Bounds:
		return v.String()
	case engine.Identity:
		return identityText(v)
	}
	return v
}

// identityText returns the text of a failure identity: what failed, then
// its location as the base name of its file and its line, or the contract
// of an assertion's record without a location.
func identityText(i engine.Identity) string {
	what := i.Assertion
	if i.Panic != "" {
		what = panicWords + i.Panic
	} else if what == "" {
		what = messageWords
	}
	if i.Where != (assert.Where{}) {
		return fmt.Sprintf("%s at %s:%d", what, filepath.Base(i.Where.File), i.Where.Line)
	}
	if i.Contract != "" {
		return fmt.Sprintf("%s (%s)", what, i.Contract)
	}
	return what
}
