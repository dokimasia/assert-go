// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package length

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, xs []int) {
	assert.Length(t, xs, 3, "the store returns three items")
}

func counted(t *testing.T, xs []int, n int) {
	assert.Equal(t, len(xs), 3, "the store returns three items") // want `length: state the check with Length`
	assert.Equal(t, 3, len(xs), "the store returns three items") // want `length: state the check with Length`
	assert.True(t, len(xs) == n, "the store returns n items")    // want `length: state the check with Length`
	if len(xs) != 3 {                                            // want `length: state the check with Length`
		t.Fatalf("the store returns %d items", len(xs))
	}
}

func uncounted(t *testing.T, xs []int, s string, pa *[3]int, n int) {
	assert.Equal(t, len(s), 3, "the name has three bytes")
	assert.Equal(t, len(pa), 3, "the array has three elements")
	assert.Length(t, xs, n, "the store returns n items")
	assert.Equal(t, len(xs), 3, "the store returns three items", assert.EquateEmpty())
	assert.NotEqual(t, len(xs), 3, "the store does not return three items")
	assert.Equal(t, cap(xs), 3, "the store has room for three items")
	expect.Equal(t, n, 3, "the count is three")
	assert.Equal(t, n, len(xs), "the read fills xs")
	assert.Equal(t, cap(xs), len(xs), "the store has no room beyond its items")
}

func claimedByOthers(t *testing.T, xs []int, s string, n int) {
	assert.True(t, len(s) == 3, "the name has three bytes")          // want `compare: state the check with Equal`
	assert.True(t, len(xs) > 7, "the store returns more than seven") // want `order: state the check with InRange`
	assert.True(t, n == len(xs), "the read fills xs")                // want `compare: state the check with Equal`
}
