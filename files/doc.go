// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package files builds trees of files for a test, and checks the files that
// the code under test reads and writes.
//
// A [Tree] states files with their content, directories and symbolic links,
// each at a slash-separated path relative to the tree's root. [Text],
// [Bytes], [Executable], [Dir] and [Link] state its entries, and
// [Entry.WithMode] states the permission bits of a file or a directory where
// they are part of the contract.
//
//	dir := files.Workspace(t, files.Tree{
//		"go.mod": files.Text("module example.com/a\n"),
//		"a/a.go": files.Text("package a\n\nfunc Old() {}\n"),
//		"keys/id": files.Text("secret\n").WithMode(0o600),
//	})
//
// [Workspace] writes a tree into a directory of the test's own, and [Write]
// writes a tree over the entries of a directory that exists, such as an edit
// between two runs of the code under test. [Equal], [Contains] and
// [Unchanged] compare a tree read from an fs.FS with a wanted one, so they
// read a directory through os.DirFS and a tree in memory through
// fstest.MapFS. [Absent], [IsFile], [IsDir], [LinksTo],
// [HasContent] and [HasMode] check the entry at one path of the operating
// system, and [Read] returns the content of a file, so that the text
// assertions apply to it.
//
// # Comparison rules
//
// A tree read from a directory follows no link, and states each file's and
// each directory's permission bits. A comparison compares a mode only where
// the wanted entry states one, and otherwise the owner's execute bit of a
// file. Content compares as bytes, so a file that ends in \r\n does not equal
// text that ends in \n.
//
// # Failure semantics
//
// Every assertion here stops the test, as the golden comparisons do. The
// record of [Equal], [Contains] and [Unchanged] states want and got, trees of
// the entries at the first 64 paths that differ, and differences, the number
// of those paths. Each tree prints as the Go expressions of this package that
// state its entries, and its call record states it as the tree literal of the
// definition. A file longer than 65,536 bytes is stated by its size and its
// digest there.
//
// A tree that breaks a rule of a tree, an entry that is no file, directory or
// link, and an error of the file system end the call with a fault. A call
// record states the fault's text under the verdict error.
//
// # Platforms
//
// A file system that records no permission bits, as on Windows, reads no
// mode and no execute bit: a comparison then compares neither, and [HasMode]
// ends the call with a fault. Windows records one permission bit, the
// owner's write bit of a file, as the file's read-only attribute. [Workspace]
// and [Write] set the attribute on each file whose mode lacks that bit, and
// set none on a directory, where Windows does not honour it. Windows stores the target of
// a link with backslashes, and a tree reads it with slashes, as [Link] states
// it, so [LinksTo] compares a target with slashes.
//
// # Allocation contracts
//
// The allocation contract of each function counts one call on a seat that
// writes no call record, such as a test's seat while recording is off. A
// failing call allocates its record and its text as well, and a recorded call
// allocates its call record. Every call that reads or writes a file allocates
// what the file system's calls allocate.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert, its internal fault, filetree and matcher, and
// the standard library's cmp, fmt, io/fs, runtime, testing and unicode/utf8.
package files
