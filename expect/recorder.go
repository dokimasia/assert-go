// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import "go.dokimi.dev/assert"

// Recorder is a [TB] that records a failure instead of stopping the test,
// the recorder of [go.dokimi.dev/assert]. A test of an assertion reads from
// it what the assertion reported.
type Recorder = assert.Recorder

// NewRecorder returns a Recorder that records failures and returns from
// Fatalf.
//
// # Allocation contract
//
// NewRecorder allocates once: the Recorder.
func NewRecorder() *Recorder { return assert.NewRecorder() }
