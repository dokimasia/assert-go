// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The members of a files vector that the paths of the tests name, beside the
// members of every vector.
const (
	workspaceAt = "workspace"
	goldenAt    = "golden"
	afterAt     = "after"
)

// The trees of the files vectors that the tests build, beside noFiles and
// aFile.
const (
	// readOnlyFile is the tree of the file a.txt, of the text a, which its
	// owner may read and not write.
	readOnlyFile = `{"type":"tree","entries":[{"path":"a.txt","text":"a","mode":256}]}`
	// listLiteral is a typed literal of another type than tree.
	listLiteral = `{"type":"list"}`
)

// The reasons of the faults of the files runners that several tests state.
const (
	// noTree is the reason of the fault of listLiteral, at its member type.
	noTree = `the type "list" is not tree`
	// noString is the reason of an argument that is no string.
	noString = "the argument is no string"
)

// TestFiles drives the rules of the files runners that the definition's
// vectors cannot. Written with testing rather than with this library,
// because a verdict is not written with the subject.
func TestFiles(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			kind       conformance.VectorKind
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault at the workspace of a vector whose workspace is no tree literal",
				kind:       conformance.PathAbsent,
				give:       filesVector(listLiteral, pathArgs("a.txt"), "pass"),
				wantPath:   inVector(fault.Field(workspaceAt), fault.Field(typeAt)),
				wantReason: noTree,
			},
			{
				name:       "returns a fault at the arguments of a tree-equal vector that states two trees",
				kind:       conformance.TreeEqual,
				give:       filesVector(aFile, "["+aFile+","+aFile+"]", "pass"),
				wantPath:   inVector(fault.Field(argsAt)),
				wantReason: "the number of arguments is 2, want 1",
			},
			{
				name:       "returns a fault at the argument of a tree-contains vector that is no tree literal",
				kind:       conformance.TreeContains,
				give:       filesVector(aFile, "["+listLiteral+"]", "pass"),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(0), fault.Field(typeAt)),
				wantReason: noTree,
			},
			{
				name:       "returns a fault at the subject of a tree-unchanged vector whose subject fails",
				kind:       conformance.TreeUnchanged,
				give:       unchanging(readOnlyFile, "rewrites-files"),
				wantPath:   inVector(fault.Field(subjectAt)),
				wantReason: "the subject fails",
			},
			{
				name:       "returns a fault at the kind of a tree-unchanged vector of no subject",
				kind:       conformance.TreeUnchanged,
				give:       unchanging(aFile, "widget"),
				wantPath:   inVector(fault.Field(subjectAt), fault.Field(kindAt)),
				wantReason: `"widget" is no subject of files`,
			},
			{
				name:       "returns a fault at the kind of a tree-unchanged vector of a subject that reads no files",
				kind:       conformance.TreeUnchanged,
				give:       unchanging(aFile, "identity"),
				wantPath:   inVector(fault.Field(subjectAt), fault.Field(kindAt)),
				wantReason: `"identity" is no subject of files`,
			},
			{
				name:       "returns a fault at the arguments of a golden-match-tree vector that states only the name",
				kind:       conformance.GoldenMatchTree,
				give:       goldenVector("["+stringOf("api")+"]", null, null),
				wantPath:   inVector(fault.Field(argsAt)),
				wantReason: "the number of arguments is 1, want 2",
			},
			{
				name:       "returns a fault at the name of a golden-match-tree vector whose name is no string",
				kind:       conformance.GoldenMatchTree,
				give:       goldenVector("["+oneLiteral+","+oneLiteral+"]", null, null),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(0)),
				wantReason: noString,
			},
			{
				name:       "returns a fault at update of a golden-match-tree vector whose update is no bool",
				kind:       conformance.GoldenMatchTree,
				give:       goldenVector("["+stringOf("api")+","+oneLiteral+"]", null, null),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(1)),
				wantReason: "the argument is no bool",
			},
			{
				name:       "returns a fault at the id of a golden-match-tree vector in another directory than the working one",
				kind:       conformance.GoldenMatchTree,
				give:       goldenVector(goldenArgs(false), null, null),
				wantPath:   inVector(),
				wantReason: "the directory of the vector is not the working directory, which golden.MatchTree resolves a name against",
			},
			{
				name:       "returns a fault at the argument of an is-file vector that is no typed literal",
				kind:       conformance.IsFile,
				give:       filesVector(aFile, "["+widget+"]", "pass"),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(0), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at the path of an is-dir vector whose path is no string",
				kind:       conformance.IsDir,
				give:       filesVector(aFile, "["+oneLiteral+"]", "pass"),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(0)),
				wantReason: noString,
			},
			{
				name:       "returns a fault at the target of a links-to vector whose target is no string",
				kind:       conformance.LinksTo,
				give:       filesVector(aFile, pathArgs("a.txt", oneLiteral), "pass"),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(1)),
				wantReason: noString,
			},
			{
				name:       "returns a fault at the content of a has-content vector whose content is no text and no bytes",
				kind:       conformance.HasContent,
				give:       filesVector(aFile, pathArgs("a.txt", oneLiteral), "pass"),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(1)),
				wantReason: "the argument is no text and no bytes",
			},
			{
				name:       "returns a fault at the mode of a has-mode vector whose mode is no integer",
				kind:       conformance.HasMode,
				give:       filesVector(aFile, pathArgs("a.txt", stringOf("rw")), "pass"),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(1)),
				wantReason: "the argument is no int",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, tt.kind, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// TestFilesEnv drives the rules of the golden-match-tree runner that need
// the vector's directory as the working directory of the process. Each case
// changes the working directory, so the cases run one at a time.
func TestFilesEnv(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault at the golden tree of a vector whose golden tree is no tree literal",
				give:       goldenVector(goldenArgs(false), listLiteral, null),
				wantPath:   inVector(fault.Field(goldenAt), fault.Field(typeAt)),
				wantReason: noTree,
			},
			{
				name:     "returns a fault at after of a vector whose golden tree after the update differs",
				give:     goldenVector(goldenArgs(true), null, noFiles),
				wantPath: inVector(fault.Field(afterAt)),
				wantReason: `the golden tree is {"type":"tree","entries":[{"path":"a.txt","text":"a"}]}, want ` +
					noFiles,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := t.TempDir()
				t.Chdir(dir)
				v := conformance.Vector{Kind: conformance.GoldenMatchTree, ID: builtID, Raw: json.RawMessage(tt.give)}
				expectFault(t, v.Check(dir), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// unchanging returns a tree-unchanged vector of the workspace, a JSON text,
// and the subject of the kind subject, that expects a pass.
func unchanging(workspace, subject string) string {
	return fmt.Sprintf(`{"workspace":%s,"subject":{"kind":%q},"expect":"pass"}`, workspace, subject)
}

// goldenVector returns a golden-match-tree vector of the workspace aFile,
// the arguments, the golden tree and the tree after an update, each a JSON
// text, that expects a pass.
func goldenVector(args, golden, after string) string {
	return fmt.Sprintf(`{"workspace":%s,"args":%s,"golden":%s,"expect":"pass","after":%s}`, aFile, args, golden, after)
}

// goldenArgs returns the arguments of a golden-match-tree vector of the name
// api and update.
func goldenArgs(update bool) string {
	return fmt.Sprintf(`[%s,{"type":"bool","value":%t}]`, stringOf("api"), update)
}
