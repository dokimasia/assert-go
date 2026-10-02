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
//   - Negative zero equals positive zero, as IEEE 754 states.
//   - Unexported fields take part.
//   - Two references to one function are equal; two functions are not.
//
// # Allocation counts
//
// [AllocationsCounted] reports false in a build with the race detector,
// msan or asan, and in a build whose -gcflags turn off optimisation or
// inlining. Neither build allocates as an ordinary one does, and
// [MaxAllocs] checks no ceiling in either.
//
// # Dependency position
//
// Imports github.com/google/go-cmp/cmp, its cmpopts subpackage, the
// standard library, testing included, and one package of this module:
// internal/prop/pattern, whose parser decides which patterns [Matches]
// accepts.
package matcher
