// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern

import "go.dokimi.dev/assert/internal/prop/alphabet"

// The members of the sets that the subset names, which the class of the
// same name contains in every engine.
const (
	// lineTerminators are the characters that . leaves out, because some
	// engine's . matches none of them.
	lineTerminators = "\n\r\u0085  "
	// digits are the members of \d.
	digits = "0123456789"
	// word are the members of \w.
	word = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	// space are the members of \s.
	space = " \t\n\f\r"
)

// past is the interval just past the last index of the default alphabet,
// which closes the last gap of a complement.
var past = alphabet.Interval{First: alphabet.Size, Last: alphabet.Size}

// Class is one character of a set: a class [...], ., \d, \w or \s.
type Class struct {
	// Members are the indices of the set's characters in the default
	// alphabet, sorted, neither overlapping nor touching.
	Members []alphabet.Interval
	// Count is the number of members, at least 1.
	Count uint64
}

// The classes that the subset names.
var (
	// dot is the class of ., every character but the line terminators.
	dot, _ = newClass(complement(membersOf(lineTerminators)))
	// digitClass is the class of \d.
	digitClass, _ = newClass(membersOf(digits))
	// wordClass is the class of \w.
	wordClass, _ = newClass(membersOf(word))
	// spaceClass is the class of \s.
	spaceClass, _ = newClass(membersOf(space))
)

// newClass returns the class of the indices that intervals contain, which
// it sorts and merges in place, and false when they contain none.
func newClass(intervals []alphabet.Interval) (Class, bool) {
	merged := alphabet.Merge(intervals)
	members := uint64(0)
	for _, v := range merged {
		members += uint64(v.Last-v.First) + 1
	}
	if members == 0 {
		return Class{}, false
	}
	return Class{Members: merged, Count: members}, true
}

// shorthand returns the class that \d, \w or \s names for the letter after
// the backslash, and false for any other letter.
func shorthand(r rune) (Class, bool) {
	if r == 'd' {
		return digitClass, true
	}
	if r == 'w' {
		return wordClass, true
	}
	if r == 's' {
		return spaceClass, true
	}
	return Class{}, false
}

// membersOf returns the indices of the characters of chars.
func membersOf(chars string) []alphabet.Interval {
	var members []alphabet.Interval
	for _, r := range chars {
		members = alphabet.AppendIndices(members, r, r)
	}
	return members
}

// complement returns the indices of the default alphabet that intervals do
// not contain, sorted. It sorts and overwrites intervals.
func complement(intervals []alphabet.Interval) []alphabet.Interval {
	var gaps []alphabet.Interval
	start := uint32(0)
	for _, v := range append(alphabet.Merge(intervals), past) {
		if v.First > start {
			gaps = append(gaps, alphabet.Interval{First: start, Last: v.First - 1})
		}
		start = v.Last + 1
	}
	return gaps
}
