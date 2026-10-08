// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package literal reads and writes the definition's typed literals: the
// language-neutral form in which the corpus, the vectors, a store's entries
// and a shape file state a value.
//
// [Decode] turns a literal into a Go value, and [Encode] turns a Go value
// into a literal. A record, a map that keeps its entries' order and a
// variant have no Go type that states them, so this package declares
// [Record], [Pairs] and [Variant] for them. [Detail] writes a value of a
// call record's detail, and [Opaque] the literal of a value that no other
// literal states.
//
// # Dependency position
//
// Imports fault, text for the text of an opaque value, and the standard
// library. The packages of the property engine, prop and conformance import
// it.
package literal
