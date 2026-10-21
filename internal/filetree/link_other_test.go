// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !windows

package filetree_test

import "testing"

// TestLink checks that a link to a directory leads to the directory, on a
// platform where a link has no kind.
func TestLink(t *testing.T) {
	t.Parallel()

	checkLinks(t)
}
