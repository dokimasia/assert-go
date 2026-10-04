// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package text formats the values that this module writes for a person, as
// fmt formats them, without following a value that contains itself.
//
// fmt walks the parts of a map, a slice, an array, a struct and an
// interface, and the target of a pointer at the top level. It follows a map
// or a slice that contains itself until the stack runs out, which ends the
// program. [Sprintf] walks each argument first, and passes it to fmt only
// when the walk ends within 65,536 values without meeting a map or a slice
// inside itself. Every other argument becomes its structural text.
//
// # Dependency position
//
// Imports the standard library alone. Every other package of this module
// may import it.
package text
