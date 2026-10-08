// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package roundtrip

import (
	"testing"

	"go.dokimi.dev/assert"
)

func label(n int) string { return encode(n) }

func encoded(t *testing.T, n int) string {
	t.Helper()
	return encode(n)
}

func TestRoundTrip(t *testing.T) {
	n := 7
	assert.Equal(t, decode(label(n)), n, "the text of a fixture parses back")
	assert.Equal(t, decode(encoded(t, n)), n, "parsing undoes formatting") // want `round-trip: state the check with RoundTrip of encoded and decode`
}
