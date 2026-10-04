// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Contains stops the test when haystack does not contain needle.
//
// What containing means depends on the haystack:
//
//   - Text contains text as a substring.
//   - A slice or array contains an element that equals needle as [Equal]
//     compares, so an int does not match a float.
//   - A map contains needle as a key of the map's key type.
//
// A haystack of any other type fails the assertion. opts relax the element
// comparison for this call alone.
//
// # Allocation contract
//
// A passing call on text allocates nothing. A call on a slice or an array
// compares its elements as [Equal] does: 76 allocations for a slice of
// three ints whose last element is needle.
func Contains(tb TB, haystack, needle any, msg string, opts ...Option) {
	tb.Helper()
	matcher.Contains(tb, matcher.Fatal, haystack, needle, msg, opts...)
}

// NotContains stops the test when haystack contains needle. See
// [Contains] for what containing means.
//
// # Allocation contract
//
// A passing call on text allocates nothing. A call on a slice or an array
// compares its elements as [Contains] does.
func NotContains(tb TB, haystack, needle any, msg string, opts ...Option) {
	tb.Helper()
	matcher.NotContains(tb, matcher.Fatal, haystack, needle, msg, opts...)
}

// ContainsInOrder stops the test when got does not contain every needle
// in the given order, each after the previous one's match ends.
//
// Use it where [Contains] is too weak. Asserting that fields render in
// a stated order catches a formatter that reorders them, which
// checking for each field separately does not.
//
//	assert.ContainsInOrder(t, err.Error(),
//	    []string{"store:", "validation:", "key"},
//	    "the error renders its fields in source order")
//
// got is a string, a []byte, or any type defined over either. The
// failure names the first needle not found and its index in needles. An
// empty needle list passes.
//
// # Allocation contract
//
// A passing call on a string allocates nothing. A call on a []byte
// allocates twice: the bytes in the interface of got, and their text.
func ContainsInOrder(tb TB, got any, needles []string, msg string) {
	tb.Helper()
	matcher.ContainsInOrder(tb, matcher.Fatal, got, needles, msg)
}

// Permutation stops the test when got and want do not contain the same
// elements, each as often, in any order.
//
//	assert.Permutation(t, store.Keys(ctx), []string{"a", "b", "c"},
//	    "Keys returns every stored key once")
//
// Elements compare as [Equal] compares them, so an int does not match a
// float, and a NaN matches nothing unless [EquateNaNs] applies. A nil
// slice does not match an empty one unless [EquateEmpty] applies. The
// check makes at most len(got)·len(want) comparisons.
//
// # Allocation contract
//
// A passing call on two slices of three ints allocates 120 times, in the
// comparisons of their elements.
func Permutation[T any](tb TB, got, want []T, msg string, opts ...Option) {
	tb.Helper()
	matcher.Permutation(tb, matcher.Fatal, got, want, msg, opts...)
}
