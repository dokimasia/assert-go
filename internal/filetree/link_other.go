// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !windows

package filetree

import (
	"os"

	"go.dokimi.dev/assert/internal/fault"
)

// symlink makes the link at path in root to target. A link has no kind on
// this platform, so root makes the same link to any target.
func symlink(root *os.Root, target, path string) error {
	if err := root.Symlink(target, path); err != nil {
		return fault.New("the link cannot be made").Because(err)
	}
	return nil
}
