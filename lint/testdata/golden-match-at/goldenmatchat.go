// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package goldenmatchat

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/golden"
)

func render() string { return "" }

func correct(t *testing.T) {
	golden.MatchAt(t, "testdata/render.txt", []byte(render()), golden.ShouldUpdate())
}

func compared(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "render.txt"))
	assert.NoError(t, err, "the golden file reads")
	expect.Equal(t, render(), string(want), "Render writes the golden output") // want `golden-match: state the check with golden\.MatchAt of os\.ReadFile\(filepath\.Join\("testdata", "render\.txt"\)\)`
	fixed, err := os.ReadFile("testdata/fixed/render.txt")
	assert.NoError(t, err, "the golden file reads")
	assert.Equal(t, string(fixed), render(), "Render writes the golden output") // want `golden-match: state the check with golden.MatchAt`
}
