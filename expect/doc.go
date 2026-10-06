// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package expect records test failures and lets the test continue.
//
// It declares every member of [go.dokimi.dev/assert] under the same name,
// and compares values the same way: each assertion, [Rejects], and the
// seat, the record and the clocks as aliases of the types of that package.
// It differs in what happens on a failure: this package reports through
// Errorf, so the test runs on and later assertions report too.
//
// Use it where several properties of one value are worth seeing at once.
// A chain here runs every method, so one run reports every property that
// fails:
//
//	expect.That(t, user).
//	    NotNil("the user was found").
//	    HasPrefix("usr_", "the id starts with its prefix").
//	    Length(3, "every field was populated")
//
// # Parity with the aborting surface
//
// Every function and chain method here calls the same matcher-core
// function as its counterpart in [go.dokimi.dev/assert], in the
// recording mode where that one uses the aborting mode, so the two
// compare values identically. A conformance test fails the build when
// one surface declares a member that the other does not, so this package
// declares at least what that one declares.
//
// # Allocation contracts
//
// The allocation contract of each function and method counts one passing
// call on a seat that writes no call record, such as a test's seat while
// recording is off, and leaves out what the caller's functions allocate. A
// failing call allocates its record and its text as well, and a recorded
// call allocates its call record.
//
// An assertion that compares values builds the interface of each value
// that it compares. Go builds the interface of a pointer and of an integer
// below 256 without allocating, and allocates one for most other values. A
// contract states the count of the values that it names.
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
// Imports go.dokimi.dev/assert for the seat and the types that this
// package aliases, and internal/matcher for the comparisons.
package expect
