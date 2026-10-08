// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"unicode/utf8"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
)

// The id of golden-match-tree, which its records state, and the operation
// that its faults name.
const (
	matchTreeID = "golden-match-tree"
	matchTreeOp = "golden.MatchTree"
)

// callerFrames is the most frames that the call site of a failing comparison
// is searched in.
const callerFrames = 64

// The text writer takes the sentence of the records of golden-match-tree
// from the package filetree, which states the records of every comparison of
// trees. This package registers it while it initialises.
func init() {
	matcher.RegisterSentence(filetree.Sentence, matchTreeID)
}

// MatchTree compares the tree in got with the golden tree in the directory
// testdata/golden/name, relative to the test's own directory, and stops the
// test with a record of golden-match-tree when they differ. The name follows
// the rules of a path: names joined by slashes, none of them empty, . or ..,
// so it cannot leave the conventional directory.
//
//	golden.MatchTree(t, "api", os.DirFS(out), golden.ShouldUpdate())
//
// The comparison follows no link. It reads no mode of the golden tree, and
// compares the owner's execute bit of each file, the one bit that git
// records. scrubbers apply to the content of every file that is valid UTF-8
// text, on both sides. The record states want and got, the trees of the
// entries at the first 64 paths that differ, and differences, the number of
// those paths. A missing golden directory fails with want nil, got the first
// 64 entries of the output, and differences the number of its entries.
//
// With update, MatchTree makes the golden directory equal the output and
// passes. It removes each entry that the output lacks, writes each entry
// that the directory lacks or has in another form, and writes the scrubbed
// content. It removes a link, and never the entry that the link points to.
//
// # Errors
//
// It ends the call with a fault for a name that is no path, for an entry of
// either tree that is no file, directory or link, and for a tree or a golden
// directory that cannot be read or written. The fault stops the test.
//
// # Allocation contract
//
// MatchTree allocates the two trees that it reads, their comparison, and
// what the walks of the two trees allocate. A passing comparison of an
// output in memory of four entries with its golden tree allocates 122
// times.
func MatchTree(tb assert.TB, name string, got fs.FS, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	dir := filepath.Join(conventionalDir, filepath.FromSlash(name))
	contract := fmt.Sprintf("the golden tree %s matches the output, and -update writes it", dir)
	run := matcher.Begin(tb)
	ended := func(err error) { run.Fault(matcher.Fatal, matchTreeID, contract, fault.In(matchTreeOp, err)) }
	output, err := filetree.Read(got, true)
	if err = cmp.Or(filetree.CheckPath(name), err); err != nil {
		ended(err)
		return
	}
	golden, err := filetree.Read(os.DirFS(dir), false)
	missing := errors.Is(err, fs.ErrNotExist)
	if err != nil && !missing {
		ended(err)
		return
	}
	output, golden = scrubbed(output, scrubbers), scrubbed(golden, scrubbers)
	paths := filetree.Differing(golden, output, false)
	if update && (missing || len(paths) > 0) {
		if err := filetree.Update(dir, output); err != nil {
			ended(err)
			return
		}
	}
	if update || (!missing && len(paths) == 0) {
		run.Pass(matcher.Fatal, matchTreeID, contract)
		return
	}
	var record filetree.Record
	if missing {
		record = filetree.Missing(output)
	} else {
		record = filetree.NewRecord(golden, output, paths)
	}
	var pcs [callerFrames]uintptr
	where := matcher.CallerWhere(pcs[:runtime.Callers(1, pcs[:])])
	run.FailRun(matcher.Fatal, assert.Failure{
		Assertion: matchTreeID, Contract: contract, Detail: record.Fields(), Where: where,
	}, record)
}

// scrubbed returns t with scrubbers applied to the content of each file that
// is valid UTF-8 text.
func scrubbed(t filetree.Tree, scrubbers []Scrubber) filetree.Tree {
	for path, e := range t {
		if e.Kind == filetree.File && utf8.ValidString(e.Content) {
			e.Content = scrub(e.Content, scrubbers)
			t[path] = e
		}
	}
	return t
}
