// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package token

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// Prefix is the text that every token of this version starts with.
const Prefix = "prop1:"

// The tag byte before each choice.
const (
	// tagNonNegative precedes a non-negative integer.
	tagNonNegative = 0
	// tagNegative precedes a negative integer.
	tagNegative = 1
	// tagFloat precedes a float.
	tagFloat = 2
	// tagSequence precedes a sequence.
	tagSequence = 3
)

// The sizes of a payload and of a token.
const (
	// floatBytes is the number of bytes of a float's payload.
	floatBytes = 8
	// maxNegative is 2^63, the largest magnitude of a negative integer.
	maxNegative = 9223372036854775808
	// stackBytes is the size of the buffer on the stack that Encode writes
	// a token in, and that Decode decodes a payload into. A longer token
	// grows into the heap.
	stackBytes = 128
	// stackText is the size of the buffer on the stack that Decode copies a
	// token's text into: the unpadded base64url text of stackBytes.
	stackText = 171
	// stackChoices is the number of choices that Decode reads on the stack
	// before it copies them out. A token of more choices grows into the
	// heap.
	stackChoices = 16
)

// lineBreaks are the characters that the base64 decoder skips, and that no
// encoder writes.
const lineBreaks = "\r\n"

// ErrInvalid reports a token that no encoder writes.
var ErrInvalid = errors.New("token: not a token that an encoder writes")

// encoding is the unpadded base64url encoding of a token's payload. It
// refuses padding, characters outside its alphabet and trailing bits that
// are not zero.
var encoding = base64.RawURLEncoding.Strict()

// Append appends the token of choices to dst and returns the extended
// slice.
//
// The binary payload of the token is written past the text that encodes
// it, so dst needs capacity for both. Append allocates only when dst lacks
// that capacity.
func Append(dst []byte, choices []choice.Choice) []byte {
	dst = append(dst, Prefix...)
	start := len(dst)
	dst = appendPayload(dst, choices)
	n := len(dst) - start
	m := encoding.EncodedLen(n)
	dst = slices.Grow(dst, m)[:start+m+n]
	copy(dst[start+m:], dst[start:start+n])
	encoding.Encode(dst[start:start+m], dst[start+m:])
	return dst[:start+m]
}

// Encode returns the token of choices. It allocates the token, and the
// buffer it writes the token in when the token and its payload exceed 128
// bytes.
func Encode(choices []choice.Choice) string {
	var buf [stackBytes]byte
	return string(Append(buf[:0], choices))
}

// Decode returns the choices that tok records. It returns an error that
// wraps [ErrInvalid] when tok is not the token that [Append] writes for
// any choices: a token without [Prefix], text outside unpadded base64url,
// an unknown tag, a payload cut short, a number of 2^64 or more, a
// negative integer below -2^63, a number with superfluous LEB128 bytes, a
// negative zero, or a NaN without the canonical bits. It checks each of
// these on the bytes as it reads them.
//
// A sequence element of 2^32 or more decodes as 2^32 - 1, because a
// [choice.Choice] stores each element in 32 bits. Every sequence has fewer
// than 2^32 element values, so a replay fits that element to 0, as it fits
// the element that the token states.
//
// Decode allocates the choices it returns, an empty list for the prefix
// alone, and the elements of each sequence. It decodes the text, the
// payload and the first 16 choices in buffers on the stack, and allocates
// for a token past them.
func Decode(tok string) ([]choice.Choice, error) {
	text, ok := strings.CutPrefix(tok, Prefix)
	if !ok {
		return nil, fmt.Errorf("%w: %q does not start with %s", ErrInvalid, tok, Prefix)
	}
	if strings.ContainsAny(text, lineBreaks) {
		return nil, fmt.Errorf("%w: %q is not unpadded base64url: it contains a line break", ErrInvalid, tok)
	}
	var src [stackText]byte
	var dst [stackBytes]byte
	data, err := encoding.AppendDecode(dst[:0], append(src[:0], text...))
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not unpadded base64url: %w", ErrInvalid, tok, err)
	}
	var stack [stackChoices]choice.Choice
	choices := stack[:0]
	for len(data) > 0 {
		var c choice.Choice
		c, data, err = readChoice(data)
		if err != nil {
			return nil, fmt.Errorf("%w: %q %w", ErrInvalid, tok, err)
		}
		choices = append(choices, c)
	}
	return append(make([]choice.Choice, 0, len(choices)), choices...), nil
}

