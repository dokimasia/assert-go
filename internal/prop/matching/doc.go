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
// # Running backwards
//
// A generator runs backwards from a string that its pattern matches in
// full, through the match that a backtracking engine finds first. The
// search tries the branches of an alternation in their stated order, and
// repeats each quantified piece as often as the rest of the pattern
// allows. A repetition that matches the empty string does not repeat
// beyond the quantifier's minimum. The search takes time exponential in
// the length of the string in the worst case, as a backtracking engine's
// does.
//
// # Errors
//
// [StringMatching] returns the fault of [pattern.Parse], of the kind
// [pattern.ErrOutside], for a pattern outside the portable subset or that
// is not UTF-8. A generator's inverse returns a fault of the kind
// [engine.ErrCannotInvert] for a value that its pattern does not match.
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
// choices and spans. The search of an inverse allocates the steps of each
// match it tries.
//
// # Dependency position
//
// Imports the fault, engine, pattern, alphabet and choice packages of this
// module and the standard library.
package matching
