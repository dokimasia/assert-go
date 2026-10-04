// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Kind -linecomment -output=kind.string_gen.go

// Kind is the kind of an event, and the completion kind of an [Interval].
// The zero Kind is no kind.
type Kind uint8

const (
	// Invoke is an invocation, by which a client starts a call. As the kind
	// of an interval, it states a pending call.
	Invoke Kind = 1 // invoke
	// OK is the completion of a call that returned its output and took
	// effect.
	OK Kind = 2 // ok
	// Fail is the completion of a call that returned an error, took no
	// effect, and returned nothing that a model checks.
	Fail Kind = 3 // fail
	// Unknown is the completion of a call that ended without an outcome,
	// such as a timeout, a lost reply or a crash.
	Unknown Kind = 4 // unknown
)

// Valid reports whether k is one of the four kinds. It allocates nothing.
func (k Kind) Valid() bool {
	return k >= Invoke && k <= Unknown
}

// MarshalText returns the kind's spelling, as the history's JSON form states
// it.
//
// # Allocation contract
//
// MarshalText allocates the text it returns.
func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}
