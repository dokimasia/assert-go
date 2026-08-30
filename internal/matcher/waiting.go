// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"fmt"
	"sync"
	"time"
)

// probe is a [Seat] that records a trial's failure instead of
// reporting it, so a retrying assertion can run a body many times and
// report only the last outcome.
//
// It is not the public recorder: this package sits below the surface
// that declares that one, and a retry loop needs nothing more than
// the first message of each attempt.
type probe struct {
	mu     sync.Mutex
	failed bool
	msg    string
}

func (*probe) Helper() {}

func (p *probe) Fatalf(format string, args ...any) { p.record(format, args...) }
func (p *probe) Errorf(format string, args ...any) { p.record(format, args...) }

func (p *probe) record(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.failed {
		p.failed = true
		p.msg = fmt.Sprintf(format, args...)
	}
}

// outcome reports whether the trial failed, and with what.
func (p *probe) outcome() (msg string, failed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.msg, p.failed
}

// Eventually runs fn every interval until it passes or timeout
// expires, reporting the last failure when it never passes.
//
// fn receives a seat of its own, so the assertions inside it record a
// trial rather than ending the test. Only the final attempt's failure
// is reported.
//
//	matcher.Eventually(seat, matcher.Fatal, 5*time.Second, 100*time.Millisecond,
//	    func(s matcher.Seat) {
//	        matcher.Equal(s, matcher.Fatal, cache.Get(key), want, "the cache caught up")
//	    }, "the cache converges")
//
// This spends real time, deliberately. It is for a condition something
// outside the test makes true, which is exactly what a controlled
// clock cannot reach: a clock only moves when someone advances it, and
// nobody will while this call is blocking. Where the subject reads a
// clock the test controls, drive that clock and read the answer
// instead.
//
// Size the timeout for the slowest machine that will run it. fn runs
// at least once however short the timeout.
func Eventually(seat Seat, mode Mode, timeout, interval time.Duration, fn func(Seat), msg string) {
	seat.Helper()

	deadline := time.Now().Add(timeout)
	for attempt := 0; ; attempt++ {
		p := &probe{}
		fn(p)

		last, failed := p.outcome()
		if !failed {
			return
		}
		if time.Now().After(deadline) {
			Report(seat, mode, "%s: still failing after %v and %d attempts: %s",
				msg, timeout, attempt+1, last)
			return
		}
		time.Sleep(interval)
	}
}

// EventuallyTrue calls pred with exponential backoff until it returns
// true or timeout expires, reporting a timeout when it never does.
//
// Backoff starts at a millisecond and doubles, capped at a quarter of
// the timeout so the last attempts are not one long sleep.
//
// It differs from [Eventually] in what it reports. A predicate has no
// failure to carry, so this says only that the wait ran out; where the
// reason matters, write the condition as assertions and use
// [Eventually]. This spends real time for the same reason [Eventually]
// does.
func EventuallyTrue(seat Seat, mode Mode, timeout time.Duration, pred func() bool, msg string) {
	seat.Helper()

	const firstBackoff = time.Millisecond

	deadline := time.Now().Add(timeout)
	backoff := firstBackoff
	maxBackoff := timeout / 4

	for attempt := 0; ; attempt++ {
		if pred() {
			return
		}
		if time.Now().After(deadline) {
			Report(seat, mode, "%s: still false after %v and %d attempts",
				msg, timeout, attempt+1)
			return
		}

		time.Sleep(backoff)
		if backoff *= 2; backoff > maxBackoff && maxBackoff > 0 {
			backoff = maxBackoff
		}
	}
}
