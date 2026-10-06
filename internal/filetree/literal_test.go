// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
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
var everyKindTree = filetree.Tree{
	"bin/run":  executableFile,
	"cache":    directory,
	"current":  linkTo("bin/run"),
	"keys":     dirWith(0o700),
	"keys/id":  fileWith("secret\n", privateMode),
	"logo.png": fileOf("\x89PNG"),
}

// TestLiteral checks the tree literal that a tree marshals to, and the tree
// that a literal decodes to.
func TestLiteral(t *testing.T) {
	t.Parallel()

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the literal of every kind and form of entry, in path order", func(t *testing.T) {
			t.Parallel()

			got, err := everyKindTree.MarshalJSON()
			assert.NoError(t, err, "the tree marshals")
			assert.Equal(t, string(got), everyKind, "the tree literal")
		})

		t.Run("states a file over the limit by its digest and its size", func(t *testing.T) {
			t.Parallel()

			got, err := filetree.Tree{"big": fileOf(strings.Repeat("a", filetree.ContentLimit+1))}.MarshalJSON()
			assert.NoError(t, err, "the tree marshals")
			assert.Equal(t, string(got), `{"type":"tree","entries":[{"path":"big","digest":`+
				`"sha256:008ffc88d3c96a9f307524eb361e47c5222a887fc45fa0c1fb8d429c5c23b430","size":65537}]}`,
				"the digest of 65,537 bytes of the letter a")
		})

		t.Run("states the mode of a file in place of executable", func(t *testing.T) {
			t.Parallel()

			got, err := filetree.Tree{"run": fileWith("", 0o700)}.MarshalJSON()
			assert.NoError(t, err, "the tree marshals")
			assert.Equal(t, string(got), `{"type":"tree","entries":[{"path":"run","text":"","mode":448}]}`,
				"the mode states the execute bit")
		})
	})

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the tree of every kind and form of entry", func(t *testing.T) {
			t.Parallel()

			got, err := filetree.Decode([]byte(everyKind))
			assert.NoError(t, err, "the literal decodes")
			assert.Equal(t, got, everyKindTree, "the tree that the literal states")
		})

		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{name: "returns a fault for a text that is no JSON", give: `{`, wantReason: "the text is no tree literal"},
			{
				name: "returns a fault at the type of another literal", give: `{"type":"list","entries":[]}`,
				wantPath: fault.Path{fault.Field("type")}, wantReason: `the type "list" is not tree`,
			},
			{
				name: "returns a fault for a literal without entries", give: `{"type":"tree"}`,
				wantPath: fault.Path{fault.Field("entries")}, wantReason: "the literal states no list of entries",
			},
			{
				name: "returns a fault at an entry that is no object", give: entries(`"a"`),
				wantPath: at(0), wantReason: "the entry is no object",
			},
			{
				name: "returns a fault at an entry of two kinds", give: entries(`{"path":"a","text":"","link":"b"}`),
				wantPath:   at(0),
				wantReason: "the entry states [text link], and an entry states one of text, bytes, directory and link",
			},
			{
				name: "returns a fault at an entry of no kind", give: entries(`{"path":"a"}`), wantPath: at(0),
				wantReason: "the entry states [], and an entry states one of text, bytes, directory and link",
			},
			{
				name: "returns a fault at a member that the kind takes not",
				give: entries(`{"path":"a","link":"b","mode":420}`), wantPath: at(0, fault.Field("mode")),
				wantReason: "a link entry takes no mode",
			},
			{
				name:       "returns a fault at an entry that does not parse",
				give:       entries(`{"path":"a","text":"","mode":-1}`),
				wantPath:   at(0),
				wantReason: "the entry does not parse",
			},
			{
				name: "returns a fault at an entry of a mode and executable",
				give: entries(`{"path":"a","text":"","mode":420,"executable":false}`),
				wantPath: at(
					0,
				),
				wantReason: "the entry states a mode and executable, and the mode states the execute bit",
			},
			{
				name:       "returns a fault at a directory that is not true",
				give:       entries(`{"path":"a","directory":false}`),
				wantPath:   at(0, fault.Field("directory")),
				wantReason: "directory is not true",
			},
			{
				name: "returns a fault at a digest, which only a record states",
				give: entries(`{"path":"a","digest":"sha256:00","size":70000}`),
				wantPath: at(
					0,
					fault.Field("digest"),
				),
				wantReason: "a digest states no content, and only a record states one",
			},
			{
				name: "returns a fault at a text that is no string", give: entries(`{"path":"a","text":null}`),
				wantPath: at(0, fault.Field("text")), wantReason: "the text is no string",
			},
			{
				name: "returns a fault at bytes in upper case", give: entries(`{"path":"a","bytes":"FF"}`),
				wantPath: at(0, fault.Field("bytes")), wantReason: `"FF" is no lowercase hexadecimal`,
			},
			{
				name: "returns a fault at bytes that are no hexadecimal", give: entries(`{"path":"a","bytes":"0g"}`),
				wantPath: at(0, fault.Field("bytes")), wantReason: `"0g" is no lowercase hexadecimal`,
			},
			{
				name:     "returns a fault at an entry out of path order",
				give:     entries(`{"path":"b","text":""}`, `{"path":"a","text":""}`),
				wantPath: at(1), wantReason: `the path "a" is not after the path before it`,
			},
			{
				name:     "returns a fault at a path stated twice",
				give:     entries(`{"path":"a","text":""}`, `{"path":"a","text":""}`),
				wantPath: at(1), wantReason: `the path "a" is not after the path before it`,
			},
			{
				name:     "returns the fault of a tree that breaks a rule",
				give:     entries(`{"path":"a","text":""}`, `{"path":"a/b","text":""}`),
				wantPath: fault.Path{fault.Key("a/b")}, wantReason: `the entry is below "a", which is a file`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := filetree.Decode([]byte(tt.give))
				expectFault(t, err, tt.wantPath, tt.wantReason)
			})
		}
	})
}

// entries returns a tree literal of the JSON of each entry.
func entries(each ...string) string {
	return `{"type":"tree","entries":[` + strings.Join(each, ",") + `]}`
}

// at returns the path of a fault at the entry i of a tree literal, and at
// the segments after it.
func at(i int, segs ...fault.Segment) fault.Path {
	return append(fault.Path{fault.Field("entries"), fault.Index(i)}, segs...)
}

// TestLiteralAllocs checks the allocation ceilings of the codec of the tree
// literal.
func TestLiteralAllocs(t *testing.T) {
	alloctest.Check(t, literalCases())
}

// BenchmarkLiteral measures the codec of the tree literal.
func BenchmarkLiteral(b *testing.B) {
	for _, c := range literalCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// literalCases returns a call of each function of the codec, of the tree
// of every kind of entry, with its allocation ceiling, measured.
func literalCases() []alloctest.Case {
	raw := []byte(everyKind)
	return []alloctest.Case{
		{Name: "MarshalJSON", Call: func(assert.TB) { keptBytes, errKept = everyKindTree.MarshalJSON() }, Allocs: 13},
		{Name: "Decode", Call: func(assert.TB) { keptTree, errKept = filetree.Decode(raw) }, Allocs: 120},
	}
}
