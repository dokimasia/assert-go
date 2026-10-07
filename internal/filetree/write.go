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
	return apply(dir, func(*os.Root) ([]step, error) { return creating(t.Full(), nil), nil })
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
	return apply(dir, func(*os.Root) ([]step, error) { return creating(wanted, kept, steps...), nil })
}

// Overwrite writes t into dir, a directory that exists, as files.Write does.
// It reads the entry at each path of t, and at each parent of one, before it
// writes anything. Where nothing is at a path, it creates the entry as [Write]
// does. A file replaces the content of the file at its path, also of a file
// that its owner may not write, and a link replaces the target of the link at
// its path. A directory keeps its entries. Each entry that t states gets its
// stated mode, and where it states none [FileMode], [ExecutableMode] or
// [DirMode], each directory after every entry below it. A parent that t
// implies keeps its mode where it exists. Every entry that t does not state
// keeps its content and its mode. Overwrite writes through os.Root and never
// follows a link: it replaces a link by removing the link itself. On Windows
// only the owner's write bit of a file's mode takes effect, as with [Write].
//
// # Errors
//
// It returns the fault of [Tree.Check] for a tree that breaks a rule of a
// tree, and a fault for a dir that cannot be opened. Before it writes
// anything, it returns a fault at the first path, in path order, whose entry
// cannot be read, is of another kind than t states, or is the entry at an
// earlier path of t, such as a second name of one file. A parent that is no
// directory is of another kind than t states. It returns a fault at the path
// of the first entry that cannot be written, whose cause is the error of the
// file system, such as a file that a directory without the owner's write bit
// refuses.
//
// # Allocation contract
//
// Overwrite allocates the tree with its implied directories, the information
// of each entry that it reads, an operation for each entry, and what the file
// system's calls allocate.
func Overwrite(dir string, t Tree) error {
	if err := t.Check(); err != nil {
		return err
	}
	return apply(dir, func(root *os.Root) ([]step, error) { return overwriting(root, t) })
}

// overwriting reads the entry at each path of the full tree of t in root, in
// path order, and returns the operations that write t over them: the
// replacement of each file and each link that is there, the operations of
// [Write] for each entry that is missing, and then the mode of each directory
// that is there and that t states, deepest first. No directory that is there
// is below one that the write creates, so the modes of the created
// directories come first. It returns the fault of the first entry that it
// refuses, before any operation runs.
func overwriting(root *os.Root, t Tree) ([]step, error) {
	full := t.Full()
	there := Tree{}
	read := map[string]fs.FileInfo{}
	var replaced, modes []step
	for _, path := range full.Paths() {
		e := full[path]
		info, err := root.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fault.At(readFault(err), fault.Key(path))
		}
		if err := refusal(info, e, read); err != nil {
			return nil, fault.At(err, fault.Key(path))
		}
		there[path], read[path] = e, info
		if e.Kind != Dir {
			replaced = append(replaced, step{path: path, run: replace(path, e)})
		} else if _, stated := t[path]; stated {
			modes = append(modes, step{path: path, run: dirMode(path, e)})
		}
	}
	slices.Reverse(modes)
	return append(creating(full, there, replaced...), modes...), nil
}

// refusal returns a fault for the entry of info, where a tree states e: an
// entry that is no file, directory or link, one of another kind than e, and
// one that is the entry of a path in read. It returns nil for an entry that a
// write replaces or keeps.
func refusal(info fs.FileInfo, e Entry, read map[string]fs.FileInfo) error {
	got, err := entryOf(info.Mode(), false)
	if err != nil {
		return err
	}
	if got.Kind != e.Kind {
		return fault.New("the entry is a %s, and the tree states a %s", got.Kind.Name(), e.Kind.Name())
	}
	for path, other := range read {
		if os.SameFile(info, other) {
			return fault.New("the file system maps the path to the entry at %q", path)
		}
	}
	return nil
}

// replace returns the operation that writes e over the entry of its kind at
// path: the content and then the mode of a file, after it gives the owner the
// permission to write the file, and the target of a link, after it removes
// the link itself.
func replace(path string, e Entry) func(root *os.Root) error {
	if e.Kind == Link {
		return func(root *os.Root) error {
			if err := root.Remove(path); err != nil {
				return err
			}
			return root.Symlink(e.Target, path)
		}
	}
	return func(root *os.Root) error {
		return errors.Join(root.Chmod(path, createFile), writeFile(root, path, os.O_TRUNC, e))
	}
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
			modes = append(modes, step{path: path, run: dirMode(path, e)})
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
	return func(root *os.Root) error { return writeFile(root, path, os.O_CREATE|os.O_EXCL, e) }
}

// writeFile opens the file at path in root for writing with flag, writes the
// content of e into it, and gives it the mode of e.
func writeFile(root *os.Root, path string, flag int, e Entry) error {
	f, err := root.OpenFile(path, os.O_WRONLY|flag, createFile)
	if err != nil {
		return err
	}
	_, err = f.WriteString(e.Content)
	return errors.Join(err, f.Close(), root.Chmod(path, modeOf(e)))
}

// dirMode returns the operation that gives the directory e at path its mode.
func dirMode(path string, e Entry) func(root *os.Root) error {
	return func(root *os.Root) error { return setDirMode(root, path, modeOf(e)) }
}

// apply opens the root of dir and runs on it the steps that plan returns, in
// order. It returns the error of plan as it is, and a fault at the path of
// the first step that fails.
func apply(dir string, plan func(root *os.Root) ([]step, error)) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fault.New("the directory cannot be opened").Because(err)
	}
	defer root.Close()
	steps, err := plan(root)
	if err != nil {
		return err
	}
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
