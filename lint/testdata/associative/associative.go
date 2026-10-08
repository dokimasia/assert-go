// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package associative

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func add(a, b int) int { return a + b }

func next() int { return 1 }

func correct(t *testing.T) {
	assert.Associative(t, add, 1, 2, 3, "addition associates")
}

func grouped(t *testing.T, a, b, c int) {
	assert.Equal(t, add(add(a, b), c), add(a, add(b, c)), "addition associates")           // want `associative: state the check with Associative`
	expect.Equal(t, add(a, add(b, c)), add(add(a, b), c), "addition associates")           // want `associative: state the check with Associative`
	assert.True(t, add(add(a, b), c) == add(a, add(b, c)), "addition associates")          // want `associative: state the check with Associative`
	assert.Equal(t, add(add(next(), b), c), add(next(), add(b, c)), "addition associates") // want `associative: state the check with Associative`
	assert.Equal(t, add(add(a, b), c), add(a, add(c, b)), "the groupings differ")
	assert.Equal(t, add(add(a, b), c), add(a, b), "the sums differ")
}
