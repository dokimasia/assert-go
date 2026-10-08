// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files

import (
	"cmp"
	"io/fs"
	"unicode/utf8"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
)

// The ids of the assertions of one path, which their records state, and the
// operations that their faults name.
const (
	absentID     = "path-absent"
	isFileID     = "is-file"
	isDirID      = "is-dir"
	linksToID    = "links-to"
	hasContentID = "has-content"
	hasModeID    = "has-mode"
	absentOp     = "files.Absent"
	isFileOp     = "files.IsFile"
	isDirOp      = "files.IsDir"
	linksToOp    = "files.LinksTo"
	hasContentOp = "files.HasContent"
	hasModeOp    = "files.HasMode"
)

// The detail fields of the records of the assertions of one path.
const (
	wantField = "want"
	gotField  = "got"
	kindField = "kind"
)

// pathCall is one call of an assertion of one path: its seat, the operation
// that its faults name, the assertion, and the contract.
type pathCall struct {
	tb       assert.TB
	op       string
	id       string
	contract string
}

// check reads the entry at path, with its content when content is set, and
// reports the verdict of the call: a fault for refused, the fault of an
// argument that the call refuses, and for an entry that cannot be read, a
// failure with the detail that detail returns for the entry, and a pass
// when detail returns nil.
func (c pathCall) check(path string, content bool, refused error, detail func(filetree.Entry) map[string]any) {
	c.tb.Helper()
	e, err := filetree.ReadPath(path, content)
	if err = cmp.Or(refused, err); err != nil {
		matcher.Fault(c.tb, matcher.Fatal, c.id, c.contract, fault.In(c.op, err))
		return
	}
	if d := detail(e); d != nil {
		matcher.Fail(c.tb, matcher.Fatal, c.id, c.contract, d)
		return
	}
	matcher.Pass(c.tb, matcher.Fatal, c.id, c.contract)
}

// Absent checks that nothing is at path, a path of the operating system: no
// file, no directory and no link, a link whose target is missing included.
// It stops the test with a record of path-absent when an entry is there,
// whose got is the entry's kind: file, directory or link. The record's
// contract is msg. Nothing is at a path below a file.
//
// Each assertion of one path reads the entry at the path itself, and follows
// no link there. The platform resolves the names before the last one, and
// follows a link among them.
//
// # Errors
//
// It ends the call with a fault for an entry that is no file, directory or
// link, and for any error of the file system but one that finds nothing at
// path.
//
// # Allocation contract
//
// Absent allocates what os.Lstat allocates: 3 allocations for a path where
// nothing is.
func Absent(tb assert.TB, path, msg string) {
	tb.Helper()
	pathCall{tb, absentOp, absentID, msg}.check(path, false, nil, func(e filetree.Entry) map[string]any {
		if e.Kind == filetree.None {
			return nil
		}
		return map[string]any{gotField: kindOf(e)}
	})
}

// IsFile checks that a file is at path. A link to a file is a link. It stops
// the test with a record of is-file when no file is there, whose got is the
// kind of the entry there, or nil for none. The reading and the faults are
// those of [Absent].
//
// # Allocation contract
//
// IsFile allocates what os.Lstat allocates for a file: twice.
func IsFile(tb assert.TB, path, msg string) {
	tb.Helper()
	pathCall{tb, isFileOp, isFileID, msg}.check(path, false, nil, kindIs(filetree.File))
}

// IsDir checks that a directory is at path. A link to a directory is a link.
// It stops the test with a record of is-dir when no directory is there,
// whose got is the kind of the entry there, or nil for none. The reading and
// the faults are those of [Absent].
//
// # Allocation contract
//
// IsDir allocates what os.Lstat allocates for a directory: twice.
func IsDir(tb assert.TB, path, msg string) {
	tb.Helper()
	pathCall{tb, isDirOp, isDirID, msg}.check(path, false, nil, kindIs(filetree.Dir))
}

// kindIs returns the detail of an assertion that an entry of kind k is at a
// path: nil for such an entry, and the kind of any other entry as got.
func kindIs(k filetree.Kind) func(filetree.Entry) map[string]any {
	return func(e filetree.Entry) map[string]any {
		if e.Kind == k {
			return nil
		}
		return map[string]any{gotField: kindOf(e)}
	}
}

