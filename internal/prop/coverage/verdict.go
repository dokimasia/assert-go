// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package coverage

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Verdict -linecomment -output=verdict.string_gen.go

// Verdict is what one check decides about one requirement. Its String is
// the verdict's spelling in the definition.
type Verdict uint8

const (
	// Met is a requirement whose share the label covers.
	Met Verdict = 0 // met
	// Refuted is a requirement whose share is above the label's share with
	// the test's certainty.
	Refuted Verdict = 1 // refuted
	// Undecided is a requirement that a check before the last one neither
	// meets nor refutes.
	Undecided Verdict = 2 // undecided
	// Unmet is a requirement that the last check, or an exact share, does
	// not meet.
	Unmet Verdict = 3 // unmet
)

// Valid reports whether v is one of the four verdicts.
func (v Verdict) Valid() bool {
	return v <= Unmet
}
