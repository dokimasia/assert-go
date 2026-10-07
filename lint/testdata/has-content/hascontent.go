// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package hascontent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

func render() string { return "" }

func correct(t *testing.T, path string) {
	files.HasContent(t, path, "hello\n", "the file holds the greeting")
}

func read(t *testing.T, path string) {
	content, err := os.ReadFile(path)
	assert.NoError(t, err, "the file reads")
	assert.Equal(t, string(content), "hello\n", "the file holds the greeting") // want `has-content: state the check with files.HasContent`
	if !bytes.Equal(content, []byte("hello\n")) {                              // want `has-content: state the check with files.HasContent`
		t.Fatal("the file holds no greeting")
	}
}

func written(t *testing.T, dir string) {
	want := render()
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil { // want `error-nil: state the check with NoError`
		t.Fatal(err)
	}
	assert.Equal(t, string(got), want, "the file holds the rendered text") // want `has-content: state the check with files\.HasContent of os\.ReadFile\(filepath\.Join\(dir, "out\.txt"\)\)`
}
