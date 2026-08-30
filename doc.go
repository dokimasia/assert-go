// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package assert reports test failures that stop the test.
//
// An assertion does not call the test framework. It reports through a
// seat, so one assertion serves a real test, a benchmark and a
// recorder without knowing which it holds. [TB] is that seat, and
// [testing.T], [testing.B] and [Recorder] all satisfy it.
//
// # Failure semantics
//
//   - An assertion that passes reports nothing and returns.
//   - An assertion that fails reports through Fatalf, which stops the
//     test. Nothing after it in the same test body runs.
//   - Every assertion takes a message last, and that message is the
//     first line of the failure. It states the contract under test, so
//     a failure says what was supposed to be true rather than only
//     what was observed.
//
// # Testing an assertion
//
// [Recorder] is a seat that records a failure instead of stopping the
// test, which is what lets an assertion be tested by reading what it
// reported rather than suffering it. Drive an assertion with a
// Recorder in place of a [testing.T], then read [Recorder.Failed] and
// [Recorder.Msg].
//
// # Dependency position
//
// Imports fmt, runtime and sync. Depends on no other package in this
// module.
package assert

//go:generate go run ./internal/gen
