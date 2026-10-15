// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !unix

package childtest

import "testing"

// CanLimitFileSize reports whether the system limits the size of a file that a
// process writes, so that [LimitFileSize] applies.
const CanLimitFileSize = false

// LimitFileSize skips t, because the system sets no limit on the size of a
// file that a process writes.
func LimitFileSize(t *testing.T, _ uint64) {
	t.Helper()
	t.Skip("childtest: the system sets no limit on the size of a file")
}
