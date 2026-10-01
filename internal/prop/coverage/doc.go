// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package coverage decides whether a label covers a required share of the
// valid cases of a run.
//
// A requirement is decided with the Wilson score interval of the label's
// share, with the constants of QuickCheck's checkCoverage: a certainty of
// 10^9 and a tolerance of [Tolerance]. The runner checks every requirement
// after each multiple of the cases setting that [Checks] returns. The test
// uses arithmetic and one square root, which IEEE 754 rounds exactly, so
// every implementation returns the same verdict from the same counts when
// it evaluates [Bound] in the order the definition writes it.
//
// # Panics
//
// [Bound] and [Decide] panic for counts that are no share: no trials, or
// successes outside [0, n].
//
// # Concurrency
//
// Every function is safe for concurrent use. The package keeps no state.
//
// # Allocation contract
//
// No function allocates.
//
// # Dependency position
//
// Imports the standard library only. Depends on no other package in this
// module.
package coverage
