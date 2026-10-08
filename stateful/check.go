// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
)

// contract is the contract of the check after a step.
const contract = "the history of the machine's steps is linearizable"

// seat is the seat of the check after a step: the case, which receives
// every report of the check but its record, and the record of a check that
// fails. It keeps no call record, so the passing checks of a run leave
// nothing in a recorded run.
type seat struct {
	assert.TB
	// failure is the record of the check, when it failed.
	failure *assert.Failure
}

// Report keeps the record of a check that fails.
func (s *seat) Report(f assert.Failure, _ bool) {
	s.failure = &f
}

// check checks the case's history against the machine's spec, with every
// call in one partition, and keeps the states that the linearization the
// check found leaves. A check that fails ends the case with its record. A
// machine without a spec checks nothing.
func (r *run[S]) check() {
	if !r.m.specified() {
		return
	}
	history.Linearizable(&r.seat, r.c.History(), r.m.Spec, contract, r.checks...)
	if f := r.seat.failure; f != nil {
		r.c.Report(*f, true)
	}
}
