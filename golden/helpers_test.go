// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/filetree"
)

// Whether a run may rewrite its golden files. Named rather than
// repeated, because a case passing the wrong one silently tests the
// other path.
const (
	updating = true
	checking = false
)

// goldenPerm is the mode of a golden file that a test writes.
const goldenPerm = 0o644

// lockWait is how long a case gives a call on a golden file to read or
// replace the file while another call has the file's lock or while a handle
// refuses to share the file. A call that read or replaced the file would do
// so within microseconds.
const lockWait = 100 * time.Millisecond

// mustRecordModes skips t on a platform whose file systems record no
// permission bits, where a tree reads no mode and no execute bit.
func mustRecordModes(t *testing.T) {
	t.Helper()
	if err := filetree.ModesUnrecorded(); err != nil {
		t.Skip(err)
	}
}

// read returns the content of a golden file.
func read(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	assert.NoError(t, err, "the golden file can be read back")

	return string(raw)
}

// dangling returns the path of a symbolic link whose target's directory
// does not exist. Reading it finds no file, and writing it fails, also for
// a process that may write anywhere.
func dangling(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	assert.NoError(t, os.Symlink(filepath.Join(dir, "absent", "target"), link), "the link can be made")

	return link
}

// verdicts returns the verdict of each call record that r kept, in order.
func verdicts(t *testing.T, r *assert.Recorder) []string {
	t.Helper()

	records := r.Records()
	out := make([]string, 0, len(records))
	for _, line := range records {
		var got struct {
			Verdict string `json:"verdict"`
		}
		assert.NoError(t, json.Unmarshal([]byte(line), &got), "the call record is JSON")
		out = append(out, got.Verdict)
	}
	return out
}
