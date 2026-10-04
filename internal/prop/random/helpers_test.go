// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// pinnedSeed is the seed of every pinned draw. The pinned draws come from
// the definition's executable reference, so that a change to a draw fails
// here before a vector moves.
const pinnedSeed = 42

// draws is the number of draws that a property of a draw checks, enough
// for every branch of every draw to run.
const draws = 2000

// sizes returns the lengths from minSize to maxSize, failing the test when
// they are invalid.
func sizes(tb testing.TB, minSize, maxSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewSizes(minSize, maxSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}
