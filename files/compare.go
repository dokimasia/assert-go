// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files

import (
	"io/fs"
	"runtime"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
)

// The ids of the comparisons of trees, which their records state, and the
// operations that their faults name.
const (
	equalID     = "tree-equal"
	containsID  = "tree-contains"
	unchangedID = "tree-unchanged"
	equalOp     = "files.Equal"
	containsOp  = "files.Contains"
	unchangedOp = "files.Unchanged"
)

// callerFrames is the most frames that the call site of a failing
// comparison is searched in.
const callerFrames = 64

// The arguments that name what a comparison reads and compares, named
// because a call that passes the wrong bool tests the other path.
const (
	// readingModes reads the permission bits of each file and directory.
	readingModes = true
	// everyPath compares every path of both trees.
	everyPath = false
	// wantedPaths compares the paths of the wanted tree alone.
	wantedPaths = true
)

// The text writer takes the sentence of the records of the comparisons of
// trees from this package, which registers it while it initialises.
func init() {
	matcher.RegisterSentence(filetree.Sentence, equalID, containsID, unchangedID)
}

// Equal checks that the tree in got has the entries of want, no more and no
// fewer, each as want states it, and stops the test with a record of
// tree-equal when it does not. The record's contract is msg.
//
//	files.Equal(t, os.DirFS(dir), files.Tree{
//		"a/a.go": files.Text("package a\n\nfunc New() {}\n"),
//	}, "the rename rewrites the declaration")
//
// The comparison reads got and follows no link. It compares a mode only where
// want states one, and otherwise the owner's execute bit of a file. The
// record states want and got, the trees of the entries at the first 64 paths
// that differ, and differences, the number of those paths, as int. A missing
// entry is in want alone, an extra one in got alone, and a changed one in
// both. An entry of got states its mode only where want states one.
//
// # Errors
//
// It ends the call with a fault for a want that breaks a rule of a tree, for
// an entry of got that is no file, directory or link, for a link in an fs.FS
// that does not implement fs.ReadLinkFS, and for a tree that cannot be read.
//
// # Allocation contract
//
// Equal allocates the tree that it reads with the content of each file, the
// wanted tree with its implied directories, and what the walk of got
// allocates. A passing call on a tree in memory of a file, a script and a
// link allocates 43 times.
func Equal(tb assert.TB, got fs.FS, want Tree, msg string) {
	tb.Helper()
	compareRead(tb, equalOp, equalID, msg, want, got, everyPath)
}

// Contains checks that the tree in got has every entry of want, each as want
// states it, and stops the test with a record of tree-contains when it does
// not. got may have more entries. The comparison, the record and the faults
// are those of [Equal], and a path that want lacks is no difference.
//
// # Allocation contract
//
// Contains allocates what [Equal] allocates.
func Contains(tb assert.TB, got fs.FS, want Tree, msg string) {
	tb.Helper()
	compareRead(tb, containsOp, containsID, msg, want, got, wantedPaths)
}

// Unchanged calls fn, and checks that the tree in fsys after the call equals
// the tree in it before the call, modes included. It stops the test with a
// record of tree-unchanged when the trees differ, with the tree before the
// call as the wanted one. A write of the bytes that a file has, and a change
// of a timestamp, leave the tree unchanged.
//
//	files.Unchanged(t, os.DirFS(dir), func() { _ = migrate.DryRun(dir) }, "a dry run writes nothing")
//
// A panic of fn ends the call with that panic, and compares nothing. The
// record and the faults are those of [Equal].
//
// # Allocation contract
//
// Unchanged allocates the two trees that it reads, and what fn allocates. A
// passing call on the tree of the contract of [Equal] allocates 78 times.
func Unchanged(tb assert.TB, fsys fs.FS, fn func(), msg string) {
	tb.Helper()
	run := matcher.Begin(tb)
	before, err := filetree.Read(fsys, readingModes)
	var after filetree.Tree
	if err == nil {
		fn()
		after, err = filetree.Read(fsys, readingModes)
	}
	if err != nil {
		run.Fault(matcher.Fatal, unchangedID, msg, fault.In(unchangedOp, err))
		return
	}
	report(run, unchangedID, msg, before, after, filetree.Differing(before, after, everyPath))
}

// compareRead reads got and reports the verdict of the comparison id, whose
// faults name op, of the tree in got with want.
func compareRead(tb assert.TB, op, id, msg string, want Tree, got fs.FS, contains bool) {
	tb.Helper()
	run := matcher.Begin(tb)
	wanted := want.internal()
	err := wanted.Check()
	var read filetree.Tree
	if err == nil {
		read, err = filetree.Read(got, readingModes)
	}
	if err != nil {
		run.Fault(matcher.Fatal, id, msg, fault.In(op, err))
		return
	}
	report(run, id, msg, wanted, read, filetree.Differing(wanted, read, contains))
}

// report reports a pass of the comparison id when no path differs, and
// otherwise a failure whose record states want and got at paths, located at
// the caller's code.
func report(run matcher.Running, id, msg string, want, got filetree.Tree, paths []string) {
	if len(paths) == 0 {
		run.Pass(matcher.Fatal, id, msg)
		return
	}
	record := filetree.NewRecord(want, got, paths)
	var pcs [callerFrames]uintptr
	where := matcher.CallerWhere(pcs[:runtime.Callers(1, pcs[:])])
	run.FailRun(matcher.Fatal, assert.Failure{Assertion: id, Contract: msg, Detail: record.Fields(), Where: where},
		record)
}
