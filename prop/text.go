// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/matching"
)

// stringing is what the options of one [String] state.
type stringing struct {
	// lengths are the bounds on the string's length in characters.
	lengths lengths
	// alphabet are the characters, the simplest first, when stated is set.
	alphabet string
	// stated reports whether the options state an alphabet.
	stated bool
}

// StringOption configures [String]: a [SizeOption], or [Alphabet]. A later
// option overrides an earlier one of the same kind.
type StringOption interface {
	// applyString applies the option to the options of a string.
	applyString(s *stringing)
}

// Alphabet states the characters that [String] chooses from, the simplest
// first. [String] panics when chars is empty, is not UTF-8, or repeats a
// character.
func Alphabet(chars string) StringOption {
	return alphabet(chars)
}

// alphabet is the option that [Alphabet] returns: the characters, the
// simplest first.
type alphabet string

// applyString makes the string choose from the characters of a.
func (a alphabet) applyString(s *stringing) {
	s.alphabet, s.stated = string(a), true
}

// String returns a generator of strings of the characters of the default
// alphabet, or of the one that [Alphabet] states, with lengths in characters
// that the options admit: one sequence of indices into the alphabet. Its
// simplest value is the shortest string of the alphabet's first character.
//
// The default alphabet starts with the characters a reader can type: the
// digits, the lowercase and the uppercase letters, space and the printable
// ASCII punctuation, then every other code point of the Basic Multilingual
// Plane except the surrogates, then the code points above it. A string
// shrinks towards "0", "00" and on. It panics when the options state no
// length or no alphabet.
func String(opts ...StringOption) Generator[string] {
	var s stringing
	for _, o := range opts {
		o.applyString(&s)
	}
	if s.stated {
		return Generator[string](engine.StringOver(s.alphabet, s.lengths.sizes()))
	}
	return Generator[string](engine.String(s.lengths.sizes()))
}

// Bytes returns a generator of byte strings with lengths that the options
// admit: one sequence of elements in [0, 255]. Its simplest value is the
// shortest string of zero bytes. A fuzzer's bytes decode into such a
// sequence one to one. It panics when the options state no length.
func Bytes(opts ...SizeOption) Generator[[]byte] {
	return Generator[[]byte](engine.Bytes(sizesOf(opts)))
}

// StringMatching returns a generator of the strings that expr matches in
// full, for an expr of the portable subset that the regular expression
// engines of every target language read the same way:
//
//   - A literal is any character but \ . ^ $ | ? * + ( ) [ ] { }, or one of
//     those after a backslash.
//   - . is any character but \n, \r, U+0085, U+2028 and U+2029.
//   - \d is the ASCII digits, \w the ASCII letters, the digits and the
//     underscore, and \s space, \t, \n, \f and \r.
//   - A class [...] contains characters, ranges and those three escapes,
//     and a leading ^ negates it. Inside a class, \, [, ] and a hyphen that
//     forms no range take a backslash, and a hyphen may also come first or
//     last. A class contains no &&, --, || or ~~.
//   - A group is (...) or (?:...), and | separates alternatives.
//   - The quantifiers are *, +, ?, {m}, {m,} and {m,n}, with counts of at
//     most 1,000. A count of two or more digits does not start with 0. No
//     quantifier follows another.
//   - ^ may be the first character of expr, and $ its last.
//
// Its simplest value takes the first branch of each alternation, the fewest
// repetitions, and the simplest character of each class in the order of
// the default alphabet, so [A0a] shrinks to "0". It panics for an expr
// outside the subset, naming the position and the construct at fault.
func StringMatching(expr string) Generator[string] {
	g, err := matching.StringMatching(expr)
	if err != nil {
		panic("prop: string-matching: " + err.Error())
	}
	return Generator[string](g)
}
