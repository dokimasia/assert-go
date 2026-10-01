// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"fmt"
	"slices"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/prop/alphabet"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// The ids and the sizes of the text generators.
const (
	// stringID is the id of a string.
	stringID = "string"
	// bytesID is the id of a byte string.
	bytesID = "bytes"
	// byteValues is the number of element values of a byte string's
	// sequence.
	byteValues = 256
)

// String returns a generator of strings over the default alphabet, with
// lengths that sizes admits: one sequence of indices into the alphabet.
// Its simplest value is the minimum number of copies of "0".
func String(sizes choice.Sizes) Generator[string] {
	bounds := choice.MustSequenceBounds(alphabet.Size, sizes)
	return newGenerator(stringID, func(c *Case) string {
		span := c.openSpan(stringID)
		defer c.closeSpan(span)
		indices := c.sequence(bounds)
		runes := make([]rune, len(indices))
		for i, index := range indices {
			runes[i] = alphabet.Rune(index)
		}
		return string(runes)
	})
}

// StringOver returns a generator of strings over chars, in their stated
// order, with lengths that sizes admits: one sequence of indices into
// chars. Its simplest value is the minimum number of copies of the first
// character. It panics when chars is empty, is not valid UTF-8, or repeats
// a character.
func StringOver(chars string, sizes choice.Sizes) Generator[string] {
	runes := []rune(chars)
	if len(runes) == 0 || !utf8.ValidString(chars) {
		panic(fmt.Sprintf("prop: alphabet %q is no non-empty string of Unicode scalar values", chars))
	}
	sorted := slices.Clone(runes)
	slices.Sort(sorted)
	if len(slices.Compact(sorted)) != len(runes) {
		panic(fmt.Sprintf("prop: alphabet %q repeats a character", chars))
	}
	bounds := choice.MustSequenceBounds(uint32(len(runes)), sizes)
	return newGenerator(stringID, func(c *Case) string {
		span := c.openSpan(stringID)
		defer c.closeSpan(span)
		indices := c.sequence(bounds)
		out := make([]rune, len(indices))
		for i, index := range indices {
			out[i] = runes[index]
		}
		return string(out)
	})
}

// Bytes returns a generator of byte strings with lengths that sizes
// admits: one sequence with 256 element values. Its simplest value is the
// minimum number of zero bytes.
func Bytes(sizes choice.Sizes) Generator[[]byte] {
	bounds := choice.MustSequenceBounds(byteValues, sizes)
	return newGenerator(bytesID, func(c *Case) []byte {
		span := c.openSpan(bytesID)
		defer c.closeSpan(span)
		elements := c.sequence(bounds)
		out := make([]byte, len(elements))
		for i, element := range elements {
			out[i] = byte(element)
		}
		return out
	})
}
