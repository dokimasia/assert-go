// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package random draws the values of generated cases from a seeded stream:
// the 64-bit three-rotate variant of Bob Jenkins's small noncryptographic
// generator.
//
// The stream uses addition, subtraction, exclusive or and rotation modulo
// 2^64 and nothing else, so every implementation of the definition
// computes the same values from the same seed. Every draw consumes the
// stream in a fixed order, including the draws it skips: equal bounds, a
// forced continue flag and a range of one value consume nothing. Two
// implementations that consume the stream differently produce different
// cases from one seed.
//
// # Draws
//
//   - [Integer], [Float] and [AppendSequence] draw a value inside bounds of
//     the choice package.
//   - [Flag] decides a collection's length one element at a time, around
//     the length that [Average] returns.
//   - [Reuse] decides whether a reusable draw repeats an earlier value of
//     its case.
//
// # Seeds
//
// Case i of a run with seed s reads the stream [ForCase](s, i). [Mix] folds
// bytes into a seed with the stream alone.
//
// # Panics
//
// [Source.Below] panics for an empty range, and [Source.Coin] for odds
// outside [0, 1], as math/rand/v2 panics for an empty range. The draws
// pass neither.
//
// # Concurrency
//
// A [Source] is not safe for concurrent use. The functions of this package
// keep no state of their own, so calls on distinct sources run
// concurrently.
//
// # Allocation contract
//
// No function or method allocates. [AppendSequence], [AppendIntegerEdges]
// and [AppendFloatEdges] grow the slice they are given, and allocate only
// when it lacks the capacity.
//
// # Dependency position
//
// Imports the choice package of this module and the standard library.
package random
