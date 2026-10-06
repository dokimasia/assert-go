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
// the timeout, and an attempt that passes after the timeout fails the call.
// An interval below a millisecond waits a millisecond.
//
// Eventually waits on the seat's clock. On the runtime clock it spends
// real time, up to timeout, so it suits a condition that another goroutine
// or process makes true. On a [Controlled] clock, which a [Recorder]
// supplies through [Recorder.WithClock], it advances that clock between
// attempts and spends no real time.
//
// # Allocation contract
//
// A call whose first attempt passes allocates 5 times besides what fn
// allocates.
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
// timeout. A timeout below 4 ms keeps it at a millisecond. A predicate that
// returns true after the timeout fails the call.
//
// EventuallyTrue differs from [Eventually] in what it reports. A
// predicate does not report a failure of its own, so the failure states
// only that the wait ran out. Where the reason matters, write the
// condition as assertions and use [Eventually]. EventuallyTrue waits on
// the seat's clock as Eventually does.
//
// # Allocation contract
//
// A call whose first attempt passes allocates nothing besides what pred
// allocates.
func EventuallyTrue(tb TB, timeout time.Duration, pred func() bool, msg string) {
	tb.Helper()
	matcher.EventuallyTrue(tb, matcher.Fatal, timeout, pred, msg)
}
