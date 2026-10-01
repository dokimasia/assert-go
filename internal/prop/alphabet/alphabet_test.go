// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package alphabet_test

import (
	"slices"
	"strconv"
	"testing"
	"unicode"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/alphabet"
)

// The ends of the surrogates, which are code points but no Unicode scalar
// values.
const (
	firstSurrogate = 0xD800
	lastSurrogate  = 0xDFFF
)

// printable pins the first 95 characters of the default alphabet, in their
// order.
const printable = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

// TestAlphabet checks the order of the default alphabet, its size, and the
// mappings between characters and indices.
func TestAlphabet(t *testing.T) {
	t.Parallel()

	t.Run("Rune", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the printable characters at indices 0 to 94", func(t *testing.T) {
			t.Parallel()
			got := make([]rune, len(printable))
			for i := range got {
				got[i] = alphabet.Rune(uint32(i))
			}
			assert.Equal(t, string(got), printable, "the printable characters in order")
		})

		tests := []struct {
			name string
			give uint32
			want rune
		}{
			{name: "returns U+0000 at index 95", give: 95, want: 0x00},
			{name: "returns U+001F at index 126", give: 126, want: 0x1F},
			{name: "returns DEL at index 127", give: 127, want: 0x7F},
			{name: "returns U+D7FF at index 55295", give: 55295, want: 0xD7FF},
			{name: "returns U+E000 at index 55296", give: 55296, want: 0xE000},
			{name: "returns U+FFFF at index 63487", give: 63487, want: 0xFFFF},
			{name: "returns U+10000 at index 63488", give: 63488, want: 0x10000},
			{name: "returns U+10FFFF at the last index", give: alphabet.Size - 1, want: unicode.MaxRune},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, alphabet.Rune(tt.give), tt.want, "the character at the index")
			})
		}

		t.Run("panics at Size", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { alphabet.Rune(alphabet.Size) }, "an index past the end")
		})
	})

	t.Run("Index", func(t *testing.T) {
		t.Parallel()

		t.Run("returns every scalar value at an index of its own that Rune inverts", func(t *testing.T) {
			t.Parallel()
			seen := make([]bool, alphabet.Size)
			var wrong []rune
			for r := rune(0); r <= unicode.MaxRune; r++ {
				if firstSurrogate <= r && r <= lastSurrogate {
					continue
				}
				i, ok := alphabet.Index(r)
				if !ok || i >= alphabet.Size || seen[i] || alphabet.Rune(i) != r {
					wrong = append(wrong, r)
					continue
				}
				seen[i] = true
			}
			assert.Empty(t, wrong, "the scalar values without an index of their own")
			assert.False(t, slices.Contains(seen, false), "every index belongs to a scalar value")
		})

		tests := []struct {
			name string
			give rune
		}{
			{name: "reports false for the first surrogate", give: firstSurrogate},
			{name: "reports false for the last surrogate", give: lastSurrogate},
			{name: "reports false for -1", give: -1},
			{name: "reports false above U+10FFFF", give: unicode.MaxRune + 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, ok := alphabet.Index(tt.give)
				assert.False(t, ok, "no Unicode scalar value")
			})
		}
	})

	t.Run("AppendIndices", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the indices of the scalar values of a range", func(t *testing.T) {
			t.Parallel()
			for _, r := range [][2]rune{
				{'a', 'z'},
				{' ', '~'},
				{0x00, 0x7F},
				{0x1F, 0x21},
				{0xD7FE, 0xE001},
				{0xFFFE, 0x10001},
				{0x41, 0x40},
			} {
				msg := "the indices of [" + strconv.QuoteRune(r[0]) + ", " + strconv.QuoteRune(r[1]) + "]"
				assert.Equal(t, expand(alphabet.AppendIndices(nil, r[0], r[1])), indicesOf(r[0], r[1]), msg)
			}
		})

		tests := []struct {
			name   string
			lo, hi rune
			want   []alphabet.Interval
		}{
			{
				name: "returns the lowercase letters as one interval",
				lo:   'a',
				hi:   'z',
				want: []alphabet.Interval{{First: 10, Last: 35}},
			},
			{
				name: "returns the digits to z as the letters and digits then the punctuation between them",
				lo:   '0',
				hi:   'z',
				want: []alphabet.Interval{{First: 0, Last: 61}, {First: 78, Last: 90}},
			},
			{name: "returns nothing for a range whose lo exceeds its hi", lo: 'B', hi: 'A', want: nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, alphabet.AppendIndices(nil, tt.lo, tt.hi), tt.want, "the intervals of the range")
			})
		}

		t.Run("appends after the intervals already in dst without merging them", func(t *testing.T) {
			t.Parallel()
			got := alphabet.AppendIndices([]alphabet.Interval{{First: 36, Last: 36}}, 'a', 'z')
			want := []alphabet.Interval{{First: 36, Last: 36}, {First: 10, Last: 35}}
			assert.Equal(t, got, want, "the interval of dst, then the lowercase letters")
		})
	})
}

// TestAlphabetZeroAlloc checks that Rune and Index allocate nothing, and
// that AppendIndices allocates nothing into a slice with the capacity.
func TestAlphabetZeroAlloc(t *testing.T) {
	dst := make([]alphabet.Interval, 0, 128)
	assert.MaxAllocs(t, func() { _ = alphabet.Rune(70000) }, 0, "Rune allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = alphabet.Index(0x1F600) }, 0, "Index allocates nothing")
	assert.MaxAllocs(t, func() { _ = alphabet.AppendIndices(dst[:0], 0, unicode.MaxRune) }, 0,
		"AppendIndices allocates nothing into a slice with the capacity")
}

// BenchmarkAlphabet measures Rune, Index and AppendIndices under a ceiling
// of no allocation.
func BenchmarkAlphabet(b *testing.B) {
	b.Run("Rune", func(b *testing.B) {
		var got rune
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = alphabet.Rune(alphabet.Size - 1)
		}
		assert.Equal(b, got, rune(unicode.MaxRune), "the last character")
	})

	b.Run("Index", func(b *testing.B) {
		var got uint32
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = alphabet.Index(unicode.MaxRune)
		}
		assert.Equal(b, got, uint32(alphabet.Size-1), "the last index")
	})

	b.Run("AppendIndices", func(b *testing.B) {
		var got []alphabet.Interval
		dst := make([]alphabet.Interval, 0, 128)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = alphabet.AppendIndices(dst[:0], 0, unicode.MaxRune)
		}
		assert.Equal(b, got, []alphabet.Interval{{First: 0, Last: alphabet.Size - 1}}, "every index")
	})
}

// expand returns every index of the intervals, in order.
func expand(intervals []alphabet.Interval) []uint32 {
	var out []uint32
	for _, v := range intervals {
		for i := v.First; i <= v.Last; i++ {
			out = append(out, i)
		}
	}
	return out
}

// indicesOf returns the indices of the scalar values in [lo, hi], sorted,
// one at a time.
func indicesOf(lo, hi rune) []uint32 {
	var out []uint32
	for r := lo; r <= hi; r++ {
		if i, ok := alphabet.Index(r); ok {
			out = append(out, i)
		}
	}
	slices.Sort(out)
	return out
}
