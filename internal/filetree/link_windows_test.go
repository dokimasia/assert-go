// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package filetree_test

import "testing"

// TestLink checks that a link to a directory is a directory link, through
// which a reader lists the directory. Windows lists no directory through a
// file link.
func TestLink(t *testing.T) {
	t.Parallel()

	checkLinks(t)
}
