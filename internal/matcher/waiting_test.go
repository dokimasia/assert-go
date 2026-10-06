// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// lateClock is a clock whose Sleep returns a nanosecond after the duration
// has passed, as a sleep on the runtime clock returns after it.
type lateClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the current instant.
func (c *lateClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep moves the clock a nanosecond past d.
func (c *lateClock) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d + time.Nanosecond)
}

// TestWaiting runs the shared cases of the retrying assertions, and their
// cases under a seat's clock.
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

	t.Run("Eventually records the calls of each attempt under its own call", func(t *testing.T) {
		t.Parallel()

		seat := newKeepingSeat()
		attempts := 0
		matcher.Eventually(seat, matcher.Fatal, matchertest.PatientTimeout, matchertest.ShortInterval,
			func(trial matcher.Seat) {
				attempts++
				matcher.True(trial, matcher.Fatal, attempts > 1, "the second attempt passes")
			}, "the body settles")

		lines := seat.lines(t)
		if len(lines) != 3 {
			t.Fatalf("wrote %d records, want 3: %v", len(lines), lines)
		}
		if lines[0]["assertion"] != "eventually" || lines[0]["verdict"] != "pass" || lines[0]["seq"] != 1.0 {
			t.Fatalf("wrote %v first, want the passing call of eventually as 1", lines[0])
		}
		for i, verdict := range []string{"fail", "pass"} {
			got := lines[i+1]
			if got["verdict"] != verdict || got["parent"] != 1.0 || got["run"] != float64(i+1) {
				t.Fatalf("wrote %v, want a %s of attempt %d under 1", got, verdict, i+1)
			}
		}
	})

	// The attempt passes and hands its seat to a goroutine, which reports a
	// failure on it after Eventually has read the attempt's outcome.
	t.Run("Eventually ignores a failure that a goroutine reports on an ended attempt", func(t *testing.T) {
		t.Parallel()

		seat := &matchertest.Seat{}
		trials := make(chan matcher.Seat, 1)
		matcher.Eventually(seat, matcher.Fatal, matchertest.PatientTimeout, matchertest.ShortInterval,
			func(trial matcher.Seat) { trials <- trial }, "the body settles")

		reported := make(chan struct{})
		go func() {
			matcher.True(<-trials, matcher.Soft, false, matchertest.InnerReason)
			close(reported)
		}()
		select {
		case <-reported:
		case <-time.After(matchertest.PatientTimeout):
			t.Fatal("the late failure did not return")
		}
		if seat.Failed() {
			t.Fatalf("reported %q for a body whose attempt passed", seat.First())
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
	// attempts are fixed: at 0, 1, 2 and 3 seconds, the last at the
	// deadline.
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
		if got := records[0].Detail; got["attempts"] != 4 || got["last"] != matchertest.InnerReason {
			t.Fatalf("the record states %v, want 4 attempts and the last attempt's reason", got)
		}
	})

	// The wait after the first attempt is cut from an hour to the
	// millisecond that is left, so the second attempt runs at the deadline
	// and the third would run after it.
	t.Run("Eventually runs no attempt after its timeout", func(t *testing.T) {
		t.Parallel()

		clock := matcher.NewControlled(clockEpoch)
		seat := &clockedSeat{clock: clock}
		matcher.Eventually(seat, matcher.Fatal, time.Millisecond, time.Hour, func(trial matcher.Seat) {
			matcher.True(trial, matcher.Fatal, !clock.Now().Before(clockEpoch.Add(time.Hour)),
				matchertest.InnerReason)
		}, "the body settles within a millisecond")

		records := seat.Records()
		if len(records) != 1 || records[0].Detail["attempts"] != 2 {
			t.Fatalf("reported %v, want one record of 2 attempts", records)
		}
		if got := clock.Now(); !got.Equal(clockEpoch.Add(time.Millisecond)) {
			t.Fatalf("the clock reads %v, want the deadline, a millisecond after the start", got)
		}
	})

	// The wait after the first attempt is cut to the millisecond that is
	// left, and ends a nanosecond past the deadline, where the body would
	// pass.
	t.Run("Eventually runs no attempt after a wait that ends past its timeout", func(t *testing.T) {
		t.Parallel()

		clock := &lateClock{now: clockEpoch}
		seat := &clockedSeat{clock: clock}
		matcher.Eventually(seat, matcher.Fatal, time.Millisecond, time.Hour, func(trial matcher.Seat) {
			matcher.True(trial, matcher.Fatal, clock.Now().After(clockEpoch.Add(time.Millisecond)),
				matchertest.InnerReason)
		}, "the body settles within a millisecond")

		if records := seat.Records(); len(records) != 1 || records[0].Detail["attempts"] != 1 {
			t.Fatalf("reported %v, want one record of 1 attempt", records)
		}
	})

	// The wait after the attempt at 2 s is cut to the second that is left,
	// so the last attempt runs at the deadline, where the body passes.
	t.Run("Eventually runs its last attempt at the deadline", func(t *testing.T) {
		t.Parallel()

		clock := matcher.NewControlled(clockEpoch)
		seat := &clockedSeat{clock: clock}
		matcher.Eventually(seat, matcher.Fatal, 3*time.Second, 2*time.Second, func(trial matcher.Seat) {
			matcher.True(trial, matcher.Fatal, !clock.Now().Before(clockEpoch.Add(3*time.Second)),
				matchertest.InnerReason)
		}, "the body settles at the deadline")

		if records := seat.Records(); len(records) != 0 {
			t.Fatalf("reported %v, want a pass", records)
		}
	})

	// The body passes an hour after its attempt starts, which is past the
	// deadline a millisecond after the start.
	t.Run("Eventually fails a body that passes after its deadline", func(t *testing.T) {
		t.Parallel()

		clock := matcher.NewControlled(clockEpoch)
		seat := &clockedSeat{clock: clock}
		matcher.Eventually(seat, matcher.Fatal, time.Millisecond, time.Millisecond, func(matcher.Seat) {
			clock.Advance(time.Hour)
		}, "the body settles within a millisecond")

		records := seat.Records()
		if len(records) != 1 {
			t.Fatalf("reported %d records, want 1", len(records))
		}
		if got := records[0].Detail; got["attempts"] != 1 || got["last"] != "" {
			t.Fatalf("the record states %v, want 1 attempt and no last reason", got)
		}
	})

	// An interval of zero waits a millisecond, so the attempts run at 0, 1,
	// 2 and 3 ms, the last at the deadline.
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
		if got := records[0].Detail["attempts"]; got != 4 {
			t.Fatalf("the record states %v attempts, want 4", got)
		}
	})

	// A quarter of a timeout of 2 ms is below a millisecond, so every
	// backoff is a millisecond: the predicate runs at 0, 1 and 2 ms, the
	// last at the deadline.
	t.Run("EventuallyTrue keeps a backoff of a millisecond for a timeout below 4 ms", func(t *testing.T) {
		t.Parallel()

		seat := &clockedSeat{clock: matcher.NewControlled(clockEpoch)}
		matcher.EventuallyTrue(seat, matcher.Fatal, 2*time.Millisecond, func() bool { return false },
			"the predicate becomes true")

		records := seat.Records()
		if len(records) != 1 {
			t.Fatalf("reported %d records, want 1", len(records))
		}
		if got := records[0].Detail["attempts"]; got != 3 {
			t.Fatalf("the record states %v attempts, want 3", got)
		}
	})

	// The predicate runs at 0, 1 and 2 ms, and would be true from 3 ms.
	t.Run("EventuallyTrue runs no attempt after its timeout", func(t *testing.T) {
		t.Parallel()

		clock := matcher.NewControlled(clockEpoch)
		seat := &clockedSeat{clock: clock}
		matcher.EventuallyTrue(seat, matcher.Fatal, 2*time.Millisecond, func() bool {
			return !clock.Now().Before(clockEpoch.Add(3 * time.Millisecond))
		}, "the predicate becomes true within 2 ms")

		if records := seat.Records(); len(records) != 1 || records[0].Detail["attempts"] != 3 {
			t.Fatalf("reported %v, want one record of 3 attempts", records)
		}
	})

	// The predicate runs at 0 and at 1 ms and 1 ns. The next wait is cut to
	// the time that is left, and ends a nanosecond past the deadline, where
	// the predicate would be true.
	t.Run("EventuallyTrue runs no attempt after a wait that ends past its timeout", func(t *testing.T) {
		t.Parallel()

		clock := &lateClock{now: clockEpoch}
		seat := &clockedSeat{clock: clock}
		matcher.EventuallyTrue(seat, matcher.Fatal, 2*time.Millisecond, func() bool {
			return clock.Now().After(clockEpoch.Add(2 * time.Millisecond))
		}, "the predicate becomes true within 2 ms")

		if records := seat.Records(); len(records) != 1 || records[0].Detail["attempts"] != 2 {
			t.Fatalf("reported %v, want one record of 2 attempts", records)
		}
	})

	// The predicate returns true an hour after its attempt starts, which is
	// past the deadline 2 ms after the start.
	t.Run("EventuallyTrue fails a predicate that returns true after its deadline", func(t *testing.T) {
		t.Parallel()

		clock := matcher.NewControlled(clockEpoch)
		seat := &clockedSeat{clock: clock}
		matcher.EventuallyTrue(seat, matcher.Fatal, 2*time.Millisecond, func() bool {
			clock.Advance(time.Hour)
			return true
		}, "the predicate becomes true within 2 ms")

		if records := seat.Records(); len(records) != 1 || records[0].Detail["attempts"] != 1 {
			t.Fatalf("reported %v, want one record of 1 attempt", records)
		}
	})

	// The backoff doubles from a millisecond to a quarter of the timeout,
	// 250 ms, so the predicate runs at 0, 1, 3, 7, 15, 31, 63, 127, 255, 505
	// and 755 ms, and at the deadline, 1,000 ms, after a wait cut to 245 ms.
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

// TestWaitingAllocs checks the allocation ceiling of a passing call of
// each retrying assertion.
func TestWaitingAllocs(t *testing.T) {
	checkAllocs(t, waitingCases())
}

// BenchmarkWaiting measures a passing call of each retrying assertion.
func BenchmarkWaiting(b *testing.B) {
	benchAllocs(b, waitingCases())
}

// waitingCases returns a call of each retrying assertion that passes at its
// first attempt, with its allocation ceiling, measured.
func waitingCases() []allocCase {
	settled := func(matcher.Seat) {}
	ready := func() bool { return true }
	return []allocCase{
		{name: "Eventually", allocs: 4, call: func(seat matcher.Seat) {
			matcher.Eventually(seat, matcher.Fatal, time.Second, time.Millisecond, settled, allocContract)
		}},
		{name: "EventuallyTrue", call: func(seat matcher.Seat) {
			matcher.EventuallyTrue(seat, matcher.Fatal, time.Second, ready, allocContract)
		}},
	}
}
