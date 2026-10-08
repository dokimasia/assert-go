// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pathabsent

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
)

func lookup(name string) error { return fs.ErrNotExist }

func correct(t *testing.T, path string) {
	files.Absent(t, path, "the cleanup removes the file")
}

func stat(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "the cleanup removes the file")             // want `path-absent: state the check with files\.Absent of os\.Stat\(path\)`
	expect.ErrorIs(t, err, fs.ErrNotExist, "the cleanup removes the file")         // want `path-absent: state the check with files.Absent`
	assert.True(t, errors.Is(err, os.ErrNotExist), "the cleanup removes the file") // want `path-absent: state the check with files.Absent`
	if !os.IsNotExist(err) {                                                       // want `path-absent: state the check with files.Absent`
		t.Fatal("the file remains")
	}
	assert.False(t, os.IsNotExist(err), "the file remains")
	expect.ErrorIs(t, err, fs.ErrPermission, "the file is denied")
}

func stored(t *testing.T, name string, given error) {
	err := lookup(name)
	expect.ErrorIs(t, err, fs.ErrNotExist, "the store has no entry of the name")
	assert.True(t, os.IsNotExist(given), "the error states a missing name")
}
