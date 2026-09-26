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
// # Parity with the aborting surface
//
// Every function and chain method here calls the same matcher-core
// function as its counterpart in [go.dokimi.dev/assert], in the
// recording mode where that one uses the aborting mode, so the two
// compare values identically. A conformance test fails the build when
// either surface carries a member the other does not.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert for the seat, and internal/matcher for
// the comparisons.
package expect
