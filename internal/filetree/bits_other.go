// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !windows

package filetree

import (
	"io/fs"
	"os"

	"go.dokimi.dev/assert/internal/fault"
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

// setDirMode sets the permission bits of the directory at name in root to
// mode.
func setDirMode(root *os.Root, name string, mode fs.FileMode) error {
	if err := root.Chmod(name, mode); err != nil {
		return fault.New("the mode of the directory cannot be set").Because(err)
	}
	return nil
}
