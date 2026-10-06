// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"hash/maphash"
	"math"
	"reflect"

	"go.dokimi.dev/assert/internal/equality"
)

// stringSeed is the seed of the hash of a state of type string. Each process
// makes its own seed. A hash decides where the memo looks, and no output of a
// check.
var stringSeed = maphash.MakeSeed()

// operations are the functions of a spec that a search calls, with the
// defaults of Equal and Hash. equal and hash take pointers to states that
// are on the heap already, so a comparison copies no state there.
type operations[S any] struct {
	// initial returns the state before any call.
	initial func() S
	// next returns the states that a call may leave from a state.
	next func(state S, op Operation) []S
	// equal reports whether two states are interchangeable.
	equal func(a, b *S) bool
	// hash returns the hash of a state, and is nil for a spec that states
	// Equal alone, whose states all hash alike.
	hash func(state *S) uint64
	// cost returns the steps of the budget that a step from a state counts
	// for, and is nil for a spec whose every step counts for one.
	cost func(state S) int
}

// operations returns the functions that a search calls for s. A nil Equal
// compares states as assert.Equal compares them, and a nil Hash beside it
// hashes them alike: directly for a state of a basic type, such as int or
// string, and through reflection for a state of any other type. A spec that
// states Equal alone has no hash.
func (s Spec[S]) operations() operations[S] {
	ops := operations[S]{initial: s.Initial, next: s.Next, cost: s.cost}
	if s.Equal == nil {
		ops.equal, ops.hash = standard[S]()
	} else {
		equal := s.Equal
		ops.equal = func(a, b *S) bool { return equal(*a, *b) }
	}
	if s.Hash != nil {
		hash := s.Hash
		ops.hash = func(state *S) uint64 { return hash(*state) }
	}
	return ops
}

// standard returns the comparison and the hash of states of type S as
// assert.Equal compares them: with Go's == for a basic type, whose == agrees
// with assert.Equal, and through reflection for any other type.
func standard[S any]() (func(a, b *S) bool, func(state *S) uint64) {
	switch any(*new(S)).(type) {
	case bool:
		return scalar[S](boolHash)
	case int:
		return scalar[S](integerHash[int])
	case int8:
		return scalar[S](integerHash[int8])
	case int16:
		return scalar[S](integerHash[int16])
	case int32:
		return scalar[S](integerHash[int32])
	case int64:
		return scalar[S](integerHash[int64])
	case uint:
		return scalar[S](integerHash[uint])
	case uint8:
		return scalar[S](integerHash[uint8])
	case uint16:
		return scalar[S](integerHash[uint16])
	case uint32:
		return scalar[S](integerHash[uint32])
	case uint64:
		return scalar[S](integerHash[uint64])
	case uintptr:
		return scalar[S](integerHash[uintptr])
	case float32:
		return scalar[S](floatHash[float32])
	case float64:
		return scalar[S](floatHash[float64])
	case string:
		return scalar[S](stringHash)
	}
	return reflected[S], hashReflected[S]
}

// scalar returns the comparison of states of the basic type T, which S is,
// with Go's ==, and their hash with hash. Go's == on a basic type agrees with
// assert.Equal: -0 equals +0, and a NaN equals nothing.
func scalar[S any, T comparable](hash func(T) uint64) (func(a, b *S) bool, func(state *S) uint64) {
	equal := func(a, b *T) bool { return *a == *b }
	hashOf := func(state *T) uint64 { return hash(*state) }
	return any(equal).(func(a, b *S) bool), any(hashOf).(func(state *S) uint64)
}

// reflected reports whether the states at a and b are equal as assert.Equal
// compares them, through reflection.
func reflected[S any](a, b *S) bool {
	return equality.Equal(reflect.ValueOf(a).Elem(), reflect.ValueOf(b).Elem(), equality.Rules{})
}

// hashReflected returns the hash of the state at state that every state equal
// to it under [reflected] shares.
func hashReflected[S any](state *S) uint64 {
	return equality.Hash(reflect.ValueOf(state).Elem())
}

// integer is the set of the integer types.
type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// integerHash returns the hash of an integer.
func integerHash[T integer](v T) uint64 {
	return mix(uint64(v))
}

// floatHash returns the hash of a float. Adding 0 turns -0 into +0, so the
// two zeros, which Go's == reports equal, hash alike.
func floatHash[T float32 | float64](v T) uint64 {
	return mix(math.Float64bits(float64(v) + 0))
}

// boolHash returns the hash of a bool.
func boolHash(v bool) uint64 {
	if v {
		return mix(1)
	}
	return mix(0)
}

// stringHash returns the hash of a string.
func stringHash(v string) uint64 {
	return maphash.String(stringSeed, v)
}

// mix returns a hash of x whose every bit depends on every bit of x: the
// final mix of MurmurHash3.
func mix(x uint64) uint64 {
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return x
}
