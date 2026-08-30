// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package matcher compares values and reports the difference.
//
// Each assertion is one function taking a [Seat] and a [Mode]. The
// mode decides whether a failure stops the test, so the aborting and
// recording surfaces share one comparison and cannot disagree.
//
// # Comparison rules
//
//   - A nil collection does not equal an empty one. [EquateEmpty] reverses this.
//   - NaN does not equal NaN. [EquateNaNs] reverses this.
//   - Unexported fields take part.
//   - Two references to one function are equal; two functions are not.
//
// # Dependency position
//
// Imports github.com/google/go-cmp/cmp, its cmpopts subpackage, and
// reflect. Depends on no other package in this module.
package matcher
