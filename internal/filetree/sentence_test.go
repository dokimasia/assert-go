// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
)

// TestSentence checks the sentence of the record of a comparison of trees.
func TestSentence(t *testing.T) {
	t.Parallel()

	t.Run("Sentence", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give filetree.Record
			want string
		}{
			{
				name: "states each path with its wanted entry and the entry read",
				give: filetree.Record{
					Want: filetree.Tree{"a.txt": fileOf("b\n"), "keys/id": fileWith("k", privateMode)},
					Got: filetree.Tree{
						"a.txt":   textFile,
						"cache":   directory,
						"keys/id": fileWith("k", fileMode),
					},
					Differences: 3,
				},
				want: "the tree is built: 3 paths differ (-want +got)" +
					"\n\ta.txt: -files.Text(\"b\\n\") +files.Text(\"a\\n\")" +
					"\n\tcache: +files.Dir()" +
					"\n\tkeys/id: -files.Text(\"k\").WithMode(0o600) +files.Text(\"k\").WithMode(0o644)",
			},
			{
				name: "states one path in the singular",
				give: filetree.Record{Want: filetree.Tree{"b": textFile}, Got: filetree.Tree{}, Differences: 1},
				want: "the tree is built: 1 path differs (-want +got)\n\tb: -files.Text(\"a\\n\")",
			},
			{
				name: "states the number of the paths that the record does not list",
				give: filetree.Record{Want: filetree.Tree{"b": textFile}, Got: filetree.Tree{}, Differences: 66},
				want: "the tree is built: 66 paths differ (-want +got)\n\tb: -files.Text(\"a\\n\")\n\t… and 65 more",
			},
			{
				name: "states that the golden tree is missing for a record without a wanted tree",
				give: filetree.Missing(filetree.Tree{"a.txt": textFile}),
				want: "the tree is built: the golden tree is missing, and the output has 1 entries (+got)" +
					"\n\ta.txt: +files.Text(\"a\\n\")",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := filetree.Sentence(matcher.Failure{Contract: "the tree is built", Detail: tt.give.Fields()})
				assert.Equal(t, got, tt.want, "the sentence of the record")
			})
		}
	})
}

// TestSentenceAllocs checks the allocation ceiling of Sentence.
func TestSentenceAllocs(t *testing.T) {
	alloctest.Check(t, sentenceCases())
}

// BenchmarkSentence measures Sentence.
func BenchmarkSentence(b *testing.B) {
	for _, c := range sentenceCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// sentenceCases returns a call of Sentence of a record of one changed file,
// with its allocation ceiling, measured.
func sentenceCases() []alloctest.Case {
	f := matcher.Failure{Contract: "the tree is built", Detail: filetree.Record{
		Want: filetree.Tree{"a.txt": fileOf("b\n")}, Got: filetree.Tree{"a.txt": textFile}, Differences: 1,
	}.Fields()}
	return []alloctest.Case{
		{Name: "Sentence", Call: func(assert.TB) { kept = filetree.Sentence(f) }, Allocs: 11},
	}
}
