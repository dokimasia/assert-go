// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random

import (
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// The odds and the sizes of an integer draw.
const (
	// edgeOdds is the denominator of the coin that sends an integer draw
	// to an edge value: one draw in 8.
	edgeOdds = 8
	// directionOdds is the denominator of the coin that sends an integer
	// draw upward when its target lies strictly inside the bounds.
	directionOdds = 2
	// maxIntegerEdges is the number of candidate edge values of integer
	// bounds.
	maxIntegerEdges = 5
)

// offsetLimits are the largest offsets from the target that an integer
// draw allows, one chosen with even odds: 2^4 - 1, 2^8 - 1, 2^16 - 1 and
// 2^64 - 1. Small offsets are common, and every value of the bounds is
// reachable.
var offsetLimits = [...]uint64{0xF, 0xFF, 0xFFFF, 0xFFFFFFFFFFFFFFFF}

// Integer draws an integer inside b.
//
// Bounds of one value return it and consume nothing. Otherwise a
// Coin(1, 8) decides whether the draw takes an edge value, chosen with
// Below(the number of edges). Every other draw moves away from the target:
// upward when the target is the lower bound, downward when it is the upper
// bound, and by a Coin(1, 2) when it lies strictly inside. The offset is
// UpTo(min(span, limit)), where span is the number of values in that
// direction and limit is one of offsetLimits, chosen with Below(4).
func Integer(s *Source, b choice.IntegerBounds) choice.Int {
	if b.Lo() == b.Hi() {
		return b.Lo()
	}
	if s.Coin(1, edgeOdds) {
		var edges [maxIntegerEdges]choice.Int
		values := AppendIntegerEdges(edges[:0], b)
		return values[s.Below(uint64(len(values)))]
	}
	target := b.Target()
	if b.Below() == 0 || b.Above() != 0 && s.Coin(1, directionOdds) {
		return target.Add(offset(s, b.Above()))
	}
	return target.Sub(offset(s, b.Below()))
}

// AppendIntegerEdges appends the edge values of b to dst and returns the
// extended slice. The edge values are the target, the lower bound, the
// upper bound, the value above the target and the value below it, in that
// order, each once, and only those that b admits.
func AppendIntegerEdges(dst []choice.Int, b choice.IntegerBounds) []choice.Int {
	target := b.Target()
	above, below := target, target
	if b.Above() > 0 {
		above = target.Add(1)
	}
	if b.Below() > 0 {
		below = target.Sub(1)
	}
	start := len(dst)
	for _, v := range [maxIntegerEdges]choice.Int{target, b.Lo(), b.Hi(), above, below} {
		if !slices.Contains(dst[start:], v) {
			dst = append(dst, v)
		}
	}
	return dst
}

// offset draws the distance from the target of an integer draw that has
// span values in its direction: Below(4) chooses a limit of offsetLimits,
// and UpTo(min(span, limit)) the distance.
func offset(s *Source, span uint64) uint64 {
	limit := offsetLimits[s.Below(uint64(len(offsetLimits)))]
	return s.UpTo(min(span, limit))
}
