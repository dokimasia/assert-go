// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package token_test

import (
	"encoding/base64"
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/token"
)

// pinnedBytes pins the binary payload of pinnedChoices from the
// definition's executable reference: per choice a tag, then LEB128 numbers
// or the little-endian bits of a float.
const pinnedBytes = "0000" + "0101" + "007f" + "008001" + "00ac02" + "02000000000000f83f" + "030201c801"

// acceptedShortPayloads is the number of payloads of three bytes or fewer,
// with a tag below 5 in a payload of three, that the definition's
// executable reference accepts: 32,897 of 393,473.
const acceptedShortPayloads = 32897

// pinnedChoices are a sequence of every kind whose payload is pinnedBytes.
var pinnedChoices = []choice.Choice{
	integer(0), integer(-1), integer(127), integer(128), integer(300), float(1.5), sequence(1, 200),
}

// lowest is the most negative finite binary64. Its bits are nearly all
// ones, so its token contains the base64url character for 63.
var lowest = float(-math.MaxFloat64)

// TestToken checks the bytes of a token, its round trip, and the tokens
// that no encoder writes.
func TestToken(t *testing.T) {
	t.Parallel()

	t.Run("Append", func(t *testing.T) {
		t.Parallel()

		t.Run("appends the token after the bytes already in dst", func(t *testing.T) {
			t.Parallel()
			got := token.Append([]byte("seen "), pinnedChoices)
			assert.Equal(t, string(got), "seen "+tokenOf(t, pinnedBytes), "the bytes of dst, then the token")
		})
	})

	t.Run("Encode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the pinned payload of every kind in base64url", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, token.Encode(pinnedChoices), tokenOf(t, pinnedBytes), "the token of the pinned choices")
		})

		t.Run("returns the prefix alone for no choices", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, token.Encode(nil), token.Prefix, "a case of targets only")
		})

		t.Run("returns the canonical bits for every NaN", func(t *testing.T) {
			t.Parallel()
			signed := float(math.Float64frombits(0xFFF8000000000001))
			want := tokenOf(t, "02000000000000f87f")
			assert.Equal(t, token.Encode([]choice.Choice{signed}), want, "one token per NaN")
		})

		t.Run("returns base64url without padding", func(t *testing.T) {
			t.Parallel()
			got := token.Encode([]choice.Choice{lowest})
			assert.True(t, strings.Contains(got, "_"), "63 encodes as an underscore")
			assert.False(t, strings.ContainsAny(got, "+/="), "no character of the standard alphabet or padding")
		})
	})

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the pinned choices of the pinned payload", func(t *testing.T) {
			t.Parallel()
			got, err := token.Decode(tokenOf(t, pinnedBytes))
			assert.NoError(t, err, "the token is canonical")
			assert.True(t, sameChoices(got, pinnedChoices), "the pinned choices")
		})

		t.Run("returns the choices that Encode wrote for the extremes of every kind", func(t *testing.T) {
			t.Parallel()
			choices := []choice.Choice{
				integer(math.MinInt64),
				{Kind: choice.Integer, Integer: choice.UintOf(math.MaxUint64)},
				float(math.Copysign(0, -1)),
				float(math.Inf(1)),
				float(math.NaN()),
				float(math.SmallestNonzeroFloat64),
				lowest,
				sequence(),
				sequence(0, 1112063, math.MaxUint32),
			}
			got, err := token.Decode(token.Encode(choices))
			assert.NoError(t, err, "the token is canonical")
			assert.True(t, sameChoices(got, choices), "the choices that were encoded")
		})

		t.Run("returns no choices for the prefix alone", func(t *testing.T) {
			t.Parallel()
			got, err := token.Decode(token.Prefix)
			assert.NoError(t, err, "the token of no choices")
			assert.Empty(t, got, "no choices")
		})

		t.Run("returns the largest uint64 from ten LEB128 bytes", func(t *testing.T) {
			t.Parallel()
			got, err := token.Decode(tokenOf(t, "00ffffffffffffffffff01"))
			assert.NoError(t, err, "the largest number")
			want := []choice.Choice{{Kind: choice.Integer, Integer: choice.UintOf(math.MaxUint64)}}
			assert.True(t, sameChoices(got, want), "the largest uint64")
		})

		t.Run("returns the smallest int64 from the magnitude 2^63", func(t *testing.T) {
			t.Parallel()
			got, err := token.Decode(tokenOf(t, "0180808080808080808001"))
			assert.NoError(t, err, "the largest negative magnitude")
			assert.True(t, sameChoices(got, []choice.Choice{integer(math.MinInt64)}), "the smallest int64")
		})

		t.Run("returns 2^32 - 1 for a sequence element of 2^32 or more", func(t *testing.T) {
			t.Parallel()
			got, err := token.Decode(tokenOf(t, "0302"+"8080808010"+"ffffffffffffffffff01"))
			assert.NoError(t, err, "the definition accepts every element below 2^64")
			want := []choice.Choice{sequence(math.MaxUint32, math.MaxUint32)}
			assert.True(t, sameChoices(got, want), "both elements saturate")
		})

		t.Run("returns only choices that Encode writes as the same token for every short payload", func(t *testing.T) {
			t.Parallel()
			payloads := [][]byte{nil}
			for first := range 256 {
				payloads = append(payloads, []byte{byte(first)})
				for second := range 256 {
					payloads = append(payloads, []byte{byte(first), byte(second)})
				}
			}
			for tag := range 5 {
				for second := range 256 {
					for third := range 256 {
						payloads = append(payloads, []byte{byte(tag), byte(second), byte(third)})
					}
				}
			}
			accepted := 0
			for _, payload := range payloads {
				tok := token.Prefix + base64.RawURLEncoding.EncodeToString(payload)
				got, err := token.Decode(tok)
				if err != nil {
					continue
				}
				accepted++
				if encoded := token.Encode(got); encoded != tok {
					assert.Equal(t, encoded, tok, "the token of the choices that Decode returned")
					return
				}
			}
			assert.Equal(t, accepted, acceptedShortPayloads, "the canonical payloads of three bytes or fewer")
		})

		malformed := []struct {
			name  string
			give  string
			fault string
		}{
			{name: "returns ErrInvalid for another version", give: "prop2:AAA", fault: "does not start with"},
			{name: "returns ErrInvalid for no prefix", give: "AAA", fault: "does not start with"},
			{
				name:  "returns ErrInvalid for padding",
				give:  token.Prefix + "AAA=",
				fault: "not unpadded base64url",
			},
			{
				name:  "returns ErrInvalid for a lone character",
				give:  token.Prefix + "A",
				fault: "not unpadded base64url",
			},
			{
				name:  "returns ErrInvalid for a character outside base64url",
				give:  token.Prefix + "AAA*",
				fault: "not unpadded base64url",
			},
			{
				name:  "returns ErrInvalid for a character outside ASCII",
				give:  token.Prefix + "AAAé",
				fault: "not unpadded base64url",
			},
			{
				name:  "returns ErrInvalid for trailing bits",
				give:  token.Prefix + "AAB",
				fault: "not unpadded base64url",
			},
			{
				name:  "returns ErrInvalid for the standard base64 alphabet",
				give:  strings.ReplaceAll(token.Encode([]choice.Choice{lowest}), "_", "/"),
				fault: "not unpadded base64url",
			},
			{
				name:  "returns ErrInvalid for a line feed",
				give:  token.Prefix + "AA\nA",
				fault: "it contains a line break",
			},
			{
				name:  "returns ErrInvalid for a carriage return",
				give:  token.Prefix + "AA\rA",
				fault: "it contains a line break",
			},
			{
				name:  "returns ErrInvalid for a superfluous LEB128 byte",
				give:  tokenOf(t, "008000"),
				fault: "states 0 in 2 bytes, more than it needs",
			},
			{
				name:  "returns ErrInvalid for a superfluous byte in an element that saturates",
				give:  tokenOf(t, "03018080808090"+"00"),
				fault: "in 6 bytes, more than it needs",
			},
			{
				name:  "returns ErrInvalid for a negative zero",
				give:  tokenOf(t, "0100"),
				fault: "states the negative of 0",
			},
			{
				name:  "returns ErrInvalid for a NaN without the canonical bits",
				give:  tokenOf(t, "02010000000000f87f"),
				fault: "states a NaN with the bits 0x7ff8000000000001",
			},
			{
				name:  "returns ErrInvalid for a magnitude past 2^63",
				give:  tokenOf(t, "0181808080808080808001"),
				fault: "states the negative of 9223372036854775809",
			},
			{
				name:  "returns ErrInvalid for the number 2^64",
				give:  tokenOf(t, "0080808080808080808002"),
				fault: "states a number of 2^64 or more in 10 bytes",
			},
			{
				name:  "returns ErrInvalid for a number cut short",
				give:  tokenOf(t, "0080"),
				fault: "ends inside a number",
			},
			{
				name:  "returns ErrInvalid for a float cut short",
				give:  tokenOf(t, "020000"),
				fault: "ends inside a float, 6 bytes short",
			},
			{
				name:  "returns ErrInvalid for a sequence cut short",
				give:  tokenOf(t, "030201"),
				fault: "ends inside a number",
			},
			{
				name:  "returns ErrInvalid for a sequence length cut short",
				give:  tokenOf(t, "0380"),
				fault: "ends inside a number after 1 bytes",
			},
			{
				name:  "returns ErrInvalid for a sequence length past the payload",
				give:  tokenOf(t, "0380808080808080804001"),
				fault: "ends inside a number",
			},
			{name: "returns ErrInvalid for an unknown tag", give: tokenOf(t, "04"), fault: "states tag 4"},
		}
		for _, tt := range malformed {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := token.Decode(tt.give)
				assert.ErrorIs(t, err, token.ErrInvalid, "the refusal")
				assert.Contains(t, err.Error(), tt.fault, "the fault")
			})
		}
	})
}

