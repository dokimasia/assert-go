// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Limit -linecomment -output=limit.string_gen.go

// Limit is the limit that stopped an undecided search, the limit field of
// the record of an [Undecided] check. The zero Limit is no limit.
type Limit uint8

const (
	// LimitSteps is the budget of steps that [Budget] sets.
	LimitSteps Limit = 1 // steps
	// LimitMemo is the limit on the memo that [MemoLimit] sets.
	LimitMemo Limit = 2 // memo
	// LimitTime is the limit on the time of the whole check that [TimeLimit]
	// sets.
	LimitTime Limit = 3 // time
)

// MarshalText returns the limit's spelling, as the record of a check states
// it.
func (l Limit) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}
