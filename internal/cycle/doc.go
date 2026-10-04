// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package cycle tracks the pointers, maps and slices that enclose the
// position of a walk over a value, so that a walk meets a container inside
// itself as a cycle instead of following it until the stack runs out.
//
// The text that this module writes for a person and the canonical key of a
// generated value both walk values that can contain themselves, and both
// keep their path in a [Path].
//
// # Dependency position
//
// Imports the standard library alone. Every other package of this module
// may import it.
package cycle
