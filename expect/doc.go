// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package expect records test failures and lets the test continue.
//
// It carries the same members as [go.dokimi.dev/assert] under the same
// names, and compares values the same way. The difference is only what
// happens on failure: this package reports through Errorf, so the test
// runs on and later assertions report too.
//
// Reach for it where several properties of one value are worth seeing
// at once. A chain here runs every method, so one run tells you
// everything that is wrong rather than the first thing:
//
//	expect.That(t, user).
//	    NotNil("the user was found").
//	    HasPrefix("usr_", "the id carries its prefix").
//	    Length(3, "every field was populated")
//
// # What is generated
//
// The functions and chain methods come from the matcher core, so a
// member here cannot drift from its counterpart in the aborting
// surface. Edit the core and run go generate; do not edit
// expect.gen.go.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert for the seat, and internal/matcher for
// the comparisons.
package expect
