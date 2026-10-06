// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"errors"
	"io/fs"
	"os"
	"syscall"

	"go.dokimi.dev/assert/internal/fault"
)

// Read returns the tree in fsys: every entry below its root, each link as a
// link with its target, and the permission bits of each file and each
// directory as the platform records them. With modes false, it states the
// execute bit of each file alone, as an update of a golden tree reads the
// golden tree.
//
// # Errors
//
// It returns a fault for a root that is no directory, and one whose cause is
// the error of the file system for a tree that cannot be read. An entry
// that is no file, directory or link, and a link in an fs.FS that does not
// implement fs.ReadLinkFS, have a fault at their path.
//
// # Allocation contract
//
// Read allocates the tree, the content of each file, and what the walk of
// fsys allocates.
func Read(fsys fs.FS, modes bool) (Tree, error) {
	t := Tree{}
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fault.New("the tree cannot be read").Because(err)
		}
		if path == "." {
			if !d.IsDir() {
				return fault.New("the root of the tree is no directory")
			}
			return nil
		}
		e, err := readEntry(fsys, path, d, modes)
		if err != nil {
			return fault.At(err, fault.Key(path))
		}
		t[path] = e
		return nil
	})
	return t, err
}

// readEntry returns the entry at path in fsys, of the directory entry d.
func readEntry(fsys fs.FS, path string, d fs.DirEntry, modes bool) (Entry, error) {
	info, err := d.Info()
	var e Entry
	if err == nil {
		e, err = entryOf(info.Mode(), modes)
	}
	if err == nil && e.Kind == File {
		var content []byte
		content, err = fs.ReadFile(fsys, path)
		e.Content = string(content)
	}
	if err == nil && e.Kind == Link {
		e.Target, err = fs.ReadLink(fsys, path)
	}
	if err != nil {
		return Entry{}, readFault(err)
	}
	return e, nil
}

// readFault returns err, an error of a read of an entry, as a fault: the
// fault of an entry that cannot be read, whose cause is err, and a fault of
// this package as it is.
func readFault(err error) error {
	if _, ok := errors.AsType[*fault.Error](err); ok {
		return err
	}
	return fault.New("the entry cannot be read").Because(err)
}

// ReadPath returns the entry at path, a path of the operating system, and
// the zero Entry when nothing is there. Nothing is at a path below a file.
// It follows no link at path, and the platform resolves a link among the
// names before the last one. It reads the content of a file with content
// set, and the permission bits of a file or a directory as the platform
// records them.
//
// # Errors
//
// It returns a fault for an entry that is no file, directory or link, and one
// whose cause is the error of the file system for any other error than one
// that finds nothing at path.
//
// # Allocation contract
//
// ReadPath allocates the information of the entry, and the content of a file
// or the target of a link that it reads.
func ReadPath(path string, content bool) (Entry, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return Entry{}, nil
	}
	var e Entry
	if err == nil {
		e, err = entryOf(info.Mode(), true)
	}
	if err == nil && e.Kind == File && content {
		var data []byte
		data, err = os.ReadFile(path)
		e.Content = string(data)
	}
	if err == nil && e.Kind == Link {
		e.Target, err = os.Readlink(path)
	}
	if err != nil {
		return Entry{}, readFault(err)
	}
	return e, nil
}

// entryOf returns the entry of the kind that mode states, with the
// permission bits that the platform records: the nine bits with modes set,
// and the execute bit of a file alone without. A link has no mode. It
// returns a fault for a mode of a device, a pipe, a socket or any other
// irregular file.
func entryOf(mode fs.FileMode, modes bool) (Entry, error) {
	e := Entry{Mode: mode.Perm() & recordedBits, Stated: modes && recordedBits != 0}
	switch mode.Type() {
	case 0:
		e.Kind = File
	case fs.ModeDir:
		e.Kind = Dir
	case fs.ModeSymlink:
		return Entry{Kind: Link}, nil
	default:
		return Entry{}, fault.New("the entry is no file, directory or link")
	}
	if !modes {
		e.Mode &= executeBit(e.Kind)
	}
	return e, nil
}

// executeBit returns the permission bit that a tree without modes states of
// an entry of kind k: the execute bit of a file, and none of a directory.
func executeBit(k Kind) fs.FileMode {
	if k == File {
		return OwnerExecute
	}
	return 0
}