// TestTokenZeroAlloc checks that Append allocates nothing into a slice
// with the capacity for the token and its payload.
func TestTokenZeroAlloc(t *testing.T) {
	dst := make([]byte, 0, 128)
	assert.MaxAllocs(t, func() { _ = token.Append(dst[:0], pinnedChoices) }, 0,
		"Append allocates nothing into a slice with the capacity")
}

// BenchmarkToken measures the encoder and the decoder of the pinned
// choices. Append into a slice with the capacity allocates nothing, and
// Encode allocates the token alone. Decode allocates the payload, the
// choices as they grow, and the one sequence.
func BenchmarkToken(b *testing.B) {
	want := tokenOf(b, pinnedBytes)

	b.Run("Append", func(b *testing.B) {
		var got []byte
		dst := make([]byte, 0, 128)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = token.Append(dst[:0], pinnedChoices)
		}
		assert.Equal(b, string(got), want, "the token")
	})

	b.Run("Encode", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got = token.Encode(pinnedChoices)
		}
		assert.Equal(b, got, want, "the token")
	})

	b.Run("Decode", func(b *testing.B) {
		var got []choice.Choice
		c := bench.Start(b).MaxAllocs(6)
		defer c.End()
		for c.Loop() {
			got, _ = token.Decode(want)
		}
		assert.True(b, sameChoices(got, pinnedChoices), "the pinned choices")
	})
}

// tokenOf returns the token of a payload stated in hexadecimal, failing
// the test when the hexadecimal does not parse.
func tokenOf(tb testing.TB, payload string) string {
	tb.Helper()
	data, err := hex.DecodeString(payload)
	assert.NoError(tb, err, "the payload is hexadecimal")
	return token.Prefix + base64.RawURLEncoding.EncodeToString(data)
}

// sameChoices reports whether a and b are the same choices in the same
// order, with floats compared by their bits.
func sameChoices(a, b []choice.Choice) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// integer returns an integer choice of v.
func integer(v int64) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(v)}
}

// float returns a float choice of v.
func float(v float64) choice.Choice {
	return choice.Choice{Kind: choice.Float, Float: v}
}

// sequence returns a sequence choice of the elements.
func sequence(elements ...uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}
