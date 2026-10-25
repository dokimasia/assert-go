// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Outcome -linecomment -output=outcome.string_gen.go

// Outcome is how a run ended, the outcome field of a run's record. [Passed]
// passes the test, [Skipped] skips it, and every other outcome fails it.
// Each value converts from the engine's outcome of the same spelling.
type Outcome uint8

const (
	// Passed is a run whose every case passed and whose every coverage
	// requirement was met. It reports no record.
	Passed Outcome = 0 // passed
	// Counterexample is a run that found a failing case, and whose replay of
	// that case failed the same way.
	Counterexample Outcome = 1 // counterexample
	// Flaky is a run whose failing case passed or failed another way on
	// replay, or whose body requested other choices after the same values.
	Flaky Outcome = 2 // flaky
	// Rejected is a run that rejected more than ten cases for every valid
	// one.
	Rejected Outcome = 3 // rejected
	// CoverageUnmet is a run that refuted a coverage requirement or left it
	// unmet.
	CoverageUnmet Outcome = 4 // coverage-unmet
	// Vacuous is a run whose cases passed without requesting an input.
	Vacuous Outcome = 5 // vacuous
	// Skipped is a run that ended at a case that called [Case.Skipf] before
	// the run found a failing case. It does not report a record.
	Skipped Outcome = 6 // skipped
)

// Valid reports whether o is one of the seven outcomes.
func (o Outcome) Valid() bool {
	return o <= Skipped
}

// MarshalText returns the outcome's spelling, as the record of a run
// states it.
func (o Outcome) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}
