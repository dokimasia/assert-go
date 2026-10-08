// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package linksto

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

func correct(t *testing.T, path string) {
	files.LinksTo(t, path, "current.txt", "the link points at the current file")
}

func readlink(t *testing.T, path string) {
	target, err := os.Readlink(path)
	assert.NoError(t, err, "the path is a link")
	assert.Equal(t, target, "current.txt", "the link points at the current file") // want `links-to: state the check with files\.LinksTo of os\.Readlink\(path\)`
	assert.Equal(t, "current.txt", target, "the link points at the current file") // want `links-to: state the check with files\.LinksTo of os\.Readlink\(path\)`
}

func base(t *testing.T, path string) {
	target, err := os.Readlink(path)
	assert.NoError(t, err, "the path is a link")
	assert.Equal(t, filepath.Base(target), "current.txt", "the link points at a file named current.txt")
}
