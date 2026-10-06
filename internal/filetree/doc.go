// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package filetree reads, compares, records and writes trees of files: the
// part of the files assertions and of golden.MatchTree that both share.
//
// A [Tree] is a set of entries at slash-separated paths relative to its
// root, each an [Entry] of a [Kind]. [Tree.Check] refuses a tree that breaks
// a rule of the definition. [Read] reads a tree from a file system and
// follows no link, and [ReadPath] reads the entry at one path of the
// operating system. [Differing] compares a tree read with a wanted one, and
// [NewRecord] builds the record of a comparison that fails. [Write] writes a
// tree into a directory as a workspace does, and [Update] makes a directory
// equal a tree as an update of a golden tree does. [Tree.MarshalJSON] writes
// the tree literal of a record, and [Decode] reads the tree literal of an
// input.
//
// # Platforms
//
// A file system that records no permission bits reads no mode and no
// execute bit, and a comparison then compares neither. [ModesUnrecorded]
// returns the fault of an assertion that reads a mode there. The build for
// Windows takes the file systems of its platform for such file systems, and
// every other build takes its file systems for ones that record the nine
// permission bits.
//
// Windows records one permission bit, the owner's write bit of a file, as
// the file's read-only attribute. [Write] sets the attribute on each file
// whose mode lacks that bit, and sets none on a directory, where Windows does
// not honour it. Windows stores the target of a link with backslashes, and
// [Read] and [ReadPath] return it with slashes, as a tree states it.
//
// # Dependency position
//
// Imports fault, literal for the literals of a record's null and its count,
// matcher for the record's text, and the standard library. files and golden
// import it.
package filetree
