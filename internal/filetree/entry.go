// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"strconv"
)

// Kind is the kind of an entry.
type Kind uint8

const (
	// None is no entry: the kind of the zero [Entry], and of the entry at a
	// path where nothing is.
	None Kind = iota
	// File is a file, with its content.
	File
	// Dir is a directory.
	Dir
	// Link is a symbolic link, with its target.
	Link
)

// The names of the kinds, as a record states them.
var kindNames = [...]string{None: "", File: "file", Dir: "directory", Link: "link"}

// Name returns the kind as a record states it: file, directory or link, and
// the empty string for None. It allocates nothing.
func (k Kind) Name() string {
	return kindNames[k]
}

// OwnerExecute is the permission bit that lets the owner execute a file.
const OwnerExecute fs.FileMode = 0o100

// ContentLimit is the longest content that a record and a text state in
// full, in bytes. A longer content is stated by its size and its digest.
const ContentLimit = 65536

// The modes that a workspace sets where a tree states none.
const (
	// FileMode is the mode of a file.
	FileMode fs.FileMode = 0o644
	// ExecutableMode is the mode of a file that its owner may execute.
	ExecutableMode fs.FileMode = 0o755
	// DirMode is the mode of a directory.
	DirMode fs.FileMode = 0o755
)

// Entry is one entry of a tree: its kind, the content of a file, the
// target of a link, and the permission bits of a file or a directory.
//
// Mode states the nine permission bits when Stated is set. When it is not,
// Mode is [OwnerExecute] for a file that its owner may execute, and 0 for
// any other entry. A link has no mode. Two entries are equal under == when
// they state the same entry.
type Entry struct {
	// Kind is the kind of the entry.
	Kind Kind
	// Content is the content of a file.
	Content string
	// Target is the target of a link, as text that no rule of a path limits.
	Target string
	// Mode is the entry's permission bits, as Stated states.
	Mode fs.FileMode
	// Stated reports whether Mode states all nine permission bits.
	Stated bool
}

// Executable reports whether the owner may execute e. It allocates nothing.
func (e Entry) Executable() bool {
	return e.Kind == File && e.Mode&OwnerExecute != 0
}

// GoString returns e as the Go expression of package files that states it:
// files.Text("a\n") for a file, files.Executable for a file that its owner
// may execute, files.Dir() and files.Link("a.txt"), with WithMode(0o600)
// after an entry that states its mode. A content longer than [ContentLimit]
// is stated by its size and its digest in place of the text. The zero
// Entry is none.
//
// # Allocation contract
//
// GoString allocates the text that it builds and the text that it returns:
// two allocations for a file of text that a Go string literal states
// without escapes.
func (e Entry) GoString() string {
	buf := make([]byte, 0, len(e.Content)+len(e.Target)+len(`files.Executable("").WithMode(0o000)`))
	switch e.Kind {
	case File:
		if e.Executable() && !e.Stated {
			buf = append(buf, "files.Executable("...)
		} else {
			buf = append(buf, "files.Text("...)
		}
		buf = appendQuoted(buf, e.Content)
		buf = append(buf, ')')
	case Dir:
		buf = append(buf, "files.Dir()"...)
	case Link:
		buf = append(buf, "files.Link("...)
		buf = strconv.AppendQuote(buf, e.Target)
		buf = append(buf, ')')
	default:
		return "none"
	}
	if e.Stated {
		buf = append(buf, ".WithMode(0o"...)
		buf = strconv.AppendUint(buf, uint64(e.Mode), 8)
		buf = append(buf, ')')
	}
	return string(buf)
}

// appendQuoted appends content to buf as a quoted Go string, and its size and
// its digest for a content longer than [ContentLimit].
func appendQuoted(buf []byte, content string) []byte {
	if len(content) > ContentLimit {
		return fmt.Appendf(buf, "<%d bytes, %s>", len(content), digest(content))
	}
	return strconv.AppendQuote(buf, content)
}

// digest returns the SHA-256 digest of content, as a record states it.
func digest(content string) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content)))
}
