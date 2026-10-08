// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package pattern parses the portable subset: the regular expressions that
// the engines of every target language read the same way. string-matching
// generates the strings of a pattern of the subset, and the matches
// assertion tests text against one.
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
//   - A group is (...) or (?:...), and | separates alternatives. Groups nest
//     at most 100 deep.
//   - The quantifiers are *, +, ?, {m}, {m,} and {m,n}, with counts of at
//     most 1,000. A count of two or more digits does not start with 0, which
//     RE2 reads as literal text. No quantifier follows another.
//   - Along every chain of nested quantifiers, the counts multiply to at
//     most 1,000. A count is the upper one, or the lower one when there is
//     no upper one, and a count of 0 counts as 1.
//   - ^ may be the first character of the pattern, and $ its last.
//
// [Parse] returns the pieces of a pattern: a [Literal], a [Sequence], an
// [Alternation], a [Repeat] or a [Class]. A class states its members as
// indices of the default alphabet, so a reader decodes the simplest member
// first.
//
// # Errors
//
// [Parse] returns an error that wraps [ErrOutside] for a pattern outside
// the portable subset, or that is not UTF-8. Its text states the pattern,
// the position in characters and the construct at fault.
//
// # Concurrency
//
// A parsed pattern is never changed after Parse returns it, so any number
// of goroutines may read one at once.
//
// # Allocation contract
//
// [Parse] allocates the pattern's characters and its pieces.
//
// # Dependency position
//
// Imports the alphabet and choice packages of this module and the standard
// library, so the assertion core can read the subset without the property
// engine.
package pattern
