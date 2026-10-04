// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"encoding/json"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/record"
)

// clockEpoch is the instant a controlled clock starts at, chosen so a
// reading cannot pass by accident against a real clock.
var clockEpoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// allocRuns is the number of calls whose allocations an allocation case
// averages, as matcher.MaxAllocs counts them.
const allocRuns = 100

// allocContract is the contract of every call of the allocation cases.
const allocContract = "the call passes"

// allocCase is one call of a function of the package, and the ceiling of
// its allocations.
type allocCase struct {
	// name is the name of the function, which names the case's
	// sub-benchmark.
	name string
	// call calls the function on seat, which writes no call record.
	call func(seat matcher.Seat)
	// allocs is the ceiling: the allocations of one call, measured.
	allocs float64
	// fails reports whether the call reports a failure or a fault, as every
	// call of Fail does.
	fails bool
}

// checkAllocs checks the ceiling of each case with testing.AllocsPerRun in
// a build that counts allocations, and that each call reports a failure
// exactly when its case states one. It calls each case on a seat of
// internal/matchertest of its own, and reports every case that misses.
//
// testing.AllocsPerRun panics while a parallel test runs, so the test that
// calls it does not call t.Parallel.
func checkAllocs(t *testing.T, cases []allocCase) {
	t.Helper()
	for _, c := range cases {
		seat := &matchertest.Seat{}
		got := testing.AllocsPerRun(allocRuns, func() { c.call(seat) })
		if matcher.AllocationsCounted() && got > c.allocs {
			t.Errorf("%s allocates %v times per call, want at most %v", c.name, got, c.allocs)
		}
		if seat.Failed() != c.fails {
			t.Errorf("%s reported %q, and its case states a failure: %v", c.name, seat.First(), c.fails)
		}
	}
}

// benchAllocs measures each case in a sub-benchmark of its name, and
// reports its allocations.
func benchAllocs(b *testing.B, cases []allocCase) {
	b.Helper()
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			seat := &matchertest.Seat{}
			b.ReportAllocs()
			for b.Loop() {
				c.call(seat)
			}
		})
	}
}

// clockedSeat is a seat with a clock, which records what was reported.
type clockedSeat struct {
	matchertest.Seat

	clock matcher.Clock
}

// Clock returns the seat's clock.
func (s *clockedSeat) Clock() matcher.Clock { return s.clock }

// calls is the record.Calls that a seat of the tests embeds, as a seat of
// this module embeds it.
type calls = record.Calls

// keepingSeat is a seat that records what was reported and keeps the call
// record of each call, as a recorder does.
type keepingSeat struct {
	matchertest.Seat
	calls
}

// newKeepingSeat returns a seat that keeps the call record of each call.
func newKeepingSeat() *keepingSeat {
	s := &keepingSeat{}
	record.Keep(&s.calls)
	return s
}

// lines returns the call records that s keeps, each as its JSON object.
func (s *keepingSeat) lines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range record.Lines(&s.calls) {
		var object map[string]any
		if err := json.Unmarshal([]byte(line), &object); err != nil {
			t.Fatalf("the call record %s is no JSON object: %v", line, err)
		}
		out = append(out, object)
	}
	return out
}
