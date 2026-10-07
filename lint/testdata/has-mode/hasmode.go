// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package hasmode

import (
	"io/fs"
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

func correct(t *testing.T, path string) {
	files.HasMode(t, path, 0o600, "the key is private")
}

func stat(t *testing.T, path string) {
	info, err := os.Stat(path)
	assert.NoError(t, err, "the path exists")
	assert.Equal(t, info.Mode().Perm(), fs.FileMode(0o600), "the key is private") // want `has-mode: state the check with files\.HasMode of info\.Mode\(\)\.Perm\(\)`
	if info.Mode().Perm() != 0o600 {                                              // want `has-mode: state the check with files.HasMode`
		t.Fatalf("the key has the mode %v", info.Mode())
	}
}
