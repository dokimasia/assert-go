// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build unix

package childtest

import (
	"errors"
	"syscall"
	"testing"
)

// CanLimitFileSize reports whether the system limits the size of a file that a
// process writes, so that [LimitFileSize] applies.
const CanLimitFileSize = true

// LimitFileSize limits each file that the process writes to size bytes until t
// ends, so a write past the limit fails with syscall.EFBIG. The limit applies
// to every goroutine of the process, so a test sets it in the child process
// that [Run] started for it. LimitFileSize restores the limit when t ends,
// before the child writes its coverage. It fails t when the system refuses
// the limit.
func LimitFileSize(t *testing.T, size uint64) {
	t.Helper()
	var held syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &held)
	err = errors.Join(err, syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: size, Max: held.Max}))
	if err != nil {
		t.Fatalf("childtest: the size of a file cannot be limited to %d bytes: %v", size, err)
	}
	t.Cleanup(func() { _ = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &held) })
}
