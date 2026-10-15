// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build unix

package childtest_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
)

// The limits of the cases of LimitFileSize, in bytes.
const (
	// fileLimit is the limit that a case sets: 16 KiB.
	fileLimit = 16 << 10
	// hardLimit is the hard limit that a child sets before it asks for more,
	// far above the size of the coverage files that the child writes when it
	// exits: 1 GiB.
	hardLimit = 1 << 30
)

func TestLimit(t *testing.T) {
	t.Parallel()

	t.Run("CanLimitFileSize", func(t *testing.T) {
		t.Parallel()

		t.Run("reports that the system limits the size of a file", func(t *testing.T) {
			t.Parallel()
			assert.True(t, childtest.CanLimitFileSize, "a Unix system limits the size of a file")
		})
	})

	t.Run("LimitFileSize", func(t *testing.T) {
		t.Parallel()

		t.Run("fails a write past the limit with EFBIG", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				childtest.LimitFileSize(t, fileLimit)
				err := os.WriteFile(filepath.Join(t.TempDir(), "large"), make([]byte, 2*fileLimit), 0o600)
				assert.ErrorIs(t, err, syscall.EFBIG, "the write past the limit fails")
				return
			}
			out, err := childtest.Run(t, t.Name())
			assert.NoError(t, err, "the child passes:\n"+out)
		})
		t.Run("restores the limit when the test ends", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				var before syscall.Rlimit
				assert.NoError(t, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &before), "the limit is read")
				// A cleanup registered first runs last, after the cleanup of the limit.
				t.Cleanup(func() {
					var after syscall.Rlimit
					assert.NoError(t, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &after), "the limit is read again")
					assert.Equal(t, after, before, "the limit after the test is the limit before it")
				})
				childtest.LimitFileSize(t, fileLimit)
				return
			}
			out, err := childtest.Run(t, t.Name())
			assert.NoError(t, err, "the child passes:\n"+out)
		})
		t.Run("fails the test when the system refuses the limit", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				lowered := syscall.Rlimit{Cur: hardLimit, Max: hardLimit}
				assert.NoError(t, syscall.Setrlimit(syscall.RLIMIT_FSIZE, &lowered), "the hard limit is lowered")
				childtest.LimitFileSize(t, hardLimit+1)
				return
			}
			out, err := childtest.Run(t, t.Name())
			var exit *exec.ExitError
			assert.True(t, errors.As(err, &exit), "the child exits with a status other than 0")
			assert.Contains(t, out, "childtest: the size of a file cannot be limited to 1073741825 bytes",
				"the child's test fails with the refusal")
		})
	})
}
