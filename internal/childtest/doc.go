// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package childtest runs a test of a test binary in a child process of that
// binary, for a check that needs a process of its own: a check of a value
// that the module reads once per process, of the output that a test writes,
// or of a panic that ends the process.
//
// # Dependency position
//
// Imports the standard library, testing included, and internal/record for
// the name of the switch that a child's environment states.
package childtest
