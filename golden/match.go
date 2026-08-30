// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go.dokimi.dev/assert"
)

// conventionalDir is where [Match] looks, relative to the test's own
// directory. [MatchAt] takes a path instead.
const conventionalDir = "testdata/golden"

// Permissions for a golden file this package writes, and for the
// directory it creates to hold one.
const (
	filePerm = 0o644
	dirPerm  = 0o755
)

// update is registered once, so a package using this can be run with
// -update. Registering it here rather than in each test keeps one flag
// for the whole binary.
var update = flag.Bool("update", false,
	"rewrite golden files to match current output")

// ShouldUpdate reports whether -update was passed. Pass it as the
// update argument to let a run rewrite its golden files.
//
//	golden.Match(t, "response.json", got, golden.ShouldUpdate())
//
// It is a function rather than a bool so a caller cannot read the flag
// before [flag.Parse] has run, which under `go test` happens before
// the first test.
func ShouldUpdate() bool { return *update }

// Match compares got against testdata/golden/name, relative to the
// test's own directory.
//
// The file is the assertion. When it does not exist and update is
// false, that is a failure naming the flag that would create it; when
// update is true, it is written and the test passes.
//
// scrubbers are applied to both sides before the comparison, so
// content that differs between runs does not defeat it.
func Match(tb assert.TB, name string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	MatchAt(tb, filepath.Join(conventionalDir, name), got, update, scrubbers...)
}

// MatchAt is [Match] with the path taken as given, for a golden file
// that lives outside the conventional directory.
func MatchAt(tb assert.TB, path string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()

	mine := scrub(string(got), scrubbers)

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if update {
			write(tb, path, mine)
			return
		}
		tb.Fatalf("%s: the golden file does not exist; run the test with -update to create it", path)
		return
	}
	assert.NoError(tb, err, fmt.Sprintf("%s: the golden file can be read", path))

	theirs := scrub(string(raw), scrubbers)
	if update {
		if mine != theirs {
			write(tb, path, mine)
		}
		return
	}

	assert.Equal(tb, mine, theirs,
		fmt.Sprintf("%s: output matches the golden file; "+
			"read the diff before running with -update", path))
}

// write records content as the golden file at path, creating the
// directory when it is missing.
func write(tb assert.TB, path, content string) {
	tb.Helper()

	assert.NoError(tb, os.MkdirAll(filepath.Dir(path), dirPerm),
		fmt.Sprintf("%s: the golden directory can be created", path))
	assert.NoError(tb, os.WriteFile(path, []byte(content), filePerm),
		fmt.Sprintf("%s: the golden file can be written", path))
}
