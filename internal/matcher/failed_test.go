// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// failedResult keeps the result of a call of Failed.
var failedResult bool

// unstatedSeat is a seat that does not report whether its test failed. It
// passes each call to a seat of internal/matchertest, which records it.
type unstatedSeat struct {
	seat *matchertest.Seat
}

func (s unstatedSeat) Helper()                            { s.seat.Helper() }
func (s unstatedSeat) Fatalf(format string, args ...any)  { s.seat.Fatalf(format, args...) }
func (s unstatedSeat) Errorf(format string, args ...any)  { s.seat.Errorf(format, args...) }
func (s unstatedSeat) ReportFault(err error, ending bool) { s.seat.ReportFault(err, ending) }

func TestFailed(t *testing.T) {
	t.Parallel()

	t.Run("Failed", func(t *testing.T) {
		t.Parallel()

		t.Run("returns false for a seat that has not failed", func(t *testing.T) {
			t.Parallel()

			if seat := (&matchertest.Seat{}); matcher.Failed(seat, "assert.Failed") || len(seat.Faults()) != 0 {
				t.Fatalf("Failed returned true or reported the faults %v, want false alone", seat.Faults())
			}
		})

		t.Run("returns true for a seat that has failed", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			seat.Errorf("the write fails")
			if !matcher.Failed(seat, "assert.Failed") {
				t.Fatal("Failed returned false for a seat that has failed")
			}
		})

		t.Run("returns Failed of the first seat of the chain that has the method", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			seat.Errorf("the write fails")
			if !matcher.Failed(wrapper{wrapper{seat}}, "assert.Failed") {
				t.Fatal("Failed returned false for a chain whose wrapped seat has failed")
			}
		})

		t.Run("ends the call with a fault of op on a chain without Failed", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			got := matcher.Failed(unstatedSeat{seat}, "assert.Failed")
			faults := seat.Faults()
			var f *fault.Error
			if got || len(faults) != 1 || !errors.As(faults[0], &f) || f.Op != "assert.Failed" ||
				f.Reason != "the seat has no Failed" {

				t.Fatalf("Failed returned %v and the seat received the faults %v, "+
					"want false and the fault of assert.Failed", got, faults)
			}
		})
	})
}

// TestFailedAllocs checks the allocation ceiling of Failed.
func TestFailedAllocs(t *testing.T) {
	checkAllocs(t, failedCases())
}

// BenchmarkFailed measures Failed.
func BenchmarkFailed(b *testing.B) {
	benchAllocs(b, failedCases())
}

// failedCases returns a call of Failed on a seat that has the method, with its
// allocation ceiling.
func failedCases() []allocCase {
	return []allocCase{
		{name: "Failed", call: func(seat matcher.Seat) { failedResult = matcher.Failed(seat, "assert.Failed") }},
	}
}
