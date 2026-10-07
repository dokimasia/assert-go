// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package treeequal

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/files"
)

func correct(t *testing.T) {
	dir := files.Workspace(t, files.Tree{"a/a.go": files.Text("package a\n")})
	files.Equal(t, os.DirFS(dir), files.Tree{"a/a.go": files.Text("package a\n")}, "the tree is unchanged")
}
