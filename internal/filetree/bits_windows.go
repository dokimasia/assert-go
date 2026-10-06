// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package filetree

import (
	"io/fs"
	"os"

	"go.dokimi.dev/assert/internal/fault"
)

// recordedBits are the permission bits that a tree reads on this platform:
// none. Its file systems record one bit, the owner's write bit of a file, as
// the file's read-only attribute, and no entry has a mode of nine bits.
const recordedBits fs.FileMode = 0

// ModesUnrecorded returns the fault of an assertion that reads the mode of an
// entry on a platform whose file systems record no permission bits, as the
// file systems of this platform do.
//
// # Allocation contract
//
// ModesUnrecorded allocates the fault and its reason.
func ModesUnrecorded() error {
	return fault.New("the file system records no permission bits")
}

// setDirMode sets nothing. Windows does not honour the read-only attribute of
// a directory, and os.Remove does not clear it from one.
func setDirMode(*os.Root, string, fs.FileMode) error {
	return nil
}
