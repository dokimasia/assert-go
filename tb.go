// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

// TB is the failure surface an assertion reports through.
// [testing.T], [testing.B] and [Recorder] satisfy it.
//
// [testing.TB] carries an unexported method, so nothing outside the
// standard library implements it. This interface is declared here so a
// generated check body can hold one seat and satisfy it.
type TB interface {
	// Helper marks the calling function as a test helper, so a
	// failure reports its caller's line.
	Helper()
	// Fatalf records a failure and stops the test.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}
