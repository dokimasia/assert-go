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
// Eventually spends real time. [go.dokimi.dev/assert.Eventually] states
// when that is the right tool and when a controlled clock is.
//
// It is written by hand and not generated, because the core cannot hand
// the body's seat type to a public package.
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
// only that the wait ran out. Where the reason matters, write the condition as
// assertions and use [Eventually]. EventuallyTrue spends real time for
// the same reason.
func EventuallyTrue(tb assert.TB, timeout time.Duration, pred func() bool, msg string) {
	tb.Helper()
	matcher.EventuallyTrue(tb, matcher.Soft, timeout, pred, msg)
}
