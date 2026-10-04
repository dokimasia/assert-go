// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package enumtest checks the enumerations of this module: a defined
// integer type whose constants are its members, with a Valid method that
// tells a member from any other value, and the String method that stringer
// generates.
//
// The checks report through an [assert.TB], so a test of a check drives it
// with an [assert.Recorder].
//
// # Dependency position
//
// Imports assert, which the checks report through. Only the test files of
// this module import it.
package enumtest
