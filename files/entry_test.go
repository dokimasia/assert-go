// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/alloctest"
)

// TestEntry checks the entries that the constructors state, through the
// tree literal of a tree of each.
func TestEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give files.Entry
		want string
	}{
		{name: "Text returns a file of the text", give: files.Text("a\n"), want: `{"path":"e","text":"a\n"}`},
		{
			name: "Bytes returns a file of the bytes", give: files.Bytes([]byte{0x89, 'P'}),
			want: `{"path":"e","bytes":"8950"}`,
		},
		{
			name: "Executable returns a file that its owner may execute", give: files.Executable("#!/bin/sh\n"),
			want: `{"path":"e","text":"#!/bin/sh\n","executable":true}`,
		},
		{name: "Dir returns a directory", give: files.Dir(), want: `{"path":"e","directory":true}`},
		{
			name: "Link returns a link to the target", give: files.Link("../a.txt"),
			want: `{"path":"e","link":"../a.txt"}`,
		},
		{
			name: "WithMode states the mode of a file", give: files.Text("k").WithMode(privateMode),
			want: `{"path":"e","text":"k","mode":384}`,
		},
		{
			name: "WithMode states the mode of a directory", give: files.Dir().WithMode(0o700),
			want: `{"path":"e","directory":true,"mode":448}`,
		},
		{
			name: "WithMode of an executable file states the execute bit through the mode",
			give: files.Executable("x").WithMode(fileMode), want: `{"path":"e","text":"x","mode":420}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := files.Tree{"e": tt.give}.MarshalJSON()
			assert.NoError(t, err, "the tree marshals")
			assert.Equal(t, string(got), `{"type":"tree","entries":[`+tt.want+`]}`, "the literal of the entry")
		})
	}

	t.Run("Bytes", func(t *testing.T) {
		t.Parallel()

		t.Run("copies the content", func(t *testing.T) {
			t.Parallel()

			content := []byte("a")
			entry := files.Bytes(content)
			content[0] = 'b'
			assert.Equal(t, entry, files.Text("a"), "the entry keeps the content it was given")
		})
	})

	t.Run("WithMode", func(t *testing.T) {
		t.Parallel()

		panics := []struct {
			name string
			call func()
			want string
		}{
			{
				name: "panics for a bit beyond the nine permission bits",
				call: func() { files.Text("").WithMode(0o1000) },
				want: "files: WithMode(0o1000) states a bit beyond the nine permission bits",
			},
			{
				name: "panics for a link", call: func() { files.Link("a").WithMode(fileMode) },
				want: "files: WithMode(0o644) states the mode of no file and no directory",
			},
			{
				name: "panics for the zero entry", call: func() { files.Entry{}.WithMode(fileMode) },
				want: "files: WithMode(0o644) states the mode of no file and no directory",
			},
		}
		for _, tt := range panics {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, assert.Panics(t, tt.call, "the call panics"), any(tt.want), "the panic's message")
			})
		}
	})
}

// TestEntryAllocs checks the allocation ceilings of the constructors of an
// entry.
func TestEntryAllocs(t *testing.T) {
	alloctest.Check(t, entryCases())
}

// BenchmarkEntry measures the constructors of an entry.
func BenchmarkEntry(b *testing.B) {
	for _, c := range entryCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// entryCases returns a call of each constructor of an entry, with its
// allocation ceiling, measured.
func entryCases() []alloctest.Case {
	content := []byte("a\n")
	file := files.Text("a\n")
	return []alloctest.Case{
		{Name: "Text", Call: func(assert.TB) { keptEntry = files.Text("a\n") }},
		{Name: "Bytes", Call: func(assert.TB) { keptEntry = files.Bytes(content) }, Allocs: 1},
		{Name: "Executable", Call: func(assert.TB) { keptEntry = files.Executable("#!/bin/sh\n") }},
		{Name: "Dir", Call: func(assert.TB) { keptEntry = files.Dir() }},
		{Name: "Link", Call: func(assert.TB) { keptEntry = files.Link("a.txt") }},
		{Name: "WithMode", Call: func(assert.TB) { keptEntry = file.WithMode(privateMode) }},
	}
}
