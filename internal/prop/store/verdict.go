// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Verdict -linecomment -output=verdict.string_gen.go

// Verdict is what a run does with one file of its store.
type Verdict uint8

const (
	// Replay is an entry of [Format] for the property, whose choices the
	// run tries first.
	Replay Verdict = 0 // replay
	// Other is an entry of [Format] for another property of the test,
	// which the run leaves as it is.
	Other Verdict = 1 // other
	// Skip is an entry of a later format, or whose token is of a later
	// version, which the run notes and passes over.
	Skip Verdict = 2 // skip
	// Damaged is a file that is no entry. It fails the test.
	Damaged Verdict = 3 // damaged
)

// Valid reports whether v is one of the four verdicts.
func (v Verdict) Valid() bool {
	return v <= Damaged
}
