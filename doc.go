// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package assert reports test failures that stop the test.
//
// An assertion does not call the test framework. It reports through a
// seat, so one assertion works on a test, a benchmark and a recorder
// alike. [TB] is that seat, and [testing.T], [testing.B] and [Recorder]
// all satisfy it.
//
//	assert.Equal(t, store.Get(ctx, id), item, "Get returns the stored item")
//
// # Failure semantics
//
//   - An assertion that passes reports nothing to the seat and returns.
//   - An assertion that fails reports through Fatalf, which stops the
//     test. Nothing after it in the same test body runs.
//   - Every assertion takes a message last, and that message is the
//     first line of the failure. It states the contract under test, so
//     a failure states what was supposed to be true as well as what was
//     observed.
//
// # Call records
//
// With DOKIMI_ASSERT_RECORD set to 1, every assertion call on a test
// writes a call record through the test's Attr, which go test -json
// reports as an attr event of the test under the key dokimi.assert.<seq>.
// Recording is off for an unset or empty variable and for 0. Any other
// value makes every assertion end the test with a fault. A [Recorder]
// keeps the call records of its calls whatever the variable states.
//
// # Equality
//
//   - Comparison is structural and compares every field, unexported
//     ones included.
//   - A nil map or slice does not equal an empty one. Pass
//     [EquateEmpty] where that difference does not matter.
//   - NaN does not equal NaN. Pass [EquateNaNs] to reverse that.
//   - Floats compare exactly.
//   - Two references to one function are equal, and two different
//     functions are not.
//   - Values of different types never compare. The assertions take a
//     type parameter, so a mismatch is a compile error.
//
// An [Option] applies to the call it is passed to and to nothing else.
//
// # Testing an assertion
//
// [Recorder] is a seat that records a failure instead of stopping the
// test, so a test can read what an assertion reported. Drive an
// assertion with a Recorder in place of a [testing.T], then read
// [Recorder.Failed], [Recorder.Message] and [Recorder.Records].
//
// # Allocation contracts
//
// The allocation contract of each function and method counts one passing
// call on a seat that writes no call record, such as a test's seat while
// recording is off, and leaves out what the caller's functions allocate. A
// failing call allocates its record and its text as well, and a recorded
// call allocates its call record. A comparison goes through go-cmp, whose
// caches can lower the count of a later call.
//
// A count also leaves out the interface that the call site builds for an
// argument of type any. An assertion that reports the argument in its
// failure, such as [CloseTo] for got, lets the argument escape. Its call
// site then allocates the interface of a computed value once. For a
// constant, the call site allocates nothing. A chain method builds that
// interface itself, so its count includes it.
//
// # Dependency position
//
// Imports internal/matcher, internal/record, and the standard library's
// cmp, context, fmt, runtime, sync and time.
package assert
