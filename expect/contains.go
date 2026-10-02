// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Contains records a failure when haystack does not contain needle.
//
// What containing means depends on the haystack:
//
//   - Text contains text as a substring.
//   - A slice or array contains an element that equals needle as [Equal]
//     compares, so an int does not match a float.
//   - A map contains needle as a key of the map's key type.
//
// Any other type cannot be asked, and asking fails naming the type.
// opts relax the element comparison for this call alone.
func Contains(tb assert.TB, haystack, needle any, msg string, opts ...Option) {
	tb.Helper()
	matcher.Contains(tb, matcher.Soft, haystack, needle, msg, opts...)
}

// NotContains records a failure when haystack contains needle. See
// [Contains] for what containing means.
func NotContains(tb assert.TB, haystack, needle any, msg string, opts ...Option) {
	tb.Helper()
	matcher.NotContains(tb, matcher.Soft, haystack, needle, msg, opts...)
}

// ContainsInOrder records a failure when got does not contain every
// needle in the given order, each after the previous one's match ends.
//
// Use it where [Contains] is too weak. Asserting that fields render in
// a stated order catches a formatter that reorders them, which
// checking for each field separately does not.
//
//	expect.ContainsInOrder(t, err.Error(),
//	    []string{"store:", "validation:", "key"},
//	    "the error renders its fields in source order")
//
// got is a string, a []byte, or any type defined over either. The
// failure names the first needle not found and its index in needles. An
// empty needle list passes.
func ContainsInOrder(tb assert.TB, got any, needles []string, msg string) {
	tb.Helper()
	matcher.ContainsInOrder(tb, matcher.Soft, got, needles, msg)
}
