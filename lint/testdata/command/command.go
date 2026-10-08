// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package command

import (
	"testing"

	"go.dokimi.dev/assert"
)

func check(t *testing.T, n int, f float64) {
	assert.True(t, n == 3, "the count is three")
	assert.True(t, f > 0.5, "the ratio is above a half")
}
