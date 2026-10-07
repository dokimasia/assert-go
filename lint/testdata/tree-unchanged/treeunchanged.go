// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package treeunchanged

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/files"
)

func check() {}

func correct(t *testing.T, dir string) {
	files.Unchanged(t, os.DirFS(dir), check, "the check leaves the tree as it was")
}
