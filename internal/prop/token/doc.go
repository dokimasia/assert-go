// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package token encodes the choices of one case as a replay token: one
// printable string that a caller passes back to run that case again.
//
// A token is [Prefix] followed by the unpadded base64url encoding of the
// choices in order. Each choice is a tag byte and then its payload:
//
//   - 0: a non-negative integer, in unsigned LEB128.
//   - 1: a negative integer, its magnitude in unsigned LEB128.
//   - 2: a float, its binary64 bits in eight bytes, little-endian, with
//     [choice.NaNBits] for every NaN.
//   - 3: a sequence, its length and then each element, each in unsigned
//     LEB128.
//
// A token records no bounds, because the generators state them again when
// the case replays. One choice sequence has exactly one token, and [Decode]
// accepts a token only in the form that [Append] writes. It checks that
// form on the bytes as it reads them, and reads a sequence element of 2^32
// or more, which a [choice.Choice] cannot store, as 2^32 - 1.
//
// # Errors
//
// [Decode] returns an error that wraps [ErrInvalid] for every token that
// no encoder writes, with the token and the fault in its text.
//
// # Concurrency
//
// Every function is safe for concurrent use. The package keeps no state.
//
// # Allocation contract
//
// [Append] grows the slice it is given, and allocates only when it lacks
// the capacity for the token and its binary payload. [Encode] allocates
// the token, and for a token past 128 bytes the buffer it writes in.
// [Decode] allocates the choices and the elements of their sequences. It
// reads a token's text, its payload and its first 16 choices in buffers on
// the stack, and allocates for a token past them.
//
// # Dependency position
//
// Imports the choice package of this module and the standard library.
package token
