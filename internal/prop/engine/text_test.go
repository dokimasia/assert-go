// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocations of each text generator's constructor, measured.
const (
	// stringAllocs are the allocations of String and Bytes: the decode, the
	// decode with its type erased, and the inverse.
	stringAllocs = 3
	// stringOverAllocs are the allocations of StringOver: its characters,
	// their sorted copy, the decode, the decode with its type erased, and
	// the inverse.
	stringOverAllocs = 5
)

// TestText checks the string and byte string generators: the values they
// decode, the choices they record, and the alphabets they refuse.
func TestText(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the characters of the default alphabet at the chosen indices", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, engine.String(sizes(t, 0, 4)), sequence(0, 10, 36, 62))
			assert.Equal(t, got, "0aA ", "digits, then lower case, then upper case, then the space")
			assert.Equal(t, labels(e.Case.Spans()), []string{"string"}, "one span")
		})

		t.Run("returns the pinned strings of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.String(sizes(t, 0, 8)), 42, 6)
			assert.Equal(t, values, []string{
				"d\U0010ffffx\U00016326䵙\u008f\U00048eed",
				"1",
				"1\U0010ffff\U000fff7b0",
				"\U0010ffffa\U0003ac840\U0009b44d4a譸",
				"16\U000b9a06ÿ",
				"㩮,d4",
			}, "the strings of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				{sequence(59803, 13, 1112063, 33, 88870, 19801, 143, 296685)},
				{sequence(1)},
				{sequence(1, 1112063, 1046395, 0)},
				{sequence(1112063, 10, 238724, 0, 633933, 4, 10, 35704)},
				{sequence(1, 6, 758278, 255)},
				{sequence(14958, 74, 13, 4)},
			}), "the recorded indices of each case")
		})
	})

	t.Run("StringOver", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			sizes    choice.Sizes
			give     []choice.Choice
			want     string
			recorded []choice.Choice
		}{
			{
				name:     "returns the stated characters at the chosen indices",
				sizes:    sizes(t, 1, 3),
				give:     []choice.Choice{sequence(2, 0)},
				want:     "zx",
				recorded: []choice.Choice{sequence(2, 0)},
			},
			{
				name:     "returns a recorded sequence fitted to its bounds",
				sizes:    sizes(t, 3, 3),
				give:     []choice.Choice{sequence(5, 1)},
				want:     "xyx",
				recorded: []choice.Choice{sequence(0, 1, 0)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, engine.StringOver("xyz", tt.sizes), tt.give...)
				assert.Equal(t, got, tt.want, "the string")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded indices")
			})
		}

		t.Run("returns the pinned strings of seed 7", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.StringOver("ACGT", sizes(t, 2, 6)), 7, 6)
			assert.Equal(t, values, []string{"TT", "AA", "GC", "TG", "ACC", "GGCGCC"},
				"the strings of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				{sequence(3, 3)},
				{sequence(0, 0)},
				{sequence(2, 1)},
				{sequence(3, 2)},
				{sequence(0, 1, 1)},
				{sequence(2, 2, 1, 2, 1, 1)},
			}), "the recorded indices of each case")
		})

		t.Run("returns characters outside the default alphabet's first block", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, engine.StringOver("é😀", sizes(t, 0, 3)), sequence(1, 0, 1))
			assert.Equal(t, got, "😀é😀", "the characters, not their bytes")
		})

		refusals := []struct {
			name  string
			chars string
		}{
			{name: "panics for no character", chars: ""},
			{name: "panics for a string that is not valid UTF-8", chars: "a\xffb"},
			{name: "panics for a repeated character", chars: "aba"},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Panics(t, func() { engine.StringOver(tt.chars, sizes(t, 0, 3)) }, "no alphabet")
			})
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			sizes    choice.Sizes
			give     []choice.Choice
			want     []byte
			recorded []choice.Choice
		}{
			{
				name:     "returns a byte for each element",
				sizes:    unbounded(t, 0),
				give:     []choice.Choice{sequence(104, 105)},
				want:     []byte("hi"),
				recorded: []choice.Choice{sequence(104, 105)},
			},
			{
				name:     "returns the first bytes of a sequence longer than its maximum length",
				sizes:    sizes(t, 0, 2),
				give:     []choice.Choice{sequence(1, 2, 3)},
				want:     []byte{1, 2},
				recorded: []choice.Choice{sequence(1, 2)},
			},
			{
				name:     "returns the minimum number of zero bytes past the last choice",
				sizes:    unbounded(t, 3),
				want:     []byte{0, 0, 0},
				recorded: []choice.Choice{sequence(0, 0, 0)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, e := decode(t, engine.Bytes(tt.sizes), tt.give...)
				assert.Equal(t, got, tt.want, "the byte string")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded elements")
				assert.Equal(t, labels(e.Case.Spans()), []string{"bytes"}, "one span")
			})
		}

		t.Run("returns the pinned byte strings of seed 42", func(t *testing.T) {
			t.Parallel()
			values, choices := generated(engine.Bytes(sizes(t, 0, 8)), 42, 6)
			assert.Equal(t, values, [][]byte{
				{0xe9, 0x0d, 0xff, 0x21, 0x8f, 0x02},
				{0x01},
				{0x01, 0xff, 0x7f, 0x00},
				{0xff, 0x0a, 0x1d, 0x00, 0x4d, 0x04, 0x0a, 0x8b},
				{0x01, 0x06, 0x5c, 0xff},
				{0x3a, 0x4a, 0x0d, 0x04},
			}, "the byte strings of the first six cases")
			assert.True(t, sameRecords(choices, [][]choice.Choice{
				{sequence(233, 13, 255, 33, 143, 2)},
				{sequence(1)},
				{sequence(1, 255, 127, 0)},
				{sequence(255, 10, 29, 0, 77, 4, 10, 139)},
				{sequence(1, 6, 92, 255)},
				{sequence(58, 74, 13, 4)},
			}), "the recorded elements of each case")
		})

		t.Run("returns the pinned byte strings of one length of seed 7", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(engine.Bytes(sizes(t, 4, 4)), 7, 3)
			assert.Equal(t, values, [][]byte{
				{0xf8, 0xdb, 0x5f, 0x0c}, {0x23, 0x05, 0x0e, 0xff}, {0x91, 0x45, 0xc9, 0x00},
			}, "the byte strings of the first three cases")
		})
	})
}

