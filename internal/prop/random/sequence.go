// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random

import "go.dokimi.dev/assert/internal/prop/choice"

// AppendSequence draws a sequence inside b, appends it to dst and returns
// the extended slice.
//
// Each position first takes the decision that [Flag] makes for a
// collection of b's sizes around their [Average], and when it continues, an
// element that [Integer] draws over b's element bounds. A byte string
// consumes the stream exactly as a list of integers in [0, 255] with the
// same sizes does, and gets the same values.
func AppendSequence(dst []uint32, s *Source, b choice.SequenceBounds) []uint32 {
	sizes, element := b.Sizes(), b.Element()
	average, start := Average(sizes), len(dst)
	for Flag(s, sizes, len(dst)-start, average) {
		dst = append(dst, uint32(Integer(s, element).Magnitude()))
	}
	return dst
}
