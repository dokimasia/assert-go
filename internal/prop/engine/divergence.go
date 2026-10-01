// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import "go.dokimi.dev/assert/internal/prop/tree"

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Difference -linecomment -output=divergence.string_gen.go

// Difference is what differed between two runs of the same choices.
type Difference uint8

const (
	// RequestDifference is a request with other bounds. Recorded and
	// Replayed are the [choice.Bounds] of the two requests, or nil for a
	// run that ended at that position.
	RequestDifference Difference = 0 // request
	// FingerprintDifference is another observed fingerprint. Recorded and
	// Replayed are the two fingerprints, or nil for a run that observed
	// none at that position.
	FingerprintDifference Difference = 1 // fingerprint
	// VerdictDifference is a replay that passed or failed another way.
	// Recorded and Replayed are the two failures' [Identity] values, or nil
	// for a run that passed.
	VerdictDifference Difference = 2 // verdict
)

// Valid reports whether d is one of the three differences.
func (d Difference) Valid() bool {
	return d <= VerdictDifference
}

// Divergence is the first difference between two runs of the same
// choices, which makes a run flaky.
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

// requestDivergence returns the tree's report of a divergence as a request
// difference.
func requestDivergence(d *tree.DivergenceError) *Divergence {
	divergence := &Divergence{What: RequestDifference, Index: d.Index}
	if d.Recorded != nil {
		divergence.Recorded = *d.Recorded
	}
	if d.Requested != nil {
		divergence.Replayed = *d.Requested
	}
	return divergence
}
