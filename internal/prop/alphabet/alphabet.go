// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package alphabet

import "strings"

// Size is the number of characters in the default alphabet: every Unicode
// scalar value.
const Size = 1112064

// printable are the 95 printable ASCII characters at indices 0 to 94, in
// their order.
const printable = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

// rest are the inclusive ranges of code points that follow the printable
// characters, ascending. They contain every scalar value that is not a
// printable ASCII character.
var rest = [...]struct{ first, last rune }{
	{first: 0x00, last: 0x1F},
	{first: 0x7F, last: 0xD7FF},
	{first: 0xE000, last: 0xFFFF},
	{first: 0x10000, last: 0x10FFFF},
}

// Rune returns the character at index i of the default alphabet. It panics
// for an i of Size or more.
func Rune(i uint32) rune {
	if i < uint32(len(printable)) {
		return rune(printable[i])
	}
	offset := i - uint32(len(printable))
	for _, g := range rest {
		if offset <= uint32(g.last-g.first) {
			return g.first + rune(offset)
		}
		offset -= uint32(g.last - g.first + 1)
	}
	panic("alphabet: an index past the end of the default alphabet")
}

// Index returns the index of r in the default alphabet. It reports false
// for a surrogate and for a value outside [0, 0x10FFFF], which are no
// Unicode scalar values.
func Index(r rune) (uint32, bool) {
	if i := strings.IndexRune(printable, r); i >= 0 {
		return uint32(i), true
	}
	offset := uint32(len(printable))
	for _, g := range rest {
		if g.first <= r && r <= g.last {
			return offset + uint32(r-g.first), true
		}
		offset += uint32(g.last - g.first + 1)
	}
	return 0, false
}

// AppendIndices appends the indices of the code points in [lo, hi] to dst
// and returns the extended slice. The appended intervals are the fewest,
// sorted by First, that neither overlap nor touch. The surrogates have no
// index and add nothing, and so does a range whose lo exceeds its hi.
func AppendIndices(dst []Interval, lo, hi rune) []Interval {
	start := len(dst)
	for i := range len(printable) {
		if r := rune(printable[i]); lo <= r && r <= hi {
			dst = append(dst, Interval{First: uint32(i), Last: uint32(i)})
		}
	}
	offset := uint32(len(printable))
	for _, g := range rest {
		if first, last := max(lo, g.first), min(hi, g.last); first <= last {
			dst = append(dst, Interval{First: offset + uint32(first-g.first), Last: offset + uint32(last-g.first)})
		}
		offset += uint32(g.last - g.first + 1)
	}
	return dst[:start+len(Merge(dst[start:]))]
}
