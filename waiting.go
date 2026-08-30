// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"time"

	"go.dokimi.dev/assert/internal/matcher"
)

// Eventually runs fn every interval until it passes or timeout
// expires, then stops the test reporting the last failure.
//
//	assert.Eventually(t, 5*time.Second, 100*time.Millisecond,
//	    func(tb assert.TB) {
//	        assert.Equal(tb, cache.Get(key), want, "the cache caught up")
//	    }, "the cache converges")
//
// fn receives a seat of its own, so assertions inside it record an
// attempt rather than ending the test. fn runs at least once however
// short the timeout.
//
// This spends real time, deliberately. It is for a condition something
// outside the test makes true, which a controlled clock cannot reach:
// a clock only moves when someone advances it, and nobody will while
// this call is blocking. Where the subject reads a clock the test
// controls, drive that clock and read the answer instead.
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
// Backoff starts at a millisecond and doubles, capped at a quarter of
// the timeout. It differs from [Eventually] in what it reports: a
// predicate has no failure to carry, so this says only that the wait
// ran out. Where the reason matters, write the condition as assertions
// and use [Eventually]. This spends real time for the same reason.
func EventuallyTrue(tb TB, timeout time.Duration, pred func() bool, msg string) {
	tb.Helper()
	matcher.EventuallyTrue(tb, matcher.Fatal, timeout, pred, msg)
}
