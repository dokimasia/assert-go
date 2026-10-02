// Copyright ThesmOS B.V. 2026
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
}

// divergenceOf returns the engine's divergence d with each side in the
// form that [Divergence] states.
func divergenceOf(d engine.Divergence) *Divergence {
	return &Divergence{
		What:     Difference(d.What),
		Index:    d.Index,
		Recorded: versionOf(d.Recorded),
		Replayed: versionOf(d.Replayed),
	}
}

// versionOf returns one side of an engine's divergence in the form that
// [Divergence] states: bounds and an identity as text, and a fingerprint
// or nil as it is.
func versionOf(v any) any {
	switch v := v.(type) {
	case choice.Bounds:
		return boundsText(v)
	case engine.Identity:
		return identityText(v)
	}
	return v
}

// boundsText returns the text of a request's bounds.
func boundsText(b choice.Bounds) string {
	switch b.Kind() {
	case choice.Integer:
		return fmt.Sprintf("integer in [%s, %s]", b.Integer().Lo(), b.Integer().Hi())
	case choice.Float:
		f := b.Float()
		if f.NaNPolicy() == choice.AdmitNaN {
			return fmt.Sprintf("float in [%v, %v] of width %d or NaN", f.Lo(), f.Hi(), f.Width())
		}
		return fmt.Sprintf("float in [%v, %v] of width %d", f.Lo(), f.Hi(), f.Width())
	}
	s := b.Sequence()
	if most, bounded := s.Sizes().Max(); bounded {
		return fmt.Sprintf("sequence of %d to %d values below %d", s.Sizes().Min(), most, s.K())
	}
	return fmt.Sprintf("sequence of %d or more values below %d", s.Sizes().Min(), s.K())
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
