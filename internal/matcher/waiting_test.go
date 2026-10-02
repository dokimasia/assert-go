// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestWaiting(t *testing.T) {
	t.Parallel()

	t.Run("Eventually", func(t *testing.T) {
		t.Parallel()
		matchertest.RunEventually(t, func(s *matchertest.Seat, timeout, interval time.Duration,
			failing func() bool, msg string,
		) {
			matcher.Eventually(s, matcher.Fatal, timeout, interval, func(trial matcher.Seat) {
				if failing() {
					matcher.True(trial, matcher.Fatal, false, matchertest.InnerReason)
				}
			}, msg)
		})
	})

	t.Run("Eventually ends an attempt at its first aborting failure", func(t *testing.T) {
		t.Parallel()

		var after atomic.Int32
		matcher.Eventually(&matchertest.Seat{}, matcher.Fatal, 0, matchertest.ShortInterval, func(trial matcher.Seat) {
			matcher.True(trial, matcher.Fatal, false, matchertest.InnerReason)
			after.Add(1)
		}, "the attempt stops")

		if n := after.Load(); n != 0 {
			t.Fatalf("the attempt ran on past its aborting failure %d times", n)
		}
	})

	t.Run("Eventually lets an attempt run on past a recording failure", func(t *testing.T) {
		t.Parallel()

		var after atomic.Int32
		matcher.Eventually(&matchertest.Seat{}, matcher.Fatal, 0, matchertest.ShortInterval, func(trial matcher.Seat) {
			matcher.True(trial, matcher.Soft, false, matchertest.InnerReason)
			after.Add(1)
		}, "the attempt runs on")

		if n := after.Load(); n != 1 {
			t.Fatalf("the attempt ran on %d times, want once", n)
		}
	})

	t.Run("Eventually panics on the caller with a panic of the body", func(t *testing.T) {
		t.Parallel()

		raised := matchertest.Raised(func() {
			matcher.Eventually(&matchertest.Seat{}, matcher.Fatal,
				matchertest.PatientTimeout, matchertest.ShortInterval,
				func(matcher.Seat) { panic(matchertest.ErrSample) }, "the panic propagates")
		})
		if err, _ := raised.(error); !errors.Is(err, matchertest.ErrSample) {
			t.Fatalf("recovered %v, want the body's own panic value", raised)
		}
	})

	// A controlled clock moves only when the retry loop waits on it, so the
	// attempts are fixed: at 0, 1, 2 and 3 seconds the deadline has not
	// passed, and at 4 seconds it has.
	t.Run("Eventually reports its attempts on the seat's clock", func(t *testing.T) {
		t.Parallel()

		seat := &clockedSeat{clock: matcher.NewControlled(clockEpoch)}
		matcher.Eventually(seat, matcher.Fatal, 3*time.Second, time.Second, func(trial matcher.Seat) {
			matcher.True(trial, matcher.Fatal, false, matchertest.InnerReason)
		}, "the body settles")

		records := seat.Records()
		if len(records) != 1 {
			t.Fatalf("reported %d records, want 1", len(records))
		}
		if got := records[0].Detail; got["attempts"] != 5 || got["last"] != matchertest.InnerReason {
			t.Fatalf("the record states %v, want 5 attempts and the last attempt's reason", got)
		}
	})

	// An interval of zero waits a millisecond, so the attempts run at 0, 1,
	// 2, 3 and 4 ms, and the deadline has passed at the last.
	t.Run("Eventually waits a millisecond for an interval of zero on the seat's clock", func(t *testing.T) {
		t.Parallel()

		seat := &clockedSeat{clock: matcher.NewControlled(clockEpoch)}
		matcher.Eventually(seat, matcher.Fatal, 3*time.Millisecond, 0, func(trial matcher.Seat) {
			matcher.True(trial, matcher.Fatal, false, matchertest.InnerReason)
		}, "the body settles")

		records := seat.Records()
		if len(records) != 1 {
			t.Fatalf("reported %d records, want 1", len(records))
		}
		if got := records[0].Detail["attempts"]; got != 5 {
			t.Fatalf("the record states %v attempts, want 5", got)
		}
	})

	// A quarter of a timeout of 2 ms is below a millisecond, so every
	// backoff is a millisecond: the predicate runs at 0, 1, 2 and 3 ms,
	// and the deadline has passed at the last.
	t.Run("EventuallyTrue keeps a backoff of a millisecond for a timeout below 4 ms", func(t *testing.T) {
		t.Parallel()

		seat := &clockedSeat{clock: matcher.NewControlled(clockEpoch)}
		matcher.EventuallyTrue(seat, matcher.Fatal, 2*time.Millisecond, func() bool { return false },
			"the predicate becomes true")

		records := seat.Records()
		if len(records) != 1 {
			t.Fatalf("reported %d records, want 1", len(records))
		}
		if got := records[0].Detail["attempts"]; got != 4 {
			t.Fatalf("the record states %v attempts, want 4", got)
		}
	})

	// The backoff doubles from a millisecond to a quarter of the timeout,
	// 250 ms, so the predicate runs at 0, 1, 3, 7, 15, 31, 63, 127, 255, 505,
	// 755 and 1,005 ms, and the deadline has passed at the last.
	t.Run("EventuallyTrue reports its attempts under a capped backoff on the seat's clock", func(t *testing.T) {
		t.Parallel()

		seat := &clockedSeat{clock: matcher.NewControlled(clockEpoch)}
		matcher.EventuallyTrue(seat, matcher.Fatal, time.Second, func() bool { return false },
			"the predicate becomes true")

		records := seat.Records()
		if len(records) != 1 {
			t.Fatalf("reported %d records, want 1", len(records))
		}
		if got := records[0].Detail["attempts"]; got != 12 {
			t.Fatalf("the record states %v attempts, want 12", got)
		}
	})

	t.Run("EventuallyTrue", func(t *testing.T) {
		t.Parallel()
		matchertest.RunEventuallyTrue(t, func(s *matchertest.Seat, timeout time.Duration,
			pred func() bool, msg string,
		) {
			matcher.EventuallyTrue(s, matcher.Fatal, timeout, pred, msg)
		})
	})
}
