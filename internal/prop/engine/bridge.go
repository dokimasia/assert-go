// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"encoding/binary"
	"math"
	"math/bits"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// The rules of the fuzz bridge.
const (
	// unboundedLengths is the number of lengths beyond its minimum that a
	// sequence without a maximum length may take, read from two bytes.
	unboundedLengths = 0xFFFF
	// byteBits is the number of bits of one byte.
	byteBits = 8
)

// bridging is a provider that decodes every value from a fuzzer's bytes,
// in request order, so every byte string decodes to a valid case.
//
//   - An integer in [lo, hi] reads ceil(bits.Len64(hi - lo) / 8) bytes as a
//     little-endian unsigned u, and takes lo + u mod (hi - lo + 1). Bounds
//     of one value read no byte.
//   - A float reads 8 bytes, or 4 at width 32, little-endian, as the bits
//     of a value of its width, fitted to its bounds by [choice.FloatBounds]
//     coercion. Bounds of one nonzero value without NaN read no byte.
//   - A sequence reads its length as an integer from its minimum to its
//     maximum, or to its minimum plus 65,535 without a maximum, and then
//     each element as an integer in [0, k - 1].
//   - A value that needs more bytes than remain takes its target, and the
//     rest of the bytes are spent, so every later value takes its target
//     too. An element cut short ends its sequence, which zeros extend to
//     the minimum length.
type bridging struct {
	// data are the bytes not read yet.
	data []byte
}

// value returns the value that the next bytes decode to under r's bounds.
func (b *bridging) value(r request, _ int) choice.Choice {
	if r.bounds.Kind() == choice.Integer {
		return choice.Choice{Kind: choice.Integer, Integer: b.integer(r.bounds.Integer())}
	}
	if r.bounds.Kind() == choice.Float {
		return choice.Choice{Kind: choice.Float, Float: b.float(r.bounds.Float())}
	}
	return choice.Choice{Kind: choice.Sequence, Sequence: b.sequence(r.bounds.Sequence())}
}

// integer returns the lower bound plus the offset the bytes state, or the
// target when they are too few.
func (b *bridging) integer(bounds choice.IntegerBounds) choice.Int {
	offset, ok := b.upTo(bounds.Above() + bounds.Below())
	if !ok {
		return bounds.Target()
	}
	return bounds.Lo().Add(offset)
}

// float returns the value whose bits the bytes state, fitted to the
// bounds, or the target.
func (b *bridging) float(bounds choice.FloatBounds) float64 {
	if bounds.Lo() == bounds.Hi() && bounds.Lo() != 0 && bounds.NaNPolicy() == choice.ExcludeNaN {
		return bounds.Target()
	}
	raw, ok := b.take(int(bounds.Width()) / byteBits)
	if !ok {
		return bounds.Target()
	}
	value := math.Float64frombits(raw)
	if bounds.Width() == choice.Width32 {
		value = float64(math.Float32frombits(uint32(raw)))
	}
	return bounds.Coerce(choice.Choice{Kind: choice.Float, Float: value})
}

// sequence returns the length the bytes state, then as many elements as
// they hold, extended with zeros to the minimum length.
func (b *bridging) sequence(bounds choice.SequenceBounds) []uint32 {
	sizes := bounds.Sizes()
	spread := uint64(unboundedLengths)
	if maxSize, bounded := sizes.Max(); bounded {
		spread = uint64(maxSize - sizes.Min())
	}
	extra, ok := b.upTo(spread)
	if !ok {
		return bounds.Target()
	}
	var elements []uint32
	for range sizes.Min() + int(extra) {
		element, read := b.upTo(uint64(bounds.K() - 1))
		if !read {
			break
		}
		elements = append(elements, uint32(element))
	}
	for len(elements) < sizes.Min() {
		elements = append(elements, 0)
	}
	return elements
}

// upTo returns an integer in [0, span] read from the bytes that span
// needs, and false when fewer remain.
func (b *bridging) upTo(span uint64) (uint64, bool) {
	u, ok := b.take((bits.Len64(span) + byteBits - 1) / byteBits)
	if !ok || span == math.MaxUint64 {
		return u, ok
	}
	return u % (span + 1), true
}

// take returns the next count bytes as a little-endian unsigned integer,
// and false when fewer remain, in which case the rest are spent.
func (b *bridging) take(count int) (uint64, bool) {
	if count > len(b.data) {
		b.data = nil
		return 0, false
	}
	var buf [8]byte
	copy(buf[:], b.data[:count])
	b.data = b.data[count:]
	return binary.LittleEndian.Uint64(buf[:]), true
}
