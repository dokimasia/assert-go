// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// rejection keeps the records that a call of Rejects returns.
var rejection []matcher.Failure

// TestRejects runs the shared cases of the assertion that a check fails,
// and the cases of each mode and of the call records.
func TestRejects(t *testing.T) {
	t.Parallel()

	t.Run("Rejects", func(t *testing.T) {
		t.Parallel()
		matchertest.RunRejects(t, func(s *matchertest.Seat, msg string, check func(matcher.Seat)) []matcher.Failure {
			return matcher.Rejects(s, matcher.Fatal, msg, check)
		})

		t.Run("reports a check that passes through Fatalf under Fatal", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Rejects(seat, matcher.Fatal, "the check fails", func(matcher.Seat) {})
			if len(seat.Fatals()) != 1 || len(seat.Errs()) != 0 {
				t.Fatalf("reported %q through Fatalf and %q through Errorf, want one failure through Fatalf",
					seat.Fatals(), seat.Errs())
			}
		})

		t.Run("reports a check that passes through Errorf under Soft", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Rejects(seat, matcher.Soft, "the check fails", func(matcher.Seat) {})
			if len(seat.Errs()) != 1 || len(seat.Fatals()) != 0 {
				t.Fatalf("reported %q through Errorf and %q through Fatalf, want one failure through Errorf",
					seat.Errs(), seat.Fatals())
			}
		})

		t.Run("records the calls of the check under its own call, as its run 1", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Rejects(seat, matcher.Fatal, "the check fails", func(s matcher.Seat) {
				matcher.Equal(s, matcher.Fatal, 2, 1, "the value is one")
			})

			lines := seat.lines(t)
			if len(lines) != 2 || lines[0]["assertion"] != "rejects" || lines[0]["verdict"] != "pass" ||
				lines[0]["seq"] != 1.0 {
				t.Fatalf("wrote %v, want a pass of rejects as 1 and the check's call", lines)
			}
			if lines[1]["assertion"] != "equal" || lines[1]["verdict"] != "fail" || lines[1]["parent"] != 1.0 ||
				lines[1]["run"] != 1.0 {
				t.Fatalf("the second record is %v, want the check's failure under 1 in run 1", lines[1])
			}
		})

		t.Run("returns the records that the check reports from a goroutine of its own", func(t *testing.T) {
			t.Parallel()

			got := matcher.Rejects(&matchertest.Seat{}, matcher.Fatal, "the check fails", func(s matcher.Seat) {
				done := make(chan struct{})
				go func() {
					defer close(done)
					matcher.True(s, matcher.Soft, false, "the goroutine's check fails")
				}()
				<-done
			})
			if len(got) != 1 || got[0].Contract != "the goroutine's check fails" {
				t.Fatalf("returned %+v, want the record of the goroutine's failure", got)
			}
		})
	})
}

// TestRejectsAllocs checks the allocation ceiling of a passing call of
// Rejects.
func TestRejectsAllocs(t *testing.T) {
	checkAllocs(t, rejectsCases())
}

// BenchmarkRejects measures a passing call of Rejects.
func BenchmarkRejects(b *testing.B) {
	benchAllocs(b, rejectsCases())
}

// rejectsCases returns a passing call of Rejects of a check that fails at a
// call of True, with its allocation ceiling, measured.
func rejectsCases() []allocCase {
	check := func(s matcher.Seat) { matcher.True(s, matcher.Fatal, false, "the check fails") }
	return []allocCase{
		{name: "Rejects", allocs: 7, call: func(seat matcher.Seat) {
			rejection = matcher.Rejects(seat, matcher.Fatal, allocContract, check)
		}},
	}
}
