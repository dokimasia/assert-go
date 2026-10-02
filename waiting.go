// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"time"

	"go.dokimi.dev/assert/internal/matcher"
)

// Eventually runs fn every interval until it passes or timeout
// expires, then stops the test and reports the last failure.
//
//	assert.Eventually(t, 5*time.Second, 100*time.Millisecond,
//	    func(tb assert.TB) {
//	        assert.Equal(tb, cache.Get(key), want, "the cache caught up")
//	    }, "the cache converges")
//
// fn receives a seat of its own, so an assertion inside it records an
// attempt and does not end the test. fn runs at least once however short
// the timeout. An interval below a millisecond waits a millisecond.
//
// Eventually spends real time. It is for a condition that something
// outside the test makes true. A controlled clock moves only when the
// test advances it, and the test cannot advance it while this call
// blocks. Where the subject reads a clock that the test controls, drive
// that clock and read the result instead.
func Eventually(tb TB, timeout, interval time.Duration, fn func(tb TB), msg string) {
	tb.Helper()
	matcher.Eventually(tb, matcher.Fatal, timeout, interval, func(s matcher.Seat) {
		fn(s)
	}, msg)
}

// EventuallyTrue calls pred with exponential backoff until it returns
// true or timeout expires, then stops the test.
//
//	assert.EventuallyTrue(t, 5*time.Second, func() bool {
//	    return cache.Contains(key)
//	}, "the key appears in the cache")
//
// The backoff starts at a millisecond and doubles up to a quarter of the
// timeout. A timeout below 4 ms keeps it at a millisecond.
//
// EventuallyTrue differs from [Eventually] in what it reports. A
// predicate does not report a failure of its own, so the failure states
// only that the wait ran out. Where the reason matters, write the condition as
// assertions and use [Eventually]. EventuallyTrue spends real time for
// the same reason.
func EventuallyTrue(tb TB, timeout time.Duration, pred func() bool, msg string) {
	tb.Helper()
	matcher.EventuallyTrue(tb, matcher.Fatal, timeout, pred, msg)
}
