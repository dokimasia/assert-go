// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !windows

package filetree

import (
	"io/fs"
	"os"
)

// recordedBits are the permission bits that the file systems of this
// platform record: all nine.
const recordedBits = fs.ModePerm

// ModesUnrecorded returns the fault of an assertion that reads the mode of an
// entry on a platform whose file systems record no permission bits, and nil
// on this platform, whose file systems record them. It allocates nothing.
func ModesUnrecorded() error {
	return nil
}

// setMode sets the permission bits of the entry at name in root to mode.
func setMode(root *os.Root, name string, mode fs.FileMode) error {
	return root.Chmod(name, mode)
}
