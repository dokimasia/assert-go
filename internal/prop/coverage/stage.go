// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package coverage

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Stage -linecomment -output=stage.string_gen.go

// Stage is the point of a run at which a check decides a requirement. It
// selects the rule of [Decide].
type Stage uint8

const (
	// Interim is a check before the last one of a run. It can leave a
	// requirement undecided.
	Interim Stage = 0 // interim
	// Final is the last check of a run, which decides every requirement.
	Final Stage = 1 // final
	// Exhausted is a check of a run that tested every input of its domain,
	// so its shares are exact.
	Exhausted Stage = 2 // exhausted
)

// Valid reports whether s is one of the three stages.
func (s Stage) Valid() bool {
	return s <= Exhausted
}
