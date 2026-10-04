// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package matcher compares values and reports the difference.
//
// Each assertion is one function taking a [Seat] and a [Mode]. The
// mode decides whether a failure stops the test, so the aborting and
// recording surfaces call one comparison and report the same verdict.
//
// Every call reports its verdict through [Pass], [Fail] or [Fault], which
// write the call's record when its seat records calls. A call that runs a
// body, such as [Eventually] or a property, starts with [Begin] and reports
// through the [Running] it returns. The [Writer] turns each failure and
// fault into the text that a seat receives, with the sentence that a package
// registers through [RegisterSentence] for its assertions. A [Reporter]
// takes a failure's record instead of its text, and a [FaultReporter] takes
// a fault. [Note] writes a note into the log of a seat, and [NoteFault]
// notes a fault that does not end its call.
//
// # Comparison rules
//
//   - A nil collection does not equal an empty one. [EquateEmpty] reverses this.
//   - NaN does not equal NaN. [EquateNaNs] reverses this.
//   - Negative zero equals positive zero, as IEEE 754 states.
//   - Unexported fields take part.
//   - Two references to one function are equal, and two different
//     functions are not.
//
// # Allocation counts
//
// [AllocationsCounted] reports false in a build with the race detector,
// msan or asan, and in a build whose -gcflags turn off optimisation or
// inlining. Neither build allocates as an ordinary one does, and
// [MaxAllocs] checks no ceiling in either.
//
// # Allocation contracts
//
// The allocation contract of each function counts one call on a seat that
// writes no call record, such as a test's seat while recording is off, and
// leaves out what the caller's functions allocate. A failing call
// allocates its record and its text as well, and a recorded call
// allocates its call record.
//
// A count also leaves out the interface that the call site builds for an
// argument of type any. A function that reports the argument in its
// failure, such as [CloseTo] for got, lets the argument escape. Its call
// site then allocates the interface of a computed value once. For a
// constant, the call site allocates nothing.
//
// # Dependency position
//
// Imports github.com/google/go-cmp/cmp, its cmpopts subpackage, the
// standard library, testing included, and five packages of this module:
// internal/fault for the faults it reports, internal/literal for the typed
// literals of a recorded failure's detail, internal/record for the call
// records, internal/text for the text of a detail's values, and
// internal/prop/pattern, whose parser decides which patterns [Matches]
// accepts.
package matcher
