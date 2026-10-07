// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package notempty

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, xs []int) {
	assert.NotEmpty(t, xs, "the store returns an item")
}

func filled(t *testing.T, xs []int, m map[string]int, s string, arr [3]int) {
	expect.NotEqual(t, len(s), 0, "the name is set")           // want `length: state the check with NotEmpty`
	assert.True(t, len(xs) > 0, "the store returns an item")   // want `length: state the check with NotEmpty`
	assert.True(t, 0 < len(m), "the map has an entry")         // want `length: state the check with NotEmpty`
	expect.True(t, len(arr) >= 1, "the array has an element")  // want `length: state the check with NotEmpty`
	expect.True(t, (len(s)) != 0, "the name is set")           // want `length: state the check with NotEmpty`
	assert.False(t, len(xs) <= 0, "the store returns an item") // want `length: state the check with NotEmpty`
	if len(xs) == 0 {                                          // want `length: state the check with NotEmpty`
		t.Fatal("the store is empty")
	}
}
