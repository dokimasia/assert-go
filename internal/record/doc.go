// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package record writes the call record of every assertion call: its
// number in its test, the assertion, the contract, the verdict, the
// surface, the call site, and the detail of a failure.
//
// A [Call] is a record before its number. [Calls] keep the records of one
// test, one recorder or one run of a body, and number them:
//
//   - The Calls of a test write each record through the test's Attr while
//     [On] reports true.
//   - The Calls of a recorder keep each record, whatever the switch states,
//     and [Lines] returns them.
//   - The Calls of one run of a body keep its records unnumbered, until the
//     call that ran the body takes them through [Slot.Take].
//
// A call that runs a body takes its number through [Begin], before the
// calls of its body, and writes its record through [Slot.Write] when it
// ends.
//
// # Dependency position
//
// Imports the fault package and the standard library. The matcher, the
// root package and the engine of the property engine import it.
package record
