// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// The allocations of the text generators, measured. Each is the engine's
// construction of the generator: the closures of its decodes and of its
// inverse, and what it keeps of its alphabet or its pattern.
const (
	// stringAllocs are the allocations of String over the default alphabet.
	stringAllocs = 4
	// alphabetStringAllocs are the allocations of String over a stated
	// alphabet: those of String, and the alphabet's characters with the
	// sorted copy that checks them for a repeat.
	alphabetStringAllocs = 6
	// alphabetAllocs are the allocations of Alphabet.
	alphabetAllocs = 0
	// bytesAllocs are the allocations of Bytes.
	bytesAllocs = 4
	// stringMatchingAllocs are the allocations of StringMatching for the
	// pattern [a-c]{2,5}: the parsed pattern, the decoder built from it, and
	// the inverse.
	stringMatchingAllocs = 14
)

// outsidePrefix starts the panic of StringMatching for a pattern outside
// the portable subset: the package and the generator.
const outsidePrefix = "prop: string-matching: "

// TestText checks the string and byte string generators and the option
// that states an alphabet.
func TestText(t *testing.T) {
	t.Parallel()

	t.Run("Alphabet", func(t *testing.T) {
		t.Parallel()

		t.Run("makes String choose from its characters in their stated order", func(t *testing.T) {
			t.Parallel()
			got := replayedOf(prop.String(prop.Alphabet("xyz")), sequence(2, 0, 1))
			assert.Equal(t, got, "zxy", "the characters at the replayed indices")
		})

		t.Run("makes its first character the simplest", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(prop.String(prop.Alphabet("xyz"), prop.MinSize(2))), "xx", "the simplest string")
		})

		t.Run("makes a later alphabet override an earlier one", func(t *testing.T) {
			t.Parallel()
			got := first(prop.String(prop.Alphabet("xyz"), prop.Alphabet("pq"), prop.MinSize(1)))
			assert.Equal(t, got, "p", "the first character of the later alphabet")
		})

		tests := []struct {
			name string
			give string
		}{
			{name: "makes String panic for no character", give: ""},
			{name: "makes String panic for a string that is not UTF-8", give: "\xff"},
			{name: "makes String panic for a repeated character", give: "aba"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := assert.Panics(t, func() { prop.String(prop.Alphabet(tt.give)) }, "the alphabet is refused")
				assert.Contains(t, got, "prop: alphabet", "the panic names the alphabet")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the characters of the default alphabet at the replayed indices", func(t *testing.T) {
			t.Parallel()
			got := replayedOf(prop.String(), sequence(0, 10, 36))
			assert.Equal(t, got, "0aA", "a digit, a lowercase and an uppercase letter")
		})

		t.Run("returns the empty string as the simplest", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(prop.String()), "", "the empty string")
		})
	})

	t.Run("Bytes", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the replayed elements as bytes", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, replayedOf(prop.Bytes(), sequence(0, 255, 7)), []byte{0, 255, 7}, "three bytes")
		})
	})

	t.Run("StringMatching", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{name: "returns the fewest repetitions of the simplest character", give: "[a-c]{2,5}", want: "aa"},
			{name: "returns the simplest member of a class in the default alphabet's order", give: "[A0a]", want: "0"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(prop.StringMatching(tt.give)), tt.want, "the simplest match")
			})
		}

		t.Run("panics for a pattern outside the portable subset", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.StringMatching("a{01}") }, "a count with a leading zero")
			assert.Equal(t, got, outsidePrefix+`"a{01}" at 4: the count 01 has a leading zero`,
				"the panic states the generator and the pattern's fault")
		})
	})
}

// TestTextAllocs checks the allocation ceilings of the text generators
// and the alphabet option.
func TestTextAllocs(t *testing.T) {
	xyz := prop.Alphabet("xyz")
	var kept prop.StringOption
	assert.MaxAllocs(t, func() { kept = prop.Alphabet("xyz") }, alphabetAllocs, "Alphabet allocates its option")
	assert.MaxAllocs(t, func() { _ = prop.String() }, stringAllocs, "String allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.String(xyz) }, alphabetStringAllocs,
		"String allocates its decode and the alphabet's characters")
	assert.MaxAllocs(t, func() { _ = prop.Bytes() }, bytesAllocs, "Bytes allocates its decode")
	assert.MaxAllocs(t, func() { _ = prop.StringMatching("[a-c]{2,5}") }, stringMatchingAllocs,
		"StringMatching allocates the parsed pattern")
	assert.NotNil(t, kept, "the kept option")
}

// BenchmarkText measures each text generator and the alphabet option.
func BenchmarkText(b *testing.B) {
	b.Run("Alphabet", func(b *testing.B) {
		var got prop.StringOption
		c := bench.Start(b).MaxAllocs(alphabetAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Alphabet("xyz")
		}
		assert.Equal(b, first(prop.String(got, prop.MinSize(1))), "x", "the simplest string of the alphabet")
	})

	b.Run("String", func(b *testing.B) {
		var got prop.Generator[string]
		c := bench.Start(b).MaxAllocs(stringAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.String()
		}
		assert.Equal(b, replayedOf(got, sequence(0, 10)), "0a", "the string")
	})

	b.Run("Bytes", func(b *testing.B) {
		var got prop.Generator[[]byte]
		c := bench.Start(b).MaxAllocs(bytesAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Bytes()
		}
		assert.Equal(b, replayedOf(got, sequence(7)), []byte{7}, "the byte string")
	})

	b.Run("StringMatching", func(b *testing.B) {
		var got prop.Generator[string]
		c := bench.Start(b).MaxAllocs(stringMatchingAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.StringMatching("[a-c]{2,5}")
		}
		assert.Equal(b, first(got), "aa", "the simplest match")
	})
}