// TestTextAllocs checks the allocation ceilings of the text generators'
// constructors.
func TestTextAllocs(t *testing.T) {
	upTo := sizes(t, 0, 8)
	assert.MaxAllocs(t, func() { _ = engine.String(upTo) }, stringAllocs, "String allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.StringOver("ACGT", upTo) }, stringOverAllocs,
		"StringOver allocates its characters and its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Bytes(upTo) }, stringAllocs, "Bytes allocates its decodes")
}

// BenchmarkText measures the constructors of the text generators.
func BenchmarkText(b *testing.B) {
	upTo := sizes(b, 0, 8)

	b.Run("String", func(b *testing.B) {
		var got engine.Generator[string]
		c := bench.Start(b).MaxAllocs(stringAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.String(upTo)
		}
		assert.Equal(b, got.ID(), "string", "the id")
	})

	b.Run("StringOver", func(b *testing.B) {
		var got engine.Generator[string]
		c := bench.Start(b).MaxAllocs(stringOverAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.StringOver("ACGT", upTo)
		}
		assert.Equal(b, got.ID(), "string", "the id")
	})

	b.Run("Bytes", func(b *testing.B) {
		var got engine.Generator[[]byte]
		c := bench.Start(b).MaxAllocs(stringAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Bytes(upTo)
		}
		assert.Equal(b, got.ID(), "bytes", "the id")
	})
}
