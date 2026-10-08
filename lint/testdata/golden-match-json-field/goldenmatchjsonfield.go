// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package goldenmatchjsonfield

import (
	"testing"

	"go.dokimi.dev/assert/golden"
)

func correct(t *testing.T, got []byte) {
	golden.MatchJSONField(t, "testdata/golden/responses.json", "list", got, golden.ShouldUpdate(),
		golden.ScrubTimestamps())
}
