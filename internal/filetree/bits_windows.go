// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package filetree

import (
	"io/fs"
	"os"

	"go.dokimi.dev/assert/internal/fault"
)

// recordedBits are the permission bits that the file systems of this
// platform record: none.
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

// setMode sets no permission bit, which the file systems of this platform
// do not record.
func setMode(*os.Root, string, fs.FileMode) error {
	return nil
}
