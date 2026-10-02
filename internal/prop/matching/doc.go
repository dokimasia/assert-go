// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package matching builds string-matching: the generator of the strings
// that a regular expression of the portable subset matches in full. The
// pattern package parses the subset, and this package decodes a parsed
// pattern from a case.
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
// [StringMatching] returns the error of [pattern.Parse], which wraps
// [pattern.ErrOutside], for a pattern outside the portable subset or that
// is not UTF-8.
//
// # Concurrency
//
// A generator that [StringMatching] returns changes no state when it
// decodes, so one generator serves any number of cases at once.
//
// # Allocation contract
//
// [StringMatching] allocates the parsed pattern and the decoder built from
// it. A decode allocates the string it returns, and the case records its
// choices and spans.
//
// # Dependency position
//
// Imports the engine, pattern, alphabet and choice packages of this module
// and the standard library.
package matching
