// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/filetree"
)

// TestRecord checks the record of a comparison that fails, and of a missing
// golden tree.
func TestRecord(t *testing.T) {
	t.Parallel()

	t.Run("NewRecord", func(t *testing.T) {
		t.Parallel()

		t.Run("states a missing entry in want, an extra one in got, and a changed one in both", func(t *testing.T) {
			t.Parallel()

			want := filetree.Tree{"a.txt": fileOf("b\n"), "new/c.txt": textFile}
			r := filetree.NewRecord(want, read, filetree.Differing(want, read, false))
			assert.Equal(t, r, filetree.Record{
				Want: filetree.Tree{"a.txt": fileOf("b\n"), "new": directory, "new/c.txt": textFile},
				Got: filetree.Tree{
					"a.txt": fileOf("a\n"), "current": linkTo("a.txt"), "docs": directory, "docs/b.md": fileOf("# b\n"),
					"run.sh": executableFile,
				},
				Differences: 7,
			}, "an entry of got states no mode where want states none, and a script states its execute bit")
		})

		t.Run("states the mode of an entry of got where the wanted entry states one", func(t *testing.T) {
			t.Parallel()

			want := filetree.Tree{"a.txt": fileWith("a\n", privateMode), "docs": dirWith(0o700)}
			r := filetree.NewRecord(want, read, []string{"a.txt", "docs"})
			assert.Equal(t, r.Got, filetree.Tree{"a.txt": read["a.txt"], "docs": read["docs"]},
				"the modes that the comparison read")
		})

		t.Run("lists the first 64 paths and counts every path", func(t *testing.T) {
			t.Parallel()

			want, got := filetree.Tree{}, filetree.Tree{}
			paths := make([]string, 0, filetree.MaxPaths+1)
			for i := range filetree.MaxPaths + 1 {
				path := fmt.Sprintf("f%02d", i)
				want[path], got[path], paths = fileOf("b"), fileOf("a"), append(paths, path)
			}
			r := filetree.NewRecord(want, got, paths)
			assert.Length(t, r.Want, filetree.MaxPaths, "the record lists 64 wanted entries")
			assert.Length(t, r.Got, filetree.MaxPaths, "and 64 entries read")
			assert.Equal(t, r.Differences, filetree.MaxPaths+1, "and counts all 65 paths")
			assert.NotContains(t, r.Got, "f64", "the path past the 64 is left out")
		})
	})

	t.Run("Missing", func(t *testing.T) {
		t.Parallel()

		t.Run("states no wanted tree, and the output's entries without their modes", func(t *testing.T) {
			t.Parallel()

			r := filetree.Missing(read)
			assert.Equal(t, r, filetree.Record{
				Got: filetree.Tree{
					"a.txt": fileOf("a\n"), "current": linkTo("a.txt"), "docs": directory, "docs/b.md": fileOf("# b\n"),
					"run.sh": executableFile,
				},
				Differences: 5,
			}, "the record of a missing golden tree")
		})

		t.Run("lists the first 64 entries of the output and counts them all", func(t *testing.T) {
			t.Parallel()

			got := filetree.Tree{}
			for i := range filetree.MaxPaths + 1 {
				got[fmt.Sprintf("f%02d", i)] = textFile
			}
			r := filetree.Missing(got)
			assert.Length(t, r.Got, filetree.MaxPaths, "the record lists 64 entries")
			assert.Equal(t, r.Differences, filetree.MaxPaths+1, "and counts all 65")
		})
	})

	t.Run("Fields", func(t *testing.T) {
		t.Parallel()

		t.Run("returns want, got and differences", func(t *testing.T) {
			t.Parallel()

			r := filetree.Record{Want: filetree.Tree{}, Got: filetree.Tree{"a": textFile}, Differences: 1}
			assert.Equal(t, r.Fields(), map[string]any{
				"want": filetree.Tree{}, "got": filetree.Tree{"a": textFile}, "differences": 1,
			}, "the detail of the record")
		})

		t.Run("returns want nil for a missing golden tree", func(t *testing.T) {
			t.Parallel()

			fields := filetree.Missing(filetree.Tree{"a": textFile}).Fields()
			assert.Nil(t, fields["want"], "no wanted tree")
		})
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give filetree.Record
			want string
		}{
			{
				name: "returns the tree literals of want and got and the int literal of differences",
				give: filetree.Record{Want: filetree.Tree{}, Got: filetree.Tree{"a": textFile}, Differences: 1},
				want: `{"want":{"type":"tree","entries":[]},` +
					`"got":{"type":"tree","entries":[{"path":"a","text":"a\n"}]},` +
					`"differences":{"type":"int","value":1}}`,
			},
			{
				name: "returns the literal of null for a missing golden tree",
				give: filetree.Missing(filetree.Tree{"a": textFile}),
				want: `{"want":{"type":"null"},"got":{"type":"tree","entries":[{"path":"a","text":"a\n"}]},` +
					`"differences":{"type":"int","value":1}}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give.MarshalJSON()
				assert.NoError(t, err, "the record marshals")
				assert.Equal(t, string(got), tt.want, "the detail of the call record")
			})
		}
	})
}

// TestRecordAllocs checks the allocation ceilings of the functions of a
// record.
func TestRecordAllocs(t *testing.T) {
	alloctest.Check(t, recordCases())
}

// BenchmarkRecord measures the functions of a record.
func BenchmarkRecord(b *testing.B) {
	for _, c := range recordCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// recordCases returns a call of each function of a record, with its
// allocation ceiling.
func recordCases() []alloctest.Case {
	want := filetree.Tree{"a.txt": fileOf("b\n")}
	paths := []string{"a.txt"}
	r := filetree.NewRecord(want, read, paths)
	return []alloctest.Case{
		{Name: "NewRecord", Call: func(assert.TB) { keptRecord = filetree.NewRecord(want, read, paths) }, Allocs: 8},
		{Name: "Missing", Call: func(assert.TB) { keptRecord = filetree.Missing(read) }, Allocs: 4},
		{Name: "Fields", Call: func(assert.TB) { keptFields = r.Fields() }, Allocs: 3},
		{Name: "MarshalJSON", Call: func(assert.TB) { keptBytes, errKept = r.MarshalJSON() }, Allocs: 32},
	}
}
