// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package golden

import (
	"errors"
	"syscall"
	"time"
)

// retryTime is the longest time for which a call retries a refused read or
// rename of a golden file.
const retryTime = 2 * time.Second

// errSharingViolation is ERROR_SHARING_VIOLATION. Package syscall does not
// declare it.
const errSharingViolation syscall.Errno = 32

// retried calls f and returns the result of its last call. f reads a golden
// file or renames a file over one. Windows refuses the read of a file while
// another handle on it does not share the read. It refuses a rename over a
// file while another handle on it does not share its deletion. A reader and
// a virus scanner open such handles. The refused call fails with
// ERROR_ACCESS_DENIED or ERROR_SHARING_VIOLATION. retried calls f again after
// each of these two errors. It waits 1 ms before the second call and twice as
// long before each later call. It returns the error once the next wait would
// end more than retryTime after the first call.
func retried[T any](f func() (T, error)) (T, error) {
	deadline := time.Now().Add(retryTime)
	for wait := time.Millisecond; ; wait *= 2 {
		value, err := f()
		errno, ok := errors.AsType[syscall.Errno](err)
		refused := ok && (errno == syscall.ERROR_ACCESS_DENIED || errno == errSharingViolation)
		if !refused || time.Now().Add(wait).After(deadline) {
			return value, err
		}
		time.Sleep(wait)
	}
}
