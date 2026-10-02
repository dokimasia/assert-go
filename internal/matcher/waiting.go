// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// probe is a [Seat] that records a trial's failure instead of
// reporting it, so a retrying assertion can run a body many times and
// report only the last outcome.
//
// It is not the public recorder, because the package that declares the
// recorder imports this one. A retry loop needs nothing more than the
// first message of each attempt.
//
// Fatalf ends the calling goroutine once it has recorded, as a test's
// Fatalf does, so an aborting assertion stops the trial at its first
// failure. A trial runs on a goroutine of its own for that reason.
type probe struct {
	mu     sync.Mutex
	failed bool
	msg    string
}

func (*probe) Helper() {}

func (p *probe) Fatalf(format string, args ...any) {
	p.record(format, args...)
	runtime.Goexit()
}

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

// trial runs fn once with a probe of its own, on a goroutine of its own,
// and returns the probe once fn has returned or a fatal failure has ended
// it. A panic in fn panics again on the calling goroutine with the same
// value, so it is not taken for a failed attempt.
func trial(fn func(Seat)) *probe {
	p := &probe{}
	ended := make(chan any, 1)
	go func() {
		// recover returns nil for a body that returned and for one that a
		// fatal failure ended through runtime.Goexit.
		defer func() { ended <- recover() }()
		fn(p)
	}()
	if raised := <-ended; raised != nil {
		panic(raised)
	}
	return p
}

// Eventually runs fn every interval until it passes or timeout
// expires, reporting the last failure when it never passes.
//
// fn receives a seat of its own, so the assertions inside it record a
// trial rather than ending the test. Each attempt runs on a goroutine of
// its own, and an aborting assertion that fails ends that attempt there,
// as it ends a test. Only the final attempt's failure is reported. A
// panic in fn is no failed attempt: it panics again on the goroutine that
// called Eventually.
//
//	matcher.Eventually(seat, matcher.Fatal, 5*time.Second, 100*time.Millisecond,
//	    func(s matcher.Seat) {
//	        matcher.Equal(s, matcher.Fatal, cache.Get(key), want, "the cache caught up")
//	    }, "the cache converges")
//
// This spends real time. It is for a condition that something outside
// the test makes true. A controlled clock moves only when the test
// advances it, and the test cannot advance it while this call blocks.
// Where the subject reads a clock the test controls, drive that clock and
// read the result instead.
//
// Size the timeout for the slowest machine that will run it. fn runs
// at least once however short the timeout. An interval below a
// millisecond waits a millisecond, so the attempts on a controlled clock
// end at the timeout.
func Eventually(seat Seat, mode Mode, timeout, interval time.Duration, fn func(Seat), msg string) {
	seat.Helper()

	clock := ClockOf(seat)
	deadline := clock.Now().Add(timeout)
	for attempt := 0; ; attempt++ {
		last, failed := trial(fn).outcome()
		if !failed {
			return
		}
		if clock.Now().After(deadline) {
			Fail(seat, mode, "eventually", msg, map[string]any{
				"attempts": attempt + 1, "last": last,
			})
			return
		}
		wait(clock, max(interval, minWait))
	}
}

// minWait is the shortest wait between two attempts of a retrying
// assertion. Every wait moves a controlled clock forward, so the attempts
// end at the timeout.
const minWait = time.Millisecond

// EventuallyTrue calls pred with exponential backoff until it returns
// true or timeout expires, reporting a timeout when it never does.
//
// The backoff starts at a millisecond and doubles up to a quarter of the
// timeout, so the last attempts are not one long sleep. A timeout below
// 4 ms keeps the backoff at a millisecond.
//
// It differs from [Eventually] in what it reports. A predicate does not
// report a failure of its own, so this states only that the wait ran out. Where
// the reason matters, write the condition as assertions and use
// [Eventually]. This spends real time for the same reason [Eventually]
// does.
func EventuallyTrue(seat Seat, mode Mode, timeout time.Duration, pred func() bool, msg string) {
	seat.Helper()

	clock := ClockOf(seat)
	deadline := clock.Now().Add(timeout)
	backoff, maxBackoff := minWait, max(timeout/4, minWait)

	for attempt := 0; ; attempt++ {
		if pred() {
			return
		}
		if clock.Now().After(deadline) {
			Fail(seat, mode, "eventually-true", msg,
				map[string]any{"attempts": attempt + 1})
			return
		}

		wait(clock, backoff)
		backoff = min(max(2*backoff, minWait), maxBackoff)
	}
}
