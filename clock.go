// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"time"

	"go.dokimi.dev/assert/internal/matcher"
)

// Clock is where an assertion reads time.
//
// An assertion that waits, retries or measures reads it here rather
// than calling the runtime, so a test can supply time it controls and
// a busy machine cannot make the assertion flaky.
type Clock = matcher.Clock

// Clocked is a [TB] that carries a clock.
//
// [testing.TB] declares three methods and can never grow a fourth, so
// a clock reaches an assertion through a second interface a seat may
// also satisfy. A seat that does not satisfy it reads the runtime
// clock, which is what every assertion did before this existed.
type Clocked = matcher.Clocked

// System reads the runtime clock, and is what an assertion gets when
// the seat carries none.
type System = matcher.System

// Controlled is a clock that moves only when a test advances it.
//
// Now answers what [Controlled.Advance] last left it at, and an
// assertion that retries advances it rather than waiting, so a body
// that settles on the third attempt costs three attempts and no real
// time.
//
// It reaches an assertion through [Recorder.WithClock]. A controlled
// clock cannot reach the subject: code under test that calls the
// runtime directly reads a different now, and nothing here detects
// that.
type Controlled = matcher.Controlled

// NewControlled returns a [Controlled] reading start until it is
// advanced.
func NewControlled(start time.Time) *Controlled { return matcher.NewControlled(start) }
