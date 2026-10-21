// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package wrap

// wrapped returns the seat that a seat wraps, and nil for a seat that wraps
// none. [Register] sets it while the root package initialises, which
// completes before any test starts, so its readers take no lock.
var wrapped func(seat any) any

// Register makes unwrap the function that returns the seat that a seat
// wraps, and nil for a seat that wraps none. The root package calls it from
// its init function. Call it only while a package initialises.
//
// # Panics
//
// It panics for a nil unwrap and for a second registration, and registers
// nothing then.
func Register(unwrap func(seat any) any) {
	if unwrap == nil {
		panic("wrap: Register registers no function")
	}
	if wrapped != nil {
		panic("wrap: Register registers a second function")
	}
	wrapped = unwrap
}

// Unwrap returns the seat that seat wraps, as the registered function
// returns it, and nil for a seat that wraps none.
//
// # Panics
//
// It panics before a registration. The root package registers its function
// while it initialises, and every program that calls Unwrap imports the root
// package.
//
// # Allocation contract
//
// Unwrap allocates nothing besides what the method Unwrap of seat allocates.
func Unwrap(seat any) any {
	return wrapped(seat)
}

// As returns the first seat of the chain of seat that is a T, and reports
// whether the chain has one. The chain is seat, then the seat that [Unwrap]
// returns for it, then the seat that Unwrap returns for that one, and so on,
// up to the first nil.
//
// # Allocation contract
//
// As allocates nothing besides what the methods Unwrap of the chain allocate.
func As[T any](seat any) (T, bool) {
	for ; seat != nil; seat = wrapped(seat) {
		if t, ok := seat.(T); ok {
			return t, true
		}
	}
	var zero T
	return zero, false
}
