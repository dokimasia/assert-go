// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package choice defines the choices a property's case is made of: an
// integer, a float of a width, or a sequence of small integers, each
// within [Bounds].
//
// The target of bounds is the simplest value they admit. A [Key] orders
// the values of bounds from the simplest. The shrinker moves values
// towards their targets, and it keeps a candidate only when the keys of
// the candidate's choices are shortlex-smaller than the best case's.
//
// # Replay
//
// During a replay, each request of a body returns the recorded choice at
// the same position. [Bounds.Coerce] first fits that choice to the bounds
// of the request, which can differ from the bounds the case was recorded
// under. The fitted choice is a choice the bounds admit.
//
// # Integers
//
// An [Int] is any integer from -2^63 to 2^64 - 1, the union of the signed
// and the unsigned 64-bit ranges. [IntegerBounds] lie inside one of the
// two ranges. An [Int] keeps its sign whatever bounds requested it, so a
// replay coerces a recorded -5 to the target of unsigned bounds.
//
// # Floats
//
// A float choice has a [Width] of 32 or 64 bits and stores its value as a
// float64. A NaN in a choice has the bits [NaNBits].
//
// # Errors
//
// A constructor returns an error that wraps one of these, with the
// offending values in its text:
//
//   - [ErrEmpty] for bounds or sizes that admit no value.
//   - [ErrRange] for integer bounds inside neither 64-bit range.
//   - [ErrWidth] for a float width other than 32 or 64.
//   - [ErrNaNPolicy] for a NaN policy other than the two this package
//     declares.
//   - [ErrNotOfWidth] for a float bound that is not a value of its width.
//
// # Concurrency
//
// The methods of this package take value receivers and only read the
// slices a value shares, so any value is safe for concurrent use. A
// [Choice], a [Key] and the result of a sequence's coercion share the
// slice they were given, and a caller must not modify a shared slice.
//
// # Allocation contract
//
// The target of [SequenceBounds] with a minimum above zero allocates its
// zeros. A coercion of a sequence allocates the fitted sequence unless the
// recorded one already fits. [Int.String] allocates its text, and a
// constructor allocates the error it returns. Every other function and
// method allocates nothing.
//
// # Dependency position
//
// Imports the standard library only. Depends on no other package in this
// module.
package choice
