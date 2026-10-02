// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package pattern builds string-matching: the generator of the strings that
// a regular expression of the portable subset matches in full.
//
// The portable subset is what the regular expression engines of every
// target language read the same way:
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
//     most 1,000. A count of two or more digits does not start with 0, which
//     RE2 reads as literal text. No quantifier follows another.
//   - ^ may be the first character of the pattern, and $ its last.
//
// # Decoding
//
// A pattern decodes from a case in a fixed way, inside a span labelled
// string-matching:
//
//   - An alternation of two or more branches chooses one with an index that
//     decides structure, in a span labelled alternation.
//   - A quantifier decodes its repetitions as a collection does, in a span
//     labelled repeat, each repetition in a span labelled element.
//   - A class, ., \d, \w and \s choose a character with an index over their
//     members, which are in the order of the default alphabet.
//
// The targets give the first branch of each alternation, the fewest
// repetitions, and the simplest character of each class, so [A0a]
// shrinks to "0".
//
// # Errors
//
// [StringMatching] returns an error that wraps [ErrOutside] for a pattern
// outside the portable subset, or that is not UTF-8. Its text states the
// pattern, the position in characters and the construct at fault.
//
// # Concurrency
//
// A generator that [StringMatching] returns changes no state when it
// decodes, so one generator serves any number of cases at once.
//
// # Allocation contract
//
// [StringMatching] allocates the parsed pattern. A decode allocates the
// string it returns, and the case records its choices and spans.
//
// # Dependency position
//
// Imports the engine, alphabet and choice packages of this module and the
// standard library.
package pattern
