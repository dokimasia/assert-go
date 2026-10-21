// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package wrap finds a method of a seat along the chain of the seat. The
// chain is the seat, then the seat that its method Unwrap returns, then the
// seat that the Unwrap of that seat returns, and so on. The first seat of the
// chain that has the method provides it.
//
// The method Unwrap returns an assert.TB. No package that the root package
// imports can name that type, so the root package registers the function
// that calls Unwrap with [Register].
//
// # Dependency position
//
// Imports nothing. The root package, the matcher, the call records, prop and
// bench import it.
package wrap
