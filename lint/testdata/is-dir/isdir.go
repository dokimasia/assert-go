// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package isdir

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

func correct(t *testing.T, path string) {
	files.IsDir(t, path, "the setup creates the directory")
}

func stat(t *testing.T, path string) {
	info, err := os.Stat(path)
	assert.NoError(t, err, "the path exists")
	assert.True(t, info.IsDir(), "the setup creates the directory") // want `is-dir: state the check with files.IsDir`
	if !info.Mode().IsDir() {                                       // want `is-dir: state the check with files.IsDir`
		t.Fatal("the path is no directory")
	}
	assert.False(t, info.IsDir(), "the path is no directory")
}
