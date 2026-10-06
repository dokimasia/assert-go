// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Verdict -linecomment -output=verdict.string_gen.go

// Verdict is how a check of [Linearizable] ended, the outcome field of the
// record of a failing check. Every verdict but [Passed] fails the test.
type Verdict uint8

const (
	// Passed is a check whose every partition has a linearization that the
	// spec accepts. It reports no record.
	Passed Verdict = 0 // passed
	// Violated is a check whose search of a partition tried every order within
	// its limits and found no linearization that the spec accepts.
	Violated Verdict = 1 // violated
	// Undecided is a check whose search of a partition used up a limit, and
	// none of whose partitions is violated.
	Undecided Verdict = 2 // undecided
)

// MarshalText returns the verdict's spelling, as the record of a check
// states it.
func (v Verdict) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}
