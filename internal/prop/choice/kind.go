// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Kind -linecomment -output=kind.string_gen.go

// Kind is the kind of a choice. Kinds order the choices of different
// kinds at one position of a case: an integer is simpler than a float,
// and a float is simpler than a sequence. [Key.Compare] compares the kinds
// of two keys before anything else.
type Kind uint8

const (
	// Integer is a choice of one integer within [IntegerBounds].
	Integer Kind = 0 // integer
	// Float is a choice of one float of a width within [FloatBounds].
	Float Kind = 1 // float
	// Sequence is a choice of a sequence of integers in [0, k) within
	// [SequenceBounds].
	Sequence Kind = 2 // sequence
)

// Valid reports whether k is one of the three kinds.
func (k Kind) Valid() bool {
	return k <= Sequence
}
