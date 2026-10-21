// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package filetree

import (
	"os"
	"path/filepath"

	"go.dokimi.dev/assert/internal/fault"
)

// symlink makes the link at path in root to target with os.Symlink. That
// function makes a directory link to a target that is a directory and a file
// link to any other target. os.Root.Symlink makes a directory link only to a
// directory inside the root. Every directory above path is one that the write
// created or read as a directory, so os.Symlink makes the link inside the
// root.
func symlink(root *os.Root, target, path string) error {
	if err := os.Symlink(target, filepath.Join(root.Name(), filepath.FromSlash(path))); err != nil {
		return fault.New("the link cannot be made").Because(err)
	}
	return nil
}
