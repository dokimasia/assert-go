// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// The values past the members of each enumeration.
const (
	// invalidKind is the first value past the three kinds.
	invalidKind = choice.Kind(3)
	// invalidNaNPolicy is the first value past the two policies.
	invalidNaNPolicy = choice.NaNPolicy(2)
)

// byteK is one more than the largest byte, the k of a byte string.
const byteK = 256

// minIntMagnitude is the magnitude of the most negative Int, 2^63.
const minIntMagnitude = 1 << 63

// negativeZero is -0.
var negativeZero = math.Copysign(0, -1)

// signedBounds returns the bounds [lo, hi] of two int64 values, failing
// the test when they are invalid.
func signedBounds(tb testing.TB, lo, hi int64) choice.IntegerBounds {
	tb.Helper()
	b, err := choice.NewIntegerBounds(choice.IntOf(lo), choice.IntOf(hi))
	assert.NoError(tb, err, "the bounds are valid")
	return b
}

// floatBounds returns the float bounds [lo, hi] of width w under the NaN
// policy, failing the test when they are invalid.
func floatBounds(tb testing.TB, lo, hi float64, nan choice.NaNPolicy, w choice.Width) choice.FloatBounds {
	tb.Helper()
	b, err := choice.NewFloatBounds(lo, hi, nan, w)
	assert.NoError(tb, err, "the bounds are valid")
	return b
}

// sizes returns the lengths from minSize to maxSize, failing the test when
// they are invalid.
func sizes(tb testing.TB, minSize, maxSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewSizes(minSize, maxSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}

// unboundedSizes returns the lengths of minSize or more, failing the test
// when they are invalid.
func unboundedSizes(tb testing.TB, minSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewUnboundedSizes(minSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}

// sequenceBounds returns the bounds of sequences of integers in [0, k)
// with lengths from minSize to maxSize, failing the test when they are
// invalid.
func sequenceBounds(tb testing.TB, k uint32, minSize, maxSize int) choice.SequenceBounds {
	tb.Helper()
	b, err := choice.NewSequenceBounds(k, sizes(tb, minSize, maxSize))
	assert.NoError(tb, err, "the bounds are valid")
	return b
}

// sequenceChoice returns a sequence choice of the elements.
func sequenceChoice(elements ...uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}
