// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// The ids of the number generators.
const (
	// integerID is the id of an integer.
	integerID = "integer"
	// durationID is the id of a duration.
	durationID = "duration"
	// floatID is the id of a float.
	floatID = "float"
	// booleanID is the id of a boolean.
	booleanID = "boolean"
)

// Integral is every integer type, and the types defined over them.
type Integral interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Floating is float32 and float64, and the types defined over them.
type Floating interface {
	~float32 | ~float64
}

// Integer returns a generator of the integers in [lo, hi]: one integer
// choice, which the random phase may give an earlier value of its case
// with the same bounds. Its simplest value is the one closest to zero. It
// panics when lo exceeds hi.
func Integer[T Integral](lo, hi T) Generator[T] {
	return integerOf(integerID, lo, hi)
}

// Duration returns a generator of the durations in [lo, hi], decoded as an
// integer of nanoseconds is. It panics when lo exceeds hi.
func Duration(lo, hi time.Duration) Generator[time.Duration] {
	return integerOf(durationID, lo, hi)
}

// integerOf returns the integer generator with id over [lo, hi].
func integerOf[T Integral](id string, lo, hi T) Generator[T] {
	if hi < lo {
		panic(fmt.Sprintf("prop: %s(%v, %v) states no value", id, lo, hi))
	}
	bounds := choice.MustIntegerBounds(intOf(lo), intOf(hi))
	g := newGenerator(id, func(c *Case) T {
		span := c.openSpan(id)
		defer c.closeSpan(span)
		return valueOf[T](c.reusable(bounds))
	})
	g.erased.integer, g.erased.bounds = true, bounds
	g.erased.value = func(i choice.Int) any { return valueOf[T](i) }
	return g
}

// Float returns a generator of the floats in [lo, hi] of T's width, with
// NaN among them under [choice.AdmitNaN]: one float choice. Its simplest
// value has the fewest fractional bits, then the smallest magnitude. It
// panics when a bound is NaN or lo exceeds hi.
func Float[T Floating](lo, hi T, nan choice.NaNPolicy) Generator[T] {
	width := choice.Width64
	if reflect.TypeFor[T]().Kind() == reflect.Float32 {
		width = choice.Width32
	}
	bounds, err := choice.NewFloatBounds(float64(lo), float64(hi), nan, width)
	if err != nil {
		panic(fmt.Sprintf("prop: %s(%v, %v) states no value: %v", floatID, lo, hi, err))
	}
	return newGenerator(floatID, func(c *Case) T {
		span := c.openSpan(floatID)
		defer c.closeSpan(span)
		return T(c.float(bounds))
	})
}

// Boolean returns a generator that returns true with probability num/den:
// one integer choice in [0, 1], drawn by one coin of the odds in lowest
// terms. Its simplest value is false. It panics when den is 0 or num
// exceeds den.
func Boolean(num, den uint64) Generator[bool] {
	if den == 0 || num > den {
		panic(fmt.Sprintf("prop: %s odds %d/%d are no probability", booleanID, num, den))
	}
	divisor := gcd(num, den)
	num, den = num/divisor, den/divisor
	return newGenerator(booleanID, func(c *Case) bool {
		span := c.openSpan(booleanID)
		defer c.closeSpan(span)
		return c.coin(num, den)
	})
}

// signedKinds are the kinds of the signed integer types.
var signedKinds = []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64}

// intOf returns v as a choice value: through int64 for a signed type, and
// through uint64 for an unsigned one.
func intOf[T Integral](v T) choice.Int {
	if slices.Contains(signedKinds, reflect.TypeFor[T]().Kind()) {
		return choice.IntOf(int64(v))
	}
	return choice.UintOf(uint64(v))
}

// valueOf returns a choice value inside the bounds of T as a T.
func valueOf[T Integral](i choice.Int) T {
	if i.Negative() {
		return T(int64(-i.Magnitude()))
	}
	return T(i.Magnitude())
}

// gcd returns the greatest common divisor of a and b, for a b above 0.
func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
