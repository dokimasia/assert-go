// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"encoding/json"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
)

// everyKind is the tree literal of an entry of every kind and form, and
// everyKindTree the tree that it states.
const everyKind = `{"type":"tree","entries":[` +
	`{"path":"bin/run","text":"#!/bin/sh\n","executable":true},` +
	`{"path":"cache","directory":true},` +
	`{"path":"current","link":"bin/run"},` +
	`{"path":"keys","directory":true,"mode":448},` +
	`{"path":"keys/id","text":"secret\n","mode":384},` +
	`{"path":"logo.png","bytes":"89504e47"}]}`

// everyKindTree is the tree that everyKind states.
var everyKindTree = files.Tree{
	"bin/run":  files.Executable("#!/bin/sh\n"),
	"cache":    files.Dir(),
	"current":  files.Link("bin/run"),
	"keys":     files.Dir().WithMode(0o700),
	"keys/id":  files.Text("secret\n").WithMode(privateMode),
	"logo.png": files.Bytes([]byte{0x89, 'P', 'N', 'G'}),
}

// TestTree checks the tree literal that a tree marshals to and decodes from.
func TestTree(t *testing.T) {
	t.Parallel()

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the tree literal of every kind and form of entry", func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(everyKindTree)
			assert.NoError(t, err, "the tree marshals")
			assert.Equal(t, string(got), everyKind, "the tree literal, its entries in path order")
		})

		t.Run(
			"states the whole content of a file over 65,536 bytes, which UnmarshalJSON reads back",
			func(t *testing.T) {
				t.Parallel()

				big := files.Tree{"big": files.Text(strings.Repeat("a", 65537))}
				raw, err := json.Marshal(big)
				assert.NoError(t, err, "the tree marshals")
				var got files.Tree
				assert.NoError(t, json.Unmarshal(raw, &got), "the literal decodes")
				assert.Equal(t, got, big, "the tree with the whole content")
			},
		)

		t.Run("returns a fault at an entry that breaks a rule of a tree", func(t *testing.T) {
			t.Parallel()

			_, err := files.Tree{"a": files.Text(""), "a/b": files.Text("")}.MarshalJSON()
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, f.Op, "files.Tree.MarshalJSON", "the operation")
			assert.Equal(t, f.Path, fault.Path{fault.Key("a/b")}, "the entry below a file")
		})
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the tree that the literal states", func(t *testing.T) {
			t.Parallel()

			var got files.Tree
			assert.NoError(t, json.Unmarshal([]byte(everyKind), &got), "the literal decodes")
			assert.Equal(t, got, everyKindTree, "the tree of every kind and form of entry")
		})

		t.Run("returns a fault for a text that is no tree literal, and leaves the tree", func(t *testing.T) {
			t.Parallel()

			got := files.Tree{"kept": files.Dir()}
			err := got.UnmarshalJSON([]byte(`{"type":"tree","entries":[{"path":"a","digest":"sha256:00"}]}`))
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, f.Op, "files.Tree.UnmarshalJSON", "the operation")
			assert.Equal(t, f.Path, fault.Path{fault.Field("entries"), fault.Index(0), fault.Field("digest")},
				"the digest, which no input states")
			assert.Equal(t, got, files.Tree{"kept": files.Dir()}, "the tree is unchanged")
		})
	})
}

// TestTreeAllocs checks the allocation ceilings of the codec of a tree.
func TestTreeAllocs(t *testing.T) {
	alloctest.Check(t, treeCases())
}

// BenchmarkTree measures the codec of a tree.
func BenchmarkTree(b *testing.B) {
	for _, c := range treeCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// treeCases returns a call of each method of a tree, of every kind of
// entry, with its allocation ceiling.
func treeCases() []alloctest.Case {
	raw := []byte(everyKind)
	var tree files.Tree
	return []alloctest.Case{
		{Name: "MarshalJSON", Call: func(assert.TB) { keptBytes, errKept = everyKindTree.MarshalJSON() }, Allocs: 18},
		{Name: "UnmarshalJSON", Call: func(assert.TB) { errKept = tree.UnmarshalJSON(raw) }, Allocs: 160},
	}
}
