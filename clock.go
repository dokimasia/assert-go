// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"time"

	"go.dokimi.dev/assert/internal/matcher"
)

// Clock is where an assertion reads time.
//
// An assertion that waits, retries or measures reads the time here and
// not from the runtime, so a test can supply time that it controls, and a
// busy machine cannot make the assertion flaky.
type Clock = matcher.Clock

// Clocked is a [TB] that supplies a clock.
//
// A seat supplies a clock through this second interface, so [TB] keeps the
// three methods that [testing.T] and [testing.B] implement. An assertion
// reads [System] when its seat does not satisfy Clocked.
type Clocked = matcher.Clocked

// System reads the runtime clock. An assertion reads it when the seat
// supplies no clock.
type System = matcher.System

// Controlled is a clock that moves only when a test advances it.
//
// Now returns the instant that [Controlled.Advance] last moved it to. An
// assertion that retries advances this clock between attempts instead of
// sleeping against it, so a body that settles on the third attempt costs
// three attempts and no waiting.
//
// A test passes the clock to an assertion through [Recorder.WithClock].
// The subject does not read it: code under test that calls the runtime
// reads the runtime clock, and no assertion detects the difference.
type Controlled = matcher.Controlled

// NewControlled returns a [Controlled] that reads start until it is
// advanced.
//
// # Allocation contract
//
// NewControlled allocates twice: the clock and the condition that wakes
// its sleepers.
func NewControlled(start time.Time) *Controlled { return matcher.NewControlled(start) }
