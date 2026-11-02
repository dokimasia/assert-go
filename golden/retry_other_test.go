// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !windows

package golden_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
)

// TestRetry checks a call on a golden file while another handle on the file
// is open. No handle refuses a rename over a file on this platform, so the
// call does not wait for the handle to close.
func TestRetry(t *testing.T) {
	t.Parallel()

	t.Run("MatchAt", func(t *testing.T) {
		t.Parallel()

		t.Run("replaces a golden file while another handle on it is open", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(t, files.Tree{"golden.txt": files.Text("stale")})
			path := filepath.Join(dir, "golden.txt")
			f, err := os.Open(path)
			assert.NoError(t, err, "the golden file opens")
			r := assert.NewRecorder()
			golden.MatchAt(r, path, []byte("current"), updating)
			replaced, err := io.ReadAll(f)
			assert.NoError(t, err, "the handle reads the file that it opened")
			assert.NoError(t, f.Close(), "the handle closes")

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the update passes while the handle is open")
			files.Equal(t, os.DirFS(dir), files.Tree{"golden.txt": files.Text("current")},
				"the file contains the new output, and no temporary file remains")
			assert.Equal(t, string(replaced), "stale", "the handle reads the content of the replaced file")
		})
	})
}
