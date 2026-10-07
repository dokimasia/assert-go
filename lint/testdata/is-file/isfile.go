// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package isfile

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

func correct(t *testing.T, path string) {
	files.IsFile(t, path, "the export writes the file")
}

func stat(t *testing.T, path string) {
	info, err := os.Stat(path)
	assert.NoError(t, err, "the path exists")
	assert.True(t, info.Mode().IsRegular(), "the export writes the file") // want `is-file: state the check with files.IsFile`
	assert.True(t, info.Mode().IsDir() || info.Mode().IsRegular(), "the path is a directory or a file")
}
