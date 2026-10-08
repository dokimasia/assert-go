// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package alphabet orders every Unicode scalar value from the simplest: the
// default alphabet of the string generator.
//
// A string is a sequence of indices into an alphabet, so the order of the
// alphabet decides what a string shrinks towards. The default alphabet
// starts with the 95 printable ASCII characters: the digits, the lowercase
// letters, the uppercase letters, then space and the punctuation. A string
// shrinks towards "0", then "00". The other code points of the Basic
// Multilingual Plane follow in ascending order without the surrogates, and
// the code points above the plane come last.
//
// # Panics
//
// [Rune] panics for an index of [Size] or more, as an index past the end
// of a slice does.
//
// # Concurrency
//
// Every function is safe for concurrent use. The package keeps no state.
//
// # Allocation contract
//
// [Rune], [Index] and [Merge] allocate nothing. [AppendIndices] grows the
// slice it is given, and allocates only when it lacks the capacity.
//
// # Dependency position
//
// Imports the standard library only. Depends on no other package in this
// module.
package alphabet
