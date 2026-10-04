// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package align pairs the equal elements of two sequences in order, so
// that a diff can state the elements that each sequence has alone: the
// elements of two slices in the differences of two values, and the lines
// of two texts in a line diff.
//
// # Dependency position
//
// Imports the standard library alone. Every other package of this module
// may import it.
package align
