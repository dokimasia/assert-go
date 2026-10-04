// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Eventually runs fn every interval until it passes or timeout
// expires, then records the last failure and lets the test continue.
//
//	expect.Eventually(t, 5*time.Second, 100*time.Millisecond,
//	    func(tb assert.TB) {
//	        expect.Equal(tb, cache.Get(key), want, "the cache caught up")
//	    }, "the cache converges")
//
// fn receives a seat of its own, so an assertion inside it records an
// attempt and not a failure of the test. fn runs at least once however
// short the timeout. An interval below a millisecond waits a millisecond.
//
// Eventually waits on the seat's clock, as
// [go.dokimi.dev/assert.Eventually] describes.
//
// # Allocation contract
//
// A call whose first attempt passes allocates 5 times besides what fn
// allocates.
func Eventually(tb assert.TB, timeout, interval time.Duration, fn func(tb assert.TB), msg string) {
	tb.Helper()
	matcher.Eventually(tb, matcher.Soft, timeout, interval, func(trial matcher.Seat) {
		fn(trial)
	}, msg)
}

// EventuallyTrue calls pred with exponential backoff until it returns
// true or timeout expires, then records a failure and lets the test
// continue.
//
//	expect.EventuallyTrue(t, 5*time.Second, func() bool {
//	    return cache.Contains(key)
//	}, "the key appears in the cache")
//
// The backoff starts at a millisecond and doubles up to a quarter of the
// timeout. A timeout below 4 ms keeps it at a millisecond.
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
func EventuallyTrue(tb assert.TB, timeout time.Duration, pred func() bool, msg string) {
	tb.Helper()
	matcher.EventuallyTrue(tb, matcher.Soft, timeout, pred, msg)
}