// appendPayload appends the binary payload of choices to dst.
func appendPayload(dst []byte, choices []choice.Choice) []byte {
	for _, c := range choices {
		if c.Kind == choice.Integer {
			tag := byte(tagNonNegative)
			if c.Integer.Negative() {
				tag = tagNegative
			}
			dst = binary.AppendUvarint(append(dst, tag), c.Integer.Magnitude())
			continue
		}
		if c.Kind == choice.Float {
			bits := math.Float64bits(c.Float)
			if math.IsNaN(c.Float) {
				bits = choice.NaNBits
			}
			dst = binary.LittleEndian.AppendUint64(append(dst, tagFloat), bits)
			continue
		}
		dst = binary.AppendUvarint(append(dst, tagSequence), uint64(len(c.Sequence)))
		for _, element := range c.Sequence {
			dst = binary.AppendUvarint(dst, uint64(element))
		}
	}
	return dst
}

// readChoice reads one choice from the front of data and returns it with
// the bytes that follow it. It refuses a negative zero and a NaN without
// the canonical bits, which no encoder writes.
func readChoice(data []byte) (choice.Choice, []byte, error) {
	tag, data := data[0], data[1:]
	if tag == tagNonNegative || tag == tagNegative {
		magnitude, rest, err := readNumber(data)
		if err != nil {
			return choice.Choice{}, nil, err
		}
		if tag == tagNonNegative {
			return choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(magnitude)}, rest, nil
		}
		if magnitude == 0 || magnitude > maxNegative {
			return choice.Choice{}, nil, fmt.Errorf("states the negative of %d", magnitude)
		}
		return choice.Choice{Kind: choice.Integer, Integer: choice.Int{}.Sub(magnitude)}, rest, nil
	}
	if tag == tagFloat {
		if len(data) < floatBytes {
			return choice.Choice{}, nil, fmt.Errorf("ends inside a float, %d bytes short", floatBytes-len(data))
		}
		bits := binary.LittleEndian.Uint64(data)
		value := math.Float64frombits(bits)
		if math.IsNaN(value) && bits != choice.NaNBits {
			return choice.Choice{}, nil, fmt.Errorf("states a NaN with the bits %#x", bits)
		}
		return choice.Choice{Kind: choice.Float, Float: value}, data[floatBytes:], nil
	}
	if tag == tagSequence {
		return readSequence(data)
	}
	return choice.Choice{}, nil, fmt.Errorf("states tag %d", tag)
}

// readSequence reads the length and the elements of a sequence from the
// front of data, and returns the sequence with the bytes that follow it. An
// element of 2^32 or more becomes 2^32 - 1.
func readSequence(data []byte) (choice.Choice, []byte, error) {
	length, data, err := readNumber(data)
	if err != nil {
		return choice.Choice{}, nil, err
	}
	// Every element takes at least one byte, so the bytes left bound the
	// slice that a stated length can make the reader allocate.
	elements := make([]uint32, 0, min(length, uint64(len(data))))
	for range length {
		var element uint64
		element, data, err = readNumber(data)
		if err != nil {
			return choice.Choice{}, nil, err
		}
		elements = append(elements, uint32(min(element, math.MaxUint32)))
	}
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}, data, nil
}

// readNumber reads one unsigned LEB128 number from the front of data, and
// returns it with the bytes that follow it. A number of more than one byte
// whose last byte is 0 has a superfluous byte, which no encoder writes.
func readNumber(data []byte) (uint64, []byte, error) {
	number, n := binary.Uvarint(data)
	if n < 0 {
		return 0, nil, fmt.Errorf("states a number of 2^64 or more in %d bytes", -n)
	}
	if n == 0 {
		return 0, nil, fmt.Errorf("ends inside a number after %d bytes", len(data))
	}
	if n > 1 && data[n-1] == 0 {
		return 0, nil, fmt.Errorf("states %d in %d bytes, more than it needs", number, n)
	}
	return number, data[n:], nil
}
