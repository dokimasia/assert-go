// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Relation -linecomment -output=relation.string_gen.go

// Relation is a dependency of one committed transaction on another, as a
// [Link] of a cycle and an [Edge] state it. The zero Relation is no
// relation.
type Relation uint8

const (
	// WW is a write-write dependency: the second transaction appended the
	// value that follows the first transaction's value in a key's version
	// order.
	WW Relation = 1 // ww
	// WR is a write-read dependency: the second transaction read a list that
	// ends in the first transaction's value.
	WR Relation = 2 // wr
	// RW is a read-write dependency: the second transaction appended the
	// value that follows the list that the first transaction read.
	RW Relation = 3 // rw
)

// Valid reports whether r is one of the three relations. It allocates
// nothing.
func (r Relation) Valid() bool {
	return r >= WW && r <= RW
}

// MarshalText returns the relation's spelling, as the record of a check
// states it.
//
// # Allocation contract
//
// MarshalText allocates the text it returns.
func (r Relation) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}
