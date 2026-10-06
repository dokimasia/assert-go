// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
)

// The permission bits with which a write creates a file and a directory,
// before it sets the entry's mode.
const (
	createFile fs.FileMode = 0o600
	createDir  fs.FileMode = 0o700
)

// step is one operation of a write: the path that it writes, and the
// operation on the root of the directory.
type step struct {
	path string
	run  func(root *os.Root) error
}

// Write writes t into dir, a directory that exists, as a workspace does. It
// creates every entry and writes over none: a file of its content, a
// directory, and a link of its target, inside or outside dir, which it never
// follows. It writes through os.Root, so no entry is written outside dir.
//
// Each file and each directory gets its stated mode, and where it states
// none [FileMode], [ExecutableMode] or [DirMode]. Write sets the mode of a
// file after it writes the file, and the mode of a directory after every
// entry below it, so the umask cannot change a mode, and the mode of a
// directory cannot stop a write below it. On Windows only the owner's write
// bit of a file's mode takes effect, as the file's read-only attribute, and
// the mode of a directory takes none.
//
// # Errors
//
// It returns the fault of [Tree.Check] for a tree that breaks a rule of a
// tree, before it writes anything. It returns a fault at the path of the
// first entry that cannot be written, whose cause is the error of the file
// system, such as an entry that the file system maps to one already written.
//
// # Allocation contract
//
// Write allocates the tree with its implied directories, an operation for
// each entry, and what the file system's calls allocate.
func Write(dir string, t Tree) error {
	if err := t.Check(); err != nil {
		return err
	}
	return apply(dir, creating(t.Full(), nil))
}

// Update makes dir equal t, as an update of a golden tree does. It creates
// dir and its parents when they are missing. It removes each entry that t
// lacks or states in another form, and writes each entry that dir then
// lacks, as [Write] writes it. It removes a link, and never the entry that
// the link points to. It compares no mode, and writes each file and each
// directory with the mode that Write sets where a tree states none.
//
// # Errors
//
// It returns the fault of [Tree.Check] for a tree that breaks a rule of a
// tree, a fault whose cause is the error of the file system for a directory
// that cannot be created or read, and a fault at the path of the first
// entry that cannot be removed or written.
//
// # Allocation contract
//
// Update allocates the tree in dir, what Write allocates, and what the file
// system's calls allocate.
func Update(dir string, t Tree) error {
	if err := t.Check(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, DirMode); err != nil {
		return fault.New("the directory cannot be created").Because(err)
	}
	current, err := Read(os.DirFS(dir), false)
	if err != nil {
		return err
	}
	wanted := withoutModes(t.Full())
	var steps []step
	kept := Tree{}
	for _, path := range slices.Backward(current.Paths()) {
		if w, ok := wanted[path]; ok && !differs(w, current[path]) {
			kept[path] = w
			continue
		}
		steps = append(steps, step{path: path, run: func(root *os.Root) error { return root.RemoveAll(path) }})
	}
	return apply(dir, creating(wanted, kept, steps...))
}

// creating returns the operations that write each entry of full that kept
// lacks, after the operations before: each entry in path order, and then the
// mode of each directory that it writes, deepest first.
func creating(full, kept Tree, before ...step) []step {
	steps := before
	var modes []step
	for _, path := range full.Paths() {
		if _, ok := kept[path]; ok {
			continue
		}
		e := full[path]
		steps = append(steps, step{path: path, run: create(path, e)})
		if e.Kind == Dir {
			modes = append(modes, step{path: path, run: func(root *os.Root) error {
				return setDirMode(root, path, modeOf(e))
			}})
		}
	}
	slices.Reverse(modes)
	return append(steps, modes...)
}

// create returns the operation that creates the entry e at path: a file
// written with its content and then given its mode, a directory, or a link.
// None writes over an entry that is there.
func create(path string, e Entry) func(root *os.Root) error {
	if e.Kind == Dir {
		return func(root *os.Root) error { return root.Mkdir(path, createDir) }
	}
	if e.Kind == Link {
		return func(root *os.Root) error { return root.Symlink(e.Target, path) }
	}
	return func(root *os.Root) error {
		f, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, createFile)
		if err != nil {
			return err
		}
		_, err = f.WriteString(e.Content)
		return errors.Join(err, f.Close(), root.Chmod(path, modeOf(e)))
	}
}

// apply runs steps on the root of dir, in order, and returns a fault at the
// path of the first step that fails.
func apply(dir string, steps []step) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fault.New("the directory cannot be opened").Because(err)
	}
	defer root.Close()
	for _, s := range steps {
		if err := s.run(root); err != nil {
			return fault.At(fault.New("the entry cannot be written").Because(err), fault.Key(s.path))
		}
	}
	return nil
}

// modeOf returns the mode that a write sets on the file or the directory e:
// its stated mode, and where it states none [FileMode], [ExecutableMode] or
// [DirMode].
func modeOf(e Entry) fs.FileMode {
	if e.Stated {
		return e.Mode
	}
	if e.Kind == Dir {
		return DirMode
	}
	if e.Executable() {
		return ExecutableMode
	}
	return FileMode
}

// Unlock gives the owner of dir and of each directory below it the
// permission to read, write and search it, so that every entry that [Write]
// wrote can be removed, also below a directory whose mode forbids it. It
// follows no link, reports nothing, and leaves a directory that it cannot
// change as it is.
//
// # Allocation contract
//
// Unlock allocates what the walk of dir allocates.
func Unlock(dir string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			info, err := d.Info()
			if err == nil {
				_ = os.Chmod(path, info.Mode().Perm()|createDir)
			}
		}
		return nil
	})
}

// withoutModes returns t as a tree that states no mode, with the execute
// bit of each file.
func withoutModes(t Tree) Tree {
	out := make(Tree, len(t))
	for path, e := range t {
		out[path] = asCompared(e, Entry{})
	}
	return out
}
