// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random

import "math/bits"

// The constants of the generator.
const (
	// firstWord is the value that the initialisation puts in the first
	// word of the state.
	firstWord = 0xF1EA5EED
	// warmupRounds is the number of values that the initialisation
	// discards, so that nearby seeds give unrelated streams.
	warmupRounds = 20
	// rotateB, rotateC and rotateD are the rotations of the three-rotate
	// variant for 64-bit words.
	rotateB = 7
	rotateC = 13
	rotateD = 37
	// wordBits is the number of bits of every value of the stream.
	wordBits = 64
)

// Source is the stream of 64-bit values that one seed produces.
//
// The zero value is the stream of the all-zero state, every value of which
// is 0. [New] returns the stream of a seed.
//
// # Concurrency
//
// A Source is not safe for concurrent use. Every method advances its
// state.
//
// # Allocation contract
//
// No method allocates. A Source is a value of four words, so a case keeps
// one without an allocation.
type Source struct {
	// a, b, c and d are the four words of the state.
	a, b, c, d uint64
}

// New returns the stream of seed. The state starts with 0xF1EA5EED in its
// first word and seed in the other three, and the generator discards its
// first 20 values.
func New(seed uint64) Source {
	s := Source{a: firstWord, b: seed, c: seed, d: seed}
	for range warmupRounds {
		s.Next()
	}
	return s
}

// ForCase returns the stream of case index of a run with seed: the stream
// of seed + index, modulo 2^64.
func ForCase(seed, index uint64) Source {
	return New(seed + index)
}

// Next returns the next value of the stream.
func (s *Source) Next() uint64 {
	e := s.a - bits.RotateLeft64(s.b, rotateB)
	s.a = s.b ^ bits.RotateLeft64(s.c, rotateC)
	s.b = s.c + bits.RotateLeft64(s.d, rotateD)
	s.c = s.d + e
	s.d = e + s.a
	return s.d
}

// UpTo returns a uniform value in [0, hi]. For hi of 0 it returns 0 and
// consumes nothing. Otherwise every attempt consumes one value and keeps
// its top bits.Len64(hi) bits, and the draw repeats while the result
// exceeds hi. An attempt succeeds with a probability above 1/2.
func (s *Source) UpTo(hi uint64) uint64 {
	if hi == 0 {
		return 0
	}
	shift := wordBits - bits.Len64(hi)
	for {
		if v := s.Next() >> shift; v <= hi {
			return v
		}
	}
}

// Below returns a uniform value in [0, n), as UpTo(n - 1) does. It panics
// for n of 0.
func (s *Source) Below(n uint64) uint64 {
	if n == 0 {
		panic("random: Below(0) has no value to return")
	}
	return s.UpTo(n - 1)
}

// Coin reports true with probability num/den, from one Below(den): true
// when the draw is below num. It panics for den of 0 or a num above den.
func (s *Source) Coin(num, den uint64) bool {
	if num > den {
		panic("random: a coin's numerator exceeds its denominator")
	}
	return s.Below(den) < num
}
