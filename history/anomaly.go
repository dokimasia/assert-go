// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Anomaly -linecomment -output=anomaly.string_gen.go

// Anomaly is a kind of anomaly that [Serializable] and
// [HasSnapshotIsolation] find in a history of list-append transactions, the
// anomaly field of the record of a failing check. The kinds are in the order
// in which a record reports them, and the zero Anomaly is no kind.
type Anomaly uint8

const (
	// GarbageRead is a committed read of a value that no transaction
	// appended to the key.
	GarbageRead Anomaly = 1 // garbage-read
	// DuplicateAppend is a committed read that returned one value twice.
	DuplicateAppend Anomaly = 2 // duplicate-append
	// InternalInconsistency is a committed read that contradicts its own
	// transaction: it lacks the transaction's earlier appends to the key,
	// differs from what the transaction read of the key before, or contains
	// a value that the transaction appends to the key later.
	InternalInconsistency Anomaly = 3 // internal-inconsistency
	// IncompatibleOrder is two committed reads of one key of which neither
	// is a prefix of the other.
	IncompatibleOrder Anomaly = 4 // incompatible-order
	// AbortedRead is a committed read of a value that an aborted transaction
	// appended, Adya's G1a.
	AbortedRead Anomaly = 5 // aborted-read
	// IntermediateRead is a committed read of a list that ends in a value
	// that another transaction followed with an append of its own to the
	// same key, Adya's G1b.
	IntermediateRead Anomaly = 6 // intermediate-read
	// G0 is a cycle of write-write dependencies.
	G0 Anomaly = 7 // G0
	// G1c is a cycle of write-write and write-read dependencies with one
	// write-read dependency or more.
	G1c Anomaly = 8 // G1c
	// GSingle is a cycle with exactly one read-write dependency.
	GSingle Anomaly = 9 // G-single
	// GNonadjacent is a cycle with two read-write dependencies or more, no
	// two of them adjacent.
	GNonadjacent Anomaly = 10 // G-nonadjacent
	// G2 is a cycle with one read-write dependency or more.
	G2 Anomaly = 11 // G2
)

// Valid reports whether a is one of the eleven kinds. It allocates nothing.
func (a Anomaly) Valid() bool {
	return a >= GarbageRead && a <= G2
}

// MarshalText returns the kind's spelling, as the record of a check states
// it.
//
// # Allocation contract
//
// MarshalText allocates the text it returns.
func (a Anomaly) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}
