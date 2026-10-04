// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package fault states the errors of this module: what failed, where in
// its input, and why.
//
// A fault, an [Error], is a value first and text second. Its fields name
// the operation, the path in the input, the kind that errors.Is matches,
// the reason and the cause, and [Error.Error] renders them for a person. A
// fault is built where its condition is found. It gains the segments of
// its [Path] through [At] as it returns through the code that walked the
// input, and its operation through [In] at the exported function.
//
// # Dependency position
//
// Imports the standard library alone. Every other package of this module
// may import it.
package fault
