// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"io/fs"
	"strings"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// The values that a measured call returns, kept so that the compiler keeps
// the call.
var (
	kept      string
	keptEntry files.Entry
	keptBytes []byte
	errKept   error
)

// The modes of the files and directories that the tests write and read.
const (
	// fileMode is the mode of a file that a workspace writes where a tree
	// states none.
	fileMode fs.FileMode = 0o644
	// dirMode is the mode of a directory, and of an executable file, that a
	// workspace writes where a tree states none.
	dirMode fs.FileMode = 0o755
	// privateMode is a mode that only the owner may read and write.
	privateMode fs.FileMode = 0o600
)

// longName is a name longer than any file system accepts in one entry.
var longName = strings.Repeat("n", 300)

// mustRecordModes skips tb on a platform whose file systems record no
// permission bits, where a tree reads no mode and no execute bit.
func mustRecordModes(tb testing.TB) {
	tb.Helper()
	if err := filetree.ModesUnrecorded(); err != nil {
		tb.Skip(err)
	}
}

// faultSeat is the seat of a test that keeps each fault of a call that ends
// in one, and fails the test for nothing else that a seat of its own would
// not. It embeds the test's own seat, whose TempDir and Cleanup a workspace
// calls.
type faultSeat struct {
	*testing.T

	mu     sync.Mutex
	faults []error
}

// ReportFault keeps the fault, in place of ending the test.
func (s *faultSeat) ReportFault(err error, _ bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faults = append(s.faults, err)
}

// fault returns the one fault that the seat kept, and fails the test when it
// kept another number.
func (s *faultSeat) fault() *fault.Error {
	s.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	assert.Length(s.T, s.faults, 1, "the call ends in one fault")
	return assert.ErrorAs[*fault.Error](s.T, s.faults[0], "a fault")
}
