// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package alphabet_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/alphabet"
)

// TestInterval checks the union of intervals of indices.
func TestInterval(t *testing.T) {
	t.Parallel()

	t.Run("Merge", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []alphabet.Interval
			want []alphabet.Interval
		}{
			{
				name: "joins overlapping and touching intervals in order",
				give: []alphabet.Interval{
					{First: 5, Last: 6},
					{First: 0, Last: 2},
					{First: 3, Last: 3},
					{First: 8, Last: 9},
					{First: 9, Last: 12},
				},
				want: []alphabet.Interval{{First: 0, Last: 3}, {First: 5, Last: 6}, {First: 8, Last: 12}},
			},
			{
				name: "keeps a gap of one index",
				give: []alphabet.Interval{{First: 3, Last: 4}, {First: 0, Last: 1}},
				want: []alphabet.Interval{{First: 0, Last: 1}, {First: 3, Last: 4}},
			},
			{
				name: "returns the outer interval for an interval inside another",
				give: []alphabet.Interval{{First: 0, Last: 10}, {First: 2, Last: 3}},
				want: []alphabet.Interval{{First: 0, Last: 10}},
			},
			{name: "returns no interval for none", give: nil, want: nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, alphabet.Merge(tt.give), tt.want, "the union")
			})
		}
	})
}

// TestIntervalZeroAlloc checks that Merge allocates nothing.
func TestIntervalZeroAlloc(t *testing.T) {
	intervals := []alphabet.Interval{{First: 5, Last: 6}, {First: 0, Last: 2}, {First: 3, Last: 3}}
	assert.MaxAllocs(t, func() { _ = alphabet.Merge(intervals) }, 0, "Merge allocates nothing")
}

// BenchmarkInterval measures Merge of unsorted intervals under a ceiling
// of no allocation. Each iteration copies the input, because Merge sorts
// it in place.
func BenchmarkInterval(b *testing.B) {
	b.Run("Merge", func(b *testing.B) {
		var got []alphabet.Interval
		input := []alphabet.Interval{{First: 5, Last: 6}, {First: 0, Last: 2}, {First: 3, Last: 3}, {First: 8, Last: 9}}
		work := make([]alphabet.Interval, len(input))
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			copy(work, input)
			got = alphabet.Merge(work)
		}
		assert.Length(b, got, 3, "three intervals after the union")
	})
}
