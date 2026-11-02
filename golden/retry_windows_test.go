// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package golden_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
)

// TestRetry checks the calls on a golden file while another handle refuses
// to share the file. Windows refuses the read of a file while a handle on it
// does not share the read. It refuses a rename over a file while a handle on
// it does not share the deletion. os.Open shares the read and the write of a
// file, and not its deletion.
func TestRetry(t *testing.T) {
	t.Parallel()

	t.Run("MatchAt", func(t *testing.T) {
		t.Parallel()

		t.Run("replaces a golden file once the handle that refuses its deletion closes", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(t, files.Tree{"golden.txt": files.Text("stale")})
			path := filepath.Join(dir, "golden.txt")
			f, err := os.Open(path)
			assert.NoError(t, err, "the golden file opens")
			r := assert.NewRecorder()
			waitsFor(t, func() { golden.MatchAt(r, path, []byte("current"), updating) }, f.Close)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the update passes once the handle closes")
			files.Equal(t, os.DirFS(dir), files.Tree{"golden.txt": files.Text("current")},
				"the file contains the new output, and no temporary file remains")
		})

		t.Run("reads a golden file once the handle that refuses its read closes", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(files.Workspace(t, files.Tree{"golden.txt": files.Text("recorded")}), "golden.txt")
			name, err := syscall.UTF16PtrFromString(path)
			assert.NoError(t, err, "the path converts to UTF-16")
			h, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING,
				syscall.FILE_ATTRIBUTE_NORMAL, 0)
			assert.NoError(t, err, "the golden file opens without sharing")
			r := assert.NewRecorder()
			waitsFor(t, func() { golden.MatchAt(r, path, []byte("recorded"), checking) },
				func() error { return syscall.CloseHandle(h) })

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the comparison passes once the handle closes")
		})

		t.Run("ends with a fault while the handle that refuses the rename stays open", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(t, files.Tree{"golden.txt": files.Text("stale")})
			path := filepath.Join(dir, "golden.txt")
			f, err := os.Open(path)
			assert.NoError(t, err, "the golden file opens")
			r := assert.NewRecorder()
			golden.MatchAt(r, path, []byte("current"), updating)
			assert.NoError(t, f.Close(), "the handle closes")

			assert.Equal(t, verdicts(t, r), []string{"error"}, "the call record states the fault")
			faults := r.Faults()
			assert.Length(t, faults, 1, "the call ends with one fault")
			_ = assert.ErrorAs[*os.LinkError](t, faults[0], "the fault's cause is the error of the rename")
			files.Equal(t, os.DirFS(dir), files.Tree{"golden.txt": files.Text("stale")},
				"the golden file keeps its content, and no temporary file remains")
		})
	})
}

// waitsFor runs call on a goroutine and checks that the call does not end
// within lockWait. It then runs release and waits for the call to end.
func waitsFor(t *testing.T, call func(), release func() error) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		call()
	}()
	select {
	case <-done:
		t.Error("the call ended while a handle refused to share the golden file")
	case <-time.After(lockWait):
	}
	assert.NoError(t, release(), "the handle closes")
	<-done
}
