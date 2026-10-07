// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pairwise

import (
	"cmp"
	"slices"
	"sort"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, xs []int) {
	assert.Pairwise(t, xs, func(earlier, later int) bool { return earlier <= later }, "the store returns its items in order")
}

func sorted(t *testing.T, xs []int, names []string, ps []float64) {
	assert.True(t, slices.IsSorted(xs), "the store returns its items in order")                             // want `pairwise: state the check with Pairwise`
	expect.True(t, slices.IsSortedFunc(names, cmp.Compare[string]), "the names are in order")               // want `pairwise: state the check with Pairwise`
	assert.True(t, sort.SliceIsSorted(ps, func(i, j int) bool { return ps[i] < ps[j] }), "the ratios rise") // want `pairwise: state the check with Pairwise`
	assert.True(t, sort.IntsAreSorted(xs), "the store returns its items in order")                          // want `pairwise: state the check with Pairwise`
	if !sort.StringsAreSorted(names) {                                                                      // want `pairwise: state the check with Pairwise`
		t.Fatal("the names are out of order")
	}
	assert.False(t, slices.IsSorted(xs), "the store shuffles its items")
}
