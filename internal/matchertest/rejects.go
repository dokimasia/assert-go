// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// RejectsInvoke calls a surface's assertion that a check fails, on a check
// that reports through the matcher on the seat that the surface hands it.
// This package then does not name the check's seat type, so one suite
// serves surfaces whose seat types differ.
type RejectsInvoke func(seat *Seat, msg string, check func(matcher.Seat)) []matcher.Failure

// RunRejects drives invoke against every case that an assertion that a
// check fails must produce.
func RunRejects(t *testing.T, invoke RejectsInvoke) {
	t.Helper()

	t.Run("returns the check's records in call order, and passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		got := invoke(seat, contractMsg, func(s matcher.Seat) {
			matcher.Equal(s, matcher.Soft, 2, 1, "the first value is one")
			matcher.Equal(s, matcher.Fatal, 3, 1, "the second value is one")
		})

		checkOutcome(t, seat, Case{})
		if len(got) != 2 || got[0].Contract != "the first value is one" ||
			got[1].Contract != "the second value is one" {
			t.Fatalf("returned %+v, want the records of both failures in call order", got)
		}
		if got[1].Assertion != "equal" || got[1].Detail["got"] != 3 || got[1].Detail["want"] != 1 {
			t.Fatalf("the second record is %+v, want one of equal that states got 3 and want 1", got[1])
		}
	})

	t.Run("stops the check at its first failure that stops", func(t *testing.T) {
		t.Parallel()

		reached := false
		got := invoke(&Seat{}, contractMsg, func(s matcher.Seat) {
			matcher.True(s, matcher.Fatal, false, "the check fails")
			reached = true
		})

		if reached {
			t.Fatal("the check ran past a failure that stops it")
		}
		if len(got) != 1 || got[0].Assertion != "true" {
			t.Fatalf("returned %+v, want the record of the failure that stopped the check", got)
		}
	})

	t.Run("returns no record and passes for a check that fails through Fatalf alone", func(t *testing.T) {
		t.Parallel()

		reached := false
		seat := &Seat{}
		got := invoke(seat, contractMsg, func(s matcher.Seat) {
			s.Fatalf("the check fails")
			reached = true
		})

		checkOutcome(t, seat, Case{})
		if len(got) != 0 || reached {
			t.Fatalf("returned %+v and ran on: %v, want no record and a check that Fatalf stopped", got, reached)
		}
	})

	t.Run("returns no record and passes for a check that fails through Errorf alone", func(t *testing.T) {
		t.Parallel()

		reached := false
		seat := &Seat{}
		got := invoke(seat, contractMsg, func(s matcher.Seat) {
			s.Errorf("the check fails")
			reached = true
		})

		checkOutcome(t, seat, Case{})
		if len(got) != 0 || !reached {
			t.Fatalf("returned %+v and ran on: %v, want no record and a check that Errorf let run", got, reached)
		}
	})

	t.Run("reports a record of rejects without detail for a check that passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		got := invoke(seat, contractMsg, func(s matcher.Seat) {
			matcher.True(s, matcher.Fatal, true, "the check passes")
		})

		checkOutcome(t, seat, Case{Fails: true, Assertion: "rejects"})
		if len(got) != 0 {
			t.Fatalf("returned %+v, want no record of a check that passed", got)
		}
		if records := seat.Records(); len(records[0].Detail) != 0 {
			t.Fatalf("the record states %v, and rejects declares no detail field", records[0].Detail)
		}
	})
}
