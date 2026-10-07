// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package treecontains

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/files"
)

func correct(t *testing.T, dir string) {
	files.Contains(t, os.DirFS(dir), files.Tree{"go.mod": files.Text("module example.com/a\n")}, "the tree has a module")
}
