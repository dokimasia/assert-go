// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"encoding/hex"
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestBridge checks how a fuzzer's bytes decode into the choices of a
// case: the bytes each kind of choice reads, and the targets once the bytes
// run out.
func TestBridge(t *testing.T) {
	t.Parallel()

	t.Run("Bridge", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			g        engine.Generator[any]
			data     string
			want     any
			recorded []choice.Choice
		}{
			{
				name:     "reads the fewest bytes that cover an integer range",
				g:        anyOf(engine.Integer(0, 1000)),
				data:     "e803",
				want:     1000,
				recorded: integers(1000),
			},
			{
				name:     "reduces an integer modulo the size of its range",
				g:        anyOf(engine.Integer(-5, 5)),
				data:     "0c",
				want:     -4,
				recorded: integers(-4),
			},
			{
				name:     "reads eight bytes for the whole unsigned range",
				g:        anyOf(engine.Integer[uint64](0, math.MaxUint64)),
				data:     "ffffffffffffffff",
				want:     uint64(math.MaxUint64),
				recorded: []choice.Choice{unsigned(math.MaxUint64)},
			},
			{
				name:     "reads no byte for a choice with one value",
				g:        anyOf(engine.List(engine.Integer(0, 9), sizes(t, 2, 2))),
				data:     "0307",
				want:     []int{3, 7},
				recorded: integers(1, 3, 1, 7, 0),
			},
			{
				name:     "reads a float as eight little-endian bytes",
				g:        anyOf(engine.Float(math.Inf(-1), math.Inf(1), choice.ExcludeNaN)),
				data:     "000000000000f83f",
				want:     1.5,
				recorded: []choice.Choice{float(1.5)},
			},
			{
				name:     "reads a float of width 32 as four bytes",
				g:        anyOf(engine.Float[float32](0, 10, choice.ExcludeNaN)),
				data:     "0000c03f",
				want:     float32(1.5),
				recorded: []choice.Choice{float(1.5)},
			},
			{
				name:     "returns the target for a float outside its bounds",
				g:        anyOf(engine.Float(0.0, 1.0, choice.ExcludeNaN)),
				data:     "0000000000000040",
				want:     0.0,
				recorded: []choice.Choice{float(0)},
			},
			{
				name:     "reads a float of one value that admits NaN",
				g:        anyOf(engine.Float(2.5, 2.5, choice.AdmitNaN)),
				data:     "000000000000f87f",
				want:     math.NaN(),
				recorded: []choice.Choice{float(math.NaN())},
			},
			{
				name:     "reads a byte string one byte per byte after two bytes of length",
				g:        anyOf(engine.Bytes(unbounded(t, 0))),
				data:     "050068656c6c6f",
				want:     []byte("hello"),
				recorded: []choice.Choice{sequence(104, 101, 108, 108, 111)},
			},
			{
				name:     "ends a sequence at its last whole element and fills it to the minimum length",
				g:        anyOf(engine.StringOver("ab", sizes(t, 3, 10))),
				data:     "0501",
				want:     "baa",
				recorded: []choice.Choice{sequence(1, 0, 0)},
			},
			{
				name:     "reads a bounded length as an offset from the minimum modulo the spread",
				g:        anyOf(engine.StringOver("ab", sizes(t, 3, 10))),
				data:     "09" + "000000000000000000000000",
				want:     "aaaa",
				recorded: []choice.Choice{sequence(0, 0, 0, 0)},
			},
			{
				name:     "returns the target of a sequence whose length the bytes cannot state",
				g:        anyOf(engine.Bytes(unbounded(t, 2))),
				data:     "05",
				want:     []byte{0, 0},
				recorded: []choice.Choice{sequence(0, 0)},
			},
			{
				name:     "returns targets once the bytes run out",
				g:        anyOf(engine.List(engine.Integer(0, 65535), sizes(t, 0, 3))),
				data:     "0134",
				want:     []int{0},
				recorded: integers(1, 0, 0),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				data, err := hex.DecodeString(tt.data)
				assert.NoError(t, err, "the bytes are hexadecimal")
				var got any
				e := engine.Bridge(func(c *engine.Case) { got = engine.Draw(c, tt.g, drawn) }, data, engine.Settings{})
				assert.Equal(t, got, tt.want, "the decoded value", assert.EquateNaNs())
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded choices")
			})
		}

		floatThenByte := func(lo, hi float64, data []byte) [2]any {
			g, b := engine.Float(lo, hi, choice.ExcludeNaN), engine.Integer(0, 255)
			var got [2]any
			body := func(c *engine.Case) { got = [2]any{engine.Draw(c, g, "float"), engine.Draw(c, b, "byte")} }
			engine.Bridge(body, data, engine.Settings{})
			return got
		}

		t.Run("reads no byte for a float of one nonzero value without NaN", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, floatThenByte(2.5, 2.5, []byte{7}), [2]any{2.5, 7}, "the byte is the integer's")
		})

		t.Run("reads eight bytes for a float of the one value zero", func(t *testing.T) {
			t.Parallel()
			got := floatThenByte(0, 0, []byte{0, 0, 0, 0, 0, 0, 0, 0, 7})
			assert.Equal(t, got, [2]any{0.0, 7}, "the float reads the first eight bytes")
		})

		t.Run("returns the target of a float that fewer bytes than its width remain for", func(t *testing.T) {
			t.Parallel()
			got := floatThenByte(1, 2, []byte{1, 2, 3})
			assert.Equal(t, got, [2]any{1.0, 0}, "the float's target, then the integer's")
		})
	})
}
