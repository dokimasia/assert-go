// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package golden

import (
	"errors"
	"syscall"
	"time"
)

// retryTime is the longest time for which a call retries a refused rename
// over a golden file.
const retryTime = 2 * time.Second

// errSharingViolation is ERROR_SHARING_VIOLATION. Package syscall does not
// declare it.
const errSharingViolation syscall.Errno = 32

// retried calls rename and returns the error of its last call. rename
// renames a file over a golden file. Windows refuses the rename while
// another handle on either file does not share its deletion. A reader and a
// virus scanner open such handles. The refused rename fails with
// ERROR_ACCESS_DENIED or ERROR_SHARING_VIOLATION. retried calls rename again
// after each of these two errors. It waits 1 ms before the second call and
// twice as long before each later call. It returns the error once the next
// wait would end more than retryTime after the first call.
func retried(rename func() error) error {
	deadline := time.Now().Add(retryTime)
	for wait := time.Millisecond; ; wait *= 2 {
		err := rename()
		errno, ok := errors.AsType[syscall.Errno](err)
		refused := ok && (errno == syscall.ERROR_ACCESS_DENIED || errno == errSharingViolation)
		if !refused || time.Now().Add(wait).After(deadline) {
			return err
		}
		time.Sleep(wait)
	}
}
