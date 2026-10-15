// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
)

// conventionalDir is where [Match] looks, relative to the test's own
// directory. [MatchAt] takes a path instead.
const conventionalDir = "testdata/golden"

// Permissions for a golden file this package writes, and for the
// directory it creates for one.
const (
	filePerm fs.FileMode = 0o644
	dirPerm  fs.FileMode = 0o755
)

// maxLinks is the most links that an update follows from the path of a golden
// file to the file that it writes.
const maxLinks = 255

// update is registered once, so a package using this can be run with
// -update. One registration in this package keeps one flag for the whole
// test binary.
var update = flag.Bool("update", false,
	"rewrite golden files to match current output")

// ShouldUpdate reports whether -update was passed. Pass it as the
// update argument to let a run rewrite its golden files.
//
//	golden.Match(t, "response.json", got, golden.ShouldUpdate())
//
// It is a function, so a caller cannot read the flag before [flag.Parse]
// has run, which under `go test` happens before the first test.
//
// # Allocation contract
//
// ShouldUpdate allocates nothing.
func ShouldUpdate() bool { return *update }

// The ids of the golden comparisons, which their records state.
const (
	matchID     = "golden-match"
	matchAtID   = "golden-match-at"
	jsonFieldID = "golden-match-json-field"
)

// The fields of the detail of a golden comparison's record.
const (
	// wantField is the content of the golden file, and nil for a file that
	// does not exist.
	wantField = "want"
	// gotField is the content that the comparison received.
	gotField = "got"
)

// call is one call of a golden comparison: its seat, the operation that
// its faults state, the assertion and the contract that its record
// states, and the path of its golden file.
type call struct {
	tb       assert.TB
	op       string
	id       string
	contract string
	path     string
}

// newFileCall returns the call of the comparison id of the operation op
// on the golden file at path.
func newFileCall(tb assert.TB, op, id, path string) call {
	return call{
		tb: tb, op: op, id: id, path: path,
		contract: fmt.Sprintf("the golden file %s matches the output, and -update writes it", path),
	}
}

// pass reports that the call passed.
func (c call) pass() {
	c.tb.Helper()
	matcher.Pass(c.tb, matcher.Fatal, c.id, c.contract)
}

// fail reports that the call failed with detail.
func (c call) fail(detail map[string]any) {
	c.tb.Helper()
	matcher.Fail(c.tb, matcher.Fatal, c.id, c.contract, detail)
}

// fault reports that the call ended without a verdict, because of err, with
// the reason that format and args state.
func (c call) fault(err error, format string, args ...any) {
	c.tb.Helper()
	matcher.Fault(c.tb, matcher.Fatal, c.id, c.contract, fault.In(c.op, fault.New(format, args...).Because(err)))
}

// write writes content as the call's golden file, in the directory that it
// creates when the directory is missing, and passes. It writes a temporary
// file beside the file of the golden path, and renames it over that file. A
// directory that cannot be created leaves the file unwritten, and the fault
// states both errors.
func (c call) write(content string) {
	c.tb.Helper()
	dirErr := os.MkdirAll(filepath.Dir(c.path), dirPerm)
	path := target(c.path)
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err == nil {
		err = replace(f, path, content)
	}
	if err != nil {
		c.fault(errors.Join(dirErr, err), "the golden file cannot be written")
		return
	}
	c.pass()
}

// target returns the file that a write of path writes: the end of the links
// that path starts, or path itself when it is no link.
func target(path string) string {
	for range maxLinks {
		next, err := os.Readlink(path)
		if err != nil {
			break
		}
		if !filepath.IsAbs(next) {
			next = filepath.Join(filepath.Dir(path), next)
		}
		path = next
	}
	return path
}

// replace writes content into the temporary file f, and renames f over path.
// A write that fails removes f and leaves path as it was. The file keeps the
// mode of the file that it replaces, and a new file gets filePerm.
func replace(f *os.File, path, content string) error {
	mode := filePerm
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	_, err := f.WriteString(content)
	err = errors.Join(err, f.Close(), os.Chmod(f.Name(), mode))
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		return errors.Join(err, os.Remove(f.Name()))
	}
	return nil
}

// Match compares got against testdata/golden/name, relative to the
// test's own directory.
//
// The file is the assertion. A missing file fails the call while update is
// false, and update writes the file and passes. A failure is a record of
// golden-match, with the file's content as want, nil for a missing file,
// and the output as got, both scrubbed. Its contract states the file and
// the -update flag.
//
// scrubbers are applied to both sides before the comparison, so
// content that differs between runs does not defeat it.
//
// A golden file that cannot be read or written ends the call with a fault,
// which stops the test.
//
// # Allocation contract
//
// A passing comparison of a short file without scrubbers allocates 10
// times: the path of the file, and what [MatchAt] allocates.
func Match(tb assert.TB, name string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	matchFile(newFileCall(tb, "golden.Match", matchID, filepath.Join(conventionalDir, name)), got, update, scrubbers)
}

// MatchAt is [Match] with the path taken as given, for a golden file
// outside the conventional directory. A failure is a record of
// golden-match-at.
//
// # Allocation contract
//
// A passing comparison of a short file without scrubbers allocates 9
// times. Each scrubber allocates what its replacements allocate.
func MatchAt(tb assert.TB, path string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	matchFile(newFileCall(tb, "golden.MatchAt", matchAtID, path), got, update, scrubbers)
}

// matchFile compares got against the golden file of c, and reports the
// verdict of c.
func matchFile(c call, got []byte, update bool, scrubbers []Scrubber) {
	c.tb.Helper()

	mine := scrub(string(got), scrubbers)
	raw, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		if update {
			c.write(mine)
			return
		}
		c.fail(map[string]any{wantField: nil, gotField: mine})
		return
	}
	if err != nil {
		c.fault(err, "the golden file cannot be read")
		return
	}

	theirs := scrub(string(raw), scrubbers)
	if mine == theirs {
		c.pass()
		return
	}
	if update {
		c.write(mine)
		return
	}
	c.fail(map[string]any{wantField: theirs, gotField: mine})
}
