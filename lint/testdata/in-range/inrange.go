// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package inrange

import (
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func size() int { return 3 }

// maxTokens is an untyped bound.
const maxTokens = 6000

// slowStart is a typed bound.
const slowStart = 1500 * time.Millisecond

// minLength is a bound of a basic type.
const minLength int = 1

func correct(t *testing.T, n int) {
	assert.InRange(t, n, 1, 10, "the count is between one and ten")
}

func bounded(t *testing.T, n, lo, hi int, f, low, high float64, g float32) {
	assert.True(t, 0 <= n && n <= 10, "the count is at most ten")          // want `in-range: state the check with InRange`
	expect.True(t, n >= 1 && n < 10, "the count is a digit above zero")    // want `in-range: state the check with InRange`
	assert.True(t, low <= f && f <= high, "the ratio is in its range")     // want `in-range: state the check with InRange`
	assert.True(t, 0.5 < f && f < 1, "the ratio is above a half")          // want `in-range: state the check with InRange`
	assert.True(t, n > lo && n < hi, "the count is between its bounds")    // want `in-range: state the check with InRange`
	assert.True(t, 0 <= g && g <= 1, "the share is a fraction")            // want `in-range: state the check with InRange`
	assert.True(t, 0 <= size() && size() <= 10, "the size is at most ten") // want `in-range: state the check with InRange`
	assert.True(t, n <= 10 && n >= 1, "the count is between one and ten")  // want `in-range: state the check with InRange`
	if n < 0 || n > 10 {                                                   // want `in-range: state the check with InRange`
		t.Fatalf("the count %d is out of range", n)
	}
	assert.True(t, n > 0 && f < 1, "both hold")         // want `conjunction: state each operand in an assertion of its own`
	assert.True(t, n > 0 && n > 1, "the count is high") // want `conjunction: state each operand in an assertion of its own`
}

func ordered(t *testing.T, n int, u uint, f float64, d time.Duration, a, b int) {
	assert.True(t, n > 0, "the count is positive")                 // want `order: state the check with InRange`
	assert.False(t, n <= 0, "the count is positive")               // want `order: state the check with InRange`
	expect.True(t, u < 10, "the digit is below ten")               // want `order: state the check with InRange`
	assert.True(t, 10 >= n, "the count is at most ten")            // want `order: state the check with InRange`
	assert.True(t, d < time.Second, "the delay is below a second") // want `order: state the check with InRange`
	assert.True(t, f > 0.5, "the ratio is above a half")           // want `order: state the check with InRange`
	assert.True(t, n < 1<<60, "the count is below the limit")      // want `order: state the check with InRange`
	assert.True(t, n > -5, "the count is above minus five")        // want `order: state the check with InRange`
	assert.True(t, a < b, "the first is below the second")
	if n <= 0 { // want `order: state the check with InRange`
		t.Fatal("the count is not positive")
	}
	assert.True(t, "a" < "b", "the letters are in order")
}

func named(t *testing.T, tokens, length int, took time.Duration) {
	assert.True(t, tokens <= maxTokens, "the tokens fit")                // want `order: state the check with InRange`
	assert.True(t, tokens > maxTokens, "the tokens overflow")            // want `order: state the check with InRange`
	assert.True(t, took < slowStart, "Close returns before the start")   // want `order: state the check with InRange`
	assert.True(t, length <= 160+len("…"), "the summary fits on a line") // want `order: state the check with InRange`
	assert.True(t, length >= minLength, "the summary is not empty")      // want `order: state the check with InRange`
	assert.True(t, 0 <= tokens && tokens < maxTokens, "the tokens fit")  // want `in-range: state the check with InRange`
}

func exactness(t *testing.T, n int, size uint64) {
	assert.True(t, n >= -1<<53, "the count is at least -2^53")                  // want `order: state the check with InRange`
	assert.True(t, n <= 1<<53, "the count is at most 2^53")                     // want `order: state the check with InRange`
	assert.True(t, n > -1<<60, "the count is above -2^60")                      // want `order: state the check with InRange`
	assert.True(t, size < math.MaxUint64, "the size is below the largest size") // want `order: state the check with InRange`
}
