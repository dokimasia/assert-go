// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// filesContract is the contract of the call of a files vector.
const filesContract = "the files of the vector are as the vector states"

// The directories that a files vector writes below its directory: the
// workspace, and the conventional directory, which golden.MatchTree resolves
// a name against relative to the working directory.
const (
	workspaceDir    = "workspace"
	conventionalDir = "testdata/golden"
)

// The members of a files vector that the path of a fault names, beside the
// members of every vector.
const (
	workspaceMember = "workspace"
	goldenMember    = "golden"
	afterMember     = "after"
)

// filesVector is a vector of an assertion that reads files: the tree of its
// workspace, the arguments of the call or its subject, its golden tree, the
// outcome that it expects, and the golden tree that an update leaves.
type filesVector struct {
	Workspace json.RawMessage   `json:"workspace"`
	Args      []json.RawMessage `json:"args"`
	Subject   struct {
		Kind string `json:"kind"`
	} `json:"subject"`
	Golden json.RawMessage            `json:"golden"`
	Expect string                     `json:"expect"`
	Detail map[string]json.RawMessage `json:"detail"`
	After  json.RawMessage            `json:"after"`
}

// caller returns the call of the assertion of the files vector v, whose
// directory is dir and whose workspace is ws, and the fault of an input of v
// that the call cannot take. The call returns the fault of what it observes
// beside its call record.
type caller func(v filesVector, dir, ws string) (func(tb assert.TB) error, error)

// checkFiles returns the runner of the files vectors whose calls build
// returns. The runner writes the vector's workspace into the directory
// workspace below dir, makes the call on a recorder, and compares the
// verdict and the detail of the call record with the vector's.
func checkFiles(build caller) runner {
	return func(raw json.RawMessage, dir string) error {
		var v filesVector
		if err := decode(raw, &v); err != nil {
			return err
		}
		tree, err := filetree.Decode(v.Workspace)
		if err == nil {
			err = filetree.Write(dir, within(workspaceDir, tree))
		}
		if err != nil {
			return fault.At(err, fault.Field(workspaceMember))
		}
		run, err := build(v, dir, filepath.Join(dir, workspaceDir))
		if err != nil {
			return err
		}
		rec := assert.NewRecorder()
		if err := run(rec); err != nil {
			return err
		}
		// A recorder keeps the call record of every call.
		return compareVerdict(callsOf(rec)[0], v.Expect, v.Detail)
	}
}

// treeCall returns the caller of the comparison of trees compare, whose one
// argument is the wanted tree, and which compares the tree of the workspace.
func treeCall(compare func(tb assert.TB, got fs.FS, want files.Tree, msg string)) caller {
	return func(v filesVector, _, ws string) (func(assert.TB) error, error) {
		var want files.Tree
		err := v.arity(1)
		if err == nil {
			err = at(json.Unmarshal(v.Args[0], &want), fault.Field(argsMember), fault.Index(0))
		}
		return func(tb assert.TB) error {
			compare(tb, os.DirFS(ws), want, filesContract)
			return nil
		}, err
	}
}

// unchangedCall returns the call of a tree-unchanged vector, whose subject
// is a behaviour of [Subjects] that reads or writes the files of the
// workspace. The call returns a fault at the subject for a behaviour that
// fails.
func unchangedCall(v filesVector, _, ws string) (func(assert.TB) error, error) {
	build, ok := Subjects[v.Subject.Kind]
	if !ok || build().Files == nil {
		return nil, fault.At(fault.New("%q is no subject of files", v.Subject.Kind),
			fault.Field(subjectMember), fault.Field(kindMember))
	}
	subject := build()
	return func(tb assert.TB) error {
		var err error
		files.Unchanged(tb, os.DirFS(ws), func() { err = subject.Files(ws) }, filesContract)
		if err != nil {
			return fault.At(fault.New("the subject fails").Because(err), fault.Field(subjectMember))
		}
		return nil
	}, nil
}

// goldenCall returns the call of a golden-match-tree vector, whose arguments
// are the name and update. golden.MatchTree resolves the name against the
// working directory, so the call refuses a dir that is not the working
// directory, and writes the vector's golden tree below the conventional
// directory of dir. The call returns a fault at after for a golden tree after
// it that differs from the one that the vector states there.
func goldenCall(v filesVector, dir, ws string) (func(assert.TB) error, error) {
	values, err := v.arguments(2)
	var name string
	var update bool
	if err == nil {
		name, err = argument[string](values, 0)
	}
	if err == nil {
		update, err = argument[bool](values, 1)
	}
	goldenDir := path.Join(conventionalDir, name)
	if err == nil {
		err = writeGolden(dir, goldenDir, v.Golden)
	}
	return func(tb assert.TB) error {
		golden.MatchTree(tb, name, os.DirFS(ws), update)
		if v.After == nil {
			return nil
		}
		return compareAfter(filepath.Join(dir, goldenDir), v.After)
	}, err
}