// LinksTo checks that a symbolic link whose target is target is at path. The
// target compares as text, read with slashes as separators on every
// platform, and nothing follows the link. It stops the test
// with a record of links-to when no such link is there: want is target, got
// the target of the link there, or nil when no link is there, and kind the
// kind of the entry there, or nil for none. The reading and the faults are
// those of [Absent].
//
// # Allocation contract
//
// LinksTo allocates what os.Lstat and os.Readlink allocate for a link: 5
// allocations.
func LinksTo(tb assert.TB, path, target, msg string) {
	tb.Helper()
	pathCall{tb, linksToOp, linksToID, msg}.check(path, false, nil, func(e filetree.Entry) map[string]any {
		if e.Kind == filetree.Link && e.Target == target {
			return nil
		}
		var got any
		if e.Kind == filetree.Link {
			got = e.Target
		}
		return map[string]any{wantField: target, gotField: got, kindField: kindOf(e)}
	})
}

// HasContent checks that a file whose bytes equal want is at path. It
// compares bytes, so a file that ends in \r\n does not equal text that ends
// in \n. It stops the test with a record of has-content when no such file
// is there: want is want, got the file's content, or nil when no file is
// there, and kind the kind of the entry there, or nil for none. A content is
// a string when it is valid UTF-8, and a []byte otherwise. The reading and
// the faults are those of [Absent].
//
// # Allocation contract
//
// HasContent allocates what os.Lstat and os.ReadFile allocate, and the
// content of the file as the text that it compares: 8 allocations for a
// short file.
func HasContent(tb assert.TB, path, want, msg string) {
	tb.Helper()
	pathCall{tb, hasContentOp, hasContentID, msg}.check(path, true, nil, func(e filetree.Entry) map[string]any {
		if e.Kind == filetree.File && e.Content == want {
			return nil
		}
		var got any
		if e.Kind == filetree.File {
			got = textOrBytes(e.Content)
		}
		return map[string]any{wantField: textOrBytes(want), gotField: got, kindField: kindOf(e)}
	})
}

// HasMode checks that a file or a directory whose nine permission bits are
// want is at path. A link has no mode. It stops the test with a record of
// has-mode when no such entry is there: want is want, got the permission
// bits of the file or the directory there, or nil when neither is there, and
// kind the kind of the entry there, or nil for none. want and got are
// fs.FileMode values. The reading is that of [Absent].
//
//	files.HasMode(t, filepath.Join(home, ".config/tool/key"), 0o600, "the key is private")
//
// # Errors
//
// It ends the call with a fault for a want with a bit beyond the nine
// permission bits, and on a file system that records no permission bits,
// where the bits that it would compare are not stored. The other faults are
// those of [Absent].
//
// # Allocation contract
//
// HasMode allocates what os.Lstat allocates for a file: twice.
func HasMode(tb assert.TB, path string, want fs.FileMode, msg string) {
	tb.Helper()
	var refused error
	if want&^fs.ModePerm != 0 {
		refused = fault.New("the mode %O has a bit beyond the nine permission bits", uint32(want))
	}
	refused = cmp.Or(refused, filetree.ModesUnrecorded())
	pathCall{tb, hasModeOp, hasModeID, msg}.check(path, false, refused, func(e filetree.Entry) map[string]any {
		if (e.Kind == filetree.File || e.Kind == filetree.Dir) && e.Mode == want {
			return nil
		}
		var got any
		if e.Kind == filetree.File || e.Kind == filetree.Dir {
			got = e.Mode
		}
		return map[string]any{wantField: want, gotField: got, kindField: kindOf(e)}
	})
}

// kindOf returns the kind of e as a record states it, and nil for no entry.
func kindOf(e filetree.Entry) any {
	if e.Kind == filetree.None {
		return nil
	}
	return e.Kind.Name()
}

// textOrBytes returns content as a string when it is valid UTF-8, and as a
// []byte otherwise, so that the record states it as text or as bytes.
func textOrBytes(content string) any {
	if utf8.ValidString(content) {
		return content
	}
	return []byte(content)
}
