// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package assert reports test failures that stop the test.
//
// Every assertion takes a seat implementing [TB] first and a message
// last. The message states the contract under test and appears as the
// first line of a failure.
//
// # Call styles
//
//   - Function: assert.Equal(t, got, want, "Get returns the stored item")
//   - Chain:    assert.That(t, got).Equal(want, "Get returns the stored item")
//
// Both carry the same members under the same names.
//
// # Failure semantics
//
// An assertion here calls Fatalf, which stops the test at the first
// failure. [go.dokimi.dev/assert/expect] carries the same members
// reporting through Errorf, which records the failure and returns.
//
// # Equality
//
//   - A nil collection does not equal an empty one. [EquateEmpty] reverses this.
//   - Values of different types do not compare; the type parameter refuses them.
//   - NaN does not equal NaN. [EquateNaNs] reverses this.
//   - Floats compare exactly.
//   - Unexported fields take part in the comparison.
//   - Two references to one function are equal.
//
// # Dependency position
//
// Imports internal/matcher, embed, fmt, io/fs, runtime, strings and
// sync. Depends on no other package in this module.
package assert

//go:generate go run ./internal/gen
