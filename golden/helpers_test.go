// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

	var out []string
	for _, line := range r.Records() {
		var got struct {
			Verdict string `json:"verdict"`
		}
		assert.NoError(t, json.Unmarshal([]byte(line), &got), "the call record is JSON")
		out = append(out, got.Verdict)
	}
	return out
}
