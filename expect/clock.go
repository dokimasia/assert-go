// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"time"

	"go.dokimi.dev/assert"
)

// Clock is where an assertion reads time, the clock of
// [go.dokimi.dev/assert].
type Clock = assert.Clock

// Clocked is a [TB] that supplies a clock.
type Clocked = assert.Clocked

// System reads the runtime clock. An assertion reads it when the seat
// supplies no clock.
type System = assert.System

// Controlled is a clock that moves only when a test advances it. A test
// passes it to an assertion through [Recorder.WithClock].
type Controlled = assert.Controlled

// NewControlled returns a [Controlled] that reads start until it is
// advanced.
//
// # Allocation contract
//
// NewControlled allocates twice: the clock and the condition that wakes
// its sleepers.
func NewControlled(start time.Time) *Controlled { return assert.NewControlled(start) }
