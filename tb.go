// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

// TB is the seat that an assertion reports through.
// [testing.T], [testing.B] and [Recorder] satisfy it.
//
// [testing.TB] declares an unexported method, so no type outside the
// standard library implements it. This package declares TB so that any
// type with these three methods is a seat, such as the seat of a
// generated check body.
type TB interface {
	// Helper marks the calling function as a test helper, so a
	// failure reports its caller's line.
	Helper()
	// Fatalf records a failure and stops the test.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}
