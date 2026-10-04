// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
)

// byteK is one more than the largest byte, the k of a byte string.
const byteK = 256

// TestSequence checks what a sequence draw consumes from the stream, and
// what it returns.
func TestSequence(t *testing.T) {
	t.Parallel()

	t.Run("AppendSequence", func(t *testing.T) {
		t.Parallel()

		pinned := []struct {
			name string
			b    choice.SequenceBounds
			want [][]uint32
		}{
			{
				name: "returns the pinned draws of byte strings",
				b:    sequenceBounds(t, byteK, 0, 16),
				want: [][]uint32{nil, {233, 13, 255, 33, 143}, nil, {2, 98, 202, 167, 0}, {12, 163, 182, 4}, {149}},
			},
			{
				name: "returns the pinned draws of sequences with a minimum length",
				b:    sequenceBounds(t, 3, 2, 6),
				want: [][]uint32{{0, 1, 1, 0}, {1, 2}, {0, 0}, {1, 2, 2}, {0, 1, 2}, {1, 2}},
			},
		}
		for _, tt := range pinned {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := random.New(pinnedSeed)
				got := make([][]uint32, len(tt.want))
				for i := range got {
					got[i] = random.AppendSequence(nil, &s, tt.b)
				}
				assert.Equal(t, got, tt.want, "the first draws of the seed")
			})
		}

		t.Run("consumes the stream as a list of integers does", func(t *testing.T) {
			t.Parallel()
			b := sequenceBounds(t, byteK, 0, 16)
			for seed := range uint64(100) {
				s, twin := random.New(seed), random.New(seed)
				var want []uint32
				for random.Flag(&twin, b.Sizes(), len(want)) {
					want = append(want, uint32(random.Integer(&twin, b.Element()).Magnitude()))
				}
				msg := "the draw of seed " + strconv.FormatUint(seed, 10)
				assert.Equal(t, random.AppendSequence(nil, &s, b), want, msg)
			}
		})

		t.Run("returns sequences the bounds admit", func(t *testing.T) {
			t.Parallel()
			b := sequenceBounds(t, 3, 2, 6)
			s := random.New(11)
			for range draws {
				assert.True(t, b.Admits(random.AppendSequence(nil, &s, b)), "a sequence of the bounds")
			}
		})

		t.Run("appends after the elements already in dst", func(t *testing.T) {
			t.Parallel()
			b := sequenceBounds(t, 3, 2, 6)
			s, twin := random.New(pinnedSeed), random.New(pinnedSeed)
			got := random.AppendSequence([]uint32{7, 7}, &s, b)
			want := append([]uint32{7, 7}, random.AppendSequence(nil, &twin, b)...)
			assert.Equal(t, got, want, "the elements of dst, then the draw")
		})
	})
}

// TestSequenceAllocs checks that AppendSequence allocates nothing into
// a slice with the capacity.
func TestSequenceAllocs(t *testing.T) {
	s := random.New(pinnedSeed)
	b := sequenceBounds(t, byteK, 0, 16)
	dst := make([]uint32, 0, 16)
	assert.MaxAllocs(t, func() { _ = random.AppendSequence(dst[:0], &s, b) }, 0,
		"AppendSequence allocates nothing into a slice with the capacity")
}

// BenchmarkSequence measures a byte string draw into a slice with the
// capacity, under a ceiling of no allocation.
func BenchmarkSequence(b *testing.B) {
	b.Run("AppendSequence", func(b *testing.B) {
		var got []uint32
		bounds := sequenceBounds(b, byteK, 0, 16)
		dst := make([]uint32, 0, 16)
		s := random.New(pinnedSeed)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.AppendSequence(dst[:0], &s, bounds)
		}
		assert.True(b, bounds.Admits(got), "a byte string of the bounds")
	})
}

// sequenceBounds returns the bounds of sequences of integers in [0, k)
// with lengths from minSize to maxSize, failing the test when they are
// invalid.
func sequenceBounds(tb testing.TB, k uint32, minSize, maxSize int) choice.SequenceBounds {
	tb.Helper()
	b, err := choice.NewSequenceBounds(k, sizes(tb, minSize, maxSize))
	assert.NoError(tb, err, "the bounds are valid")
	return b
}
