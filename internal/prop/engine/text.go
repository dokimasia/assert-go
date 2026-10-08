// Copyright Dokimasia B.V. 2026
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
// Its simplest value is the minimum number of copies of "0". It runs
// backwards from a string of UTF-8, through the index of each character.
func String(sizes choice.Sizes) Generator[string] {
	bounds := choice.MustSequenceBounds(alphabet.Size, sizes)
	decode := func(c *Case) string {
		span := c.openSpan(stringID)
		defer c.closeSpan(span)
		indices := c.sequence(bounds)
		runes := make([]rune, len(indices))
		for i, index := range indices {
			runes[i] = alphabet.Rune(index)
		}
		return string(runes)
	}
	return NewInvertible(stringID, decode, func(v any) ([]Step, string, error) {
		return invertText(bounds, v, alphabet.Index)
	})
}

// StringOver returns a generator of strings over chars, in their stated
// order, with lengths that sizes admits: one sequence of indices into
// chars. Its simplest value is the minimum number of copies of the first
// character. It runs backwards from a string of chars, through the index of
// each character in chars. It panics when chars is empty, is not valid
// UTF-8, or repeats a character.
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
	decode := func(c *Case) string {
		span := c.openSpan(stringID)
		defer c.closeSpan(span)
		indices := c.sequence(bounds)
		out := make([]rune, len(indices))
		for i, index := range indices {
			out[i] = runes[index]
		}
		return string(out)
	}
	return NewInvertible(stringID, decode, func(v any) ([]Step, string, error) {
		return invertText(bounds, v, func(r rune) (uint32, bool) {
			i := slices.Index(runes, r)
			return uint32(i), i >= 0
		})
	})
}

// invertText returns the one step of a string under bounds that decodes to
// v: the sequence of the index of each character, which index returns.
func invertText(bounds choice.SequenceBounds, v any, index func(rune) (uint32, bool)) ([]Step, string, error) {
	s, ok := v.(string)
	if !ok || !utf8.ValidString(s) {
		return nil, "", uninvertible("%v is no string of UTF-8", v)
	}
	var indices []uint32
	for _, r := range s {
		i, ok := index(r)
		if !ok {
			return nil, "", uninvertible("%q has the character %q outside the alphabet", s, r)
		}
		indices = append(indices, i)
	}
	step, err := sequenceStep(bounds, indices)
	return []Step{step}, s, err
}

// Bytes returns a generator of byte strings with lengths that sizes
// admits: one sequence with 256 element values. Its simplest value is the
// minimum number of zero bytes. It runs backwards from a []byte, through
// its bytes in order.
func Bytes(sizes choice.Sizes) Generator[[]byte] {
	bounds := choice.MustSequenceBounds(byteValues, sizes)
	decode := func(c *Case) []byte {
		span := c.openSpan(bytesID)
		defer c.closeSpan(span)
		elements := c.sequence(bounds)
		out := make([]byte, len(elements))
		for i, element := range elements {
			out[i] = byte(element)
		}
		return out
	}
	return NewInvertible(bytesID, decode, func(v any) ([]Step, []byte, error) {
		b, ok := v.([]byte)
		if !ok {
			return nil, nil, uninvertible("%v is no []byte", v)
		}
		elements := make([]uint32, len(b))
		for i, x := range b {
			elements[i] = uint32(x)
		}
		step, err := sequenceStep(bounds, elements)
		return []Step{step}, b, err
	})
}

// sequenceStep returns the step of the sequence s under bounds, and a fault
// of the kind ErrCannotInvert when its length is outside the bounds' sizes.
// Every element of s is below the bounds' K.
func sequenceStep(bounds choice.SequenceBounds, s []uint32) (Step, error) {
	value := choice.Choice{Kind: choice.Sequence, Sequence: s}
	if !bounds.Admits(s) {
		return Step{}, uninvertible("%d elements are outside the sizes of the sequence", len(s))
	}
	return Step{Bounds: choice.OfSequence(bounds), Value: value}, nil
}