// writeGolden writes the golden tree that raw states into dir, at the path
// goldenDir, and nothing for null. It returns a fault for a dir that is not
// the working directory, and a fault at golden for a raw that is no tree
// literal.
func writeGolden(dir, goldenDir string, raw json.RawMessage) error {
	here, _ := os.Stat(".")
	there, _ := os.Stat(dir)
	if !os.SameFile(here, there) {
		return fault.New("the directory of the vector is not the working directory, " +
			"which golden.MatchTree resolves a name against")
	}
	if string(raw) == jsonNull {
		return nil
	}
	tree, err := filetree.Decode(raw)
	if err == nil {
		err = filetree.Write(dir, within(goldenDir, tree))
	}
	return at(err, fault.Field(goldenMember))
}

// compareAfter returns how the golden tree in dir, read without its modes,
// differs from after, the tree that a vector states that an update leaves,
// or nil when they are the same.
func compareAfter(dir string, after json.RawMessage) error {
	tree, err := filetree.Read(os.DirFS(dir), false)
	// A tree of strings and modes always encodes.
	got, _ := tree.Encode()
	if err != nil || !sameJSON(got, after) {
		return fault.At(fault.New("the golden tree is %s, want %s", string(got), string(after)).Because(err),
			fault.Field(afterMember))
	}
	return nil
}

// kindCall returns the caller of the assertion of the kind of the entry at a
// path, whose one argument is the path relative to the workspace.
func kindCall(assertion func(tb assert.TB, path, msg string)) caller {
	return func(v filesVector, _, ws string) (func(assert.TB) error, error) {
		path, _, err := v.pathArguments(ws, 1)
		return func(tb assert.TB) error {
			assertion(tb, path, filesContract)
			return nil
		}, err
	}
}

// linksToCall returns the call of a links-to vector, whose arguments are the
// path and the target.
func linksToCall(v filesVector, _, ws string) (func(assert.TB) error, error) {
	path, values, err := v.pathArguments(ws, 2)
	var target string
	if err == nil {
		target, err = argument[string](values, 1)
	}
	return func(tb assert.TB) error {
		files.LinksTo(tb, path, target, filesContract)
		return nil
	}, err
}

// hasContentCall returns the call of a has-content vector, whose arguments
// are the path and the content, as text or as bytes.
func hasContentCall(v filesVector, _, ws string) (func(assert.TB) error, error) {
	path, values, err := v.pathArguments(ws, 2)
	var content string
	if err == nil {
		switch c := values[1].(type) {
		case string:
			content = c
		case []byte:
			content = string(c)
		default:
			err = fault.At(fault.New("the argument is no text and no bytes"), fault.Field(argsMember), fault.Index(1))
		}
	}
	return func(tb assert.TB) error {
		files.HasContent(tb, path, content, filesContract)
		return nil
	}, err
}

// hasModeCall returns the call of a has-mode vector, whose arguments are the
// path and the mode.
func hasModeCall(v filesVector, _, ws string) (func(assert.TB) error, error) {
	path, values, err := v.pathArguments(ws, 2)
	var mode int
	if err == nil {
		mode, err = argument[int](values, 1)
	}
	return func(tb assert.TB) error {
		files.HasMode(tb, path, fs.FileMode(mode), filesContract)
		return nil
	}, err
}

// arity returns a fault at args for a vector that states other than n
// arguments.
func (v filesVector) arity(n int) error {
	if len(v.Args) != n {
		return fault.At(fault.New("the number of arguments is %d, want %d", len(v.Args), n), fault.Field(argsMember))
	}
	return nil
}

// arguments returns the values of the n typed literals that v states as its
// arguments, and a fault at args for another number of arguments or at the
// first literal that states no value.
func (v filesVector) arguments(n int) ([]any, error) {
	if err := v.arity(n); err != nil {
		return nil, err
	}
	values, err := decodeValues(v.Args)
	return values, at(err, fault.Field(argsMember))
}

// pathArguments returns the values of the n arguments of v, and the path of
// the operating system that the first one states relative to the workspace
// ws, as arguments and argument return them.
func (v filesVector) pathArguments(ws string, n int) (string, []any, error) {
	values, err := v.arguments(n)
	var rel string
	if err == nil {
		rel, err = argument[string](values, 0)
	}
	return filepath.Join(ws, filepath.FromSlash(rel)), values, err
}

// argument returns the value at i of values, the arguments of a vector, as a
// T, and a fault at the argument for a value of another type.
func argument[T any](values []any, i int) (T, error) {
	value, ok := values[i].(T)
	if !ok {
		return value, fault.At(fault.New("the argument is no %T", value), fault.Field(argsMember), fault.Index(i))
	}
	return value, nil
}

// within returns t below the directory dir of a tree, and dir as a
// directory of its own: each path of t joined to dir.
func within(dir string, t filetree.Tree) filetree.Tree {
	out := filetree.Tree{dir: {Kind: filetree.Dir}}
	for p, e := range t {
		out[path.Join(dir, p)] = e
	}
	return out
}
