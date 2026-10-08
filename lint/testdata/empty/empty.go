// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package empty

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, xs []int) {
	assert.Empty(t, xs, "the store is empty")
}

func emptied(t *testing.T, xs []int, m map[string]int, s string, ch chan int) {
	expect.Equal(t, len(m), 0, "the map is empty")         // want `length: state the check with Empty`
	assert.Length(t, xs, 0, "the store is empty")          // want `length: state the check with Empty`
	expect.True(t, len(s) == 0, "the name is empty")       // want `length: state the check with Empty`
	assert.False(t, len(ch) > 0, "the channel is drained") // want `length: state the check with Empty`
	expect.True(t, len(xs) < 1, "the store is empty")      // want `length: state the check with Empty`
	if len(xs) != 0 {                                      // want `length: state the check with Empty`
		t.Fatal("the store is not empty")
	}
}
