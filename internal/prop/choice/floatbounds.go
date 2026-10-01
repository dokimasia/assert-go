// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import (
	"errors"
	"fmt"
	"math"
)

// ErrWidth reports a float width other than [Width32] and [Width64].
var ErrWidth = errors.New("choice: a float width is 32 or 64")

// ErrNaNPolicy reports a NaN policy other than [ExcludeNaN] and
// [AdmitNaN].
var ErrNaNPolicy = errors.New("choice: a NaN policy excludes or admits NaN")

// ErrNotOfWidth reports a float bound that is not a value of its width.
var ErrNotOfWidth = errors.New("choice: a float bound is not a value of its width")

// FloatBounds are the bounds of a float choice: an inclusive range of
// values of one width, and whether NaN is a value.
//
// An infinite bound admits its infinity. The zero value is the binary64
// bounds [0, 0] without NaN. FloatBounds are comparable, and == reports
// whether two bounds are equal, with the bounds compared as numbers.
type FloatBounds struct {
	// lo and hi are the inclusive bounds, both values of the width.
	lo, hi float64
	// target is the value of the bounds with the smallest key, computed
	// once by the constructor.
	target float64
	// nan states whether NaN is a value.
	nan NaNPolicy
	// width is the width of every value.
	width Width
}

// NewFloatBounds returns the bounds [lo, hi] of width w, with NaN a value
// when nan is [AdmitNaN].
//
// It returns [ErrWidth] when w is not a [Width], [ErrNaNPolicy] when nan is
// not a [NaNPolicy], [ErrEmpty] when a bound is NaN or lo exceeds hi, and
// [ErrNotOfWidth] when a bound is not a value of width w.
func NewFloatBounds(lo, hi float64, nan NaNPolicy, w Width) (FloatBounds, error) {
	if !w.Valid() {
		return FloatBounds{}, fmt.Errorf("%w: %d", ErrWidth, uint8(w))
	}
	if !nan.Valid() {
		return FloatBounds{}, fmt.Errorf("%w: %d", ErrNaNPolicy, uint8(nan))
	}
	if math.IsNaN(lo) || math.IsNaN(hi) || lo > hi {
		return FloatBounds{}, fmt.Errorf("%w: float bounds [%v, %v]", ErrEmpty, lo, hi)
	}
	if !Representable(lo, w) || !Representable(hi, w) {
		return FloatBounds{}, fmt.Errorf("%w: [%v, %v] at width %s", ErrNotOfWidth, lo, hi, w)
	}
	return FloatBounds{lo: lo, hi: hi, target: simplest(lo, hi), nan: nan, width: w}, nil
}

// Lo returns the inclusive lower bound, a value of the width. Every value
// the bounds admit, NaN aside, is at least Lo.
func (b FloatBounds) Lo() float64 {
	return b.lo
}

// Hi returns the inclusive upper bound, a value of the width. Every value
// the bounds admit, NaN aside, is at most Hi.
func (b FloatBounds) Hi() float64 {
	return b.hi
}

// NaNPolicy returns whether NaN is a value of the bounds.
func (b FloatBounds) NaNPolicy() NaNPolicy {
	return b.nan
}

// Width returns the width of every value the bounds admit.
func (b FloatBounds) Width() Width {
	return b.width
}

// Target returns the value of the bounds with the smallest [FloatKey]. It
// is never NaN.
func (b FloatBounds) Target() float64 {
	return b.target
}

// Admits reports whether x is a value of the bounds: NaN under
// [AdmitNaN], and otherwise a value of the width inside the range.
func (b FloatBounds) Admits(x float64) bool {
	if math.IsNaN(x) {
		return b.nan == AdmitNaN
	}
	return b.lo <= x && x <= b.hi && Representable(x, b.width)
}

// Coerce returns the recorded value when c is a float the bounds admit,
// with any NaN as [NaN], and the target otherwise.
func (b FloatBounds) Coerce(c Choice) float64 {
	if c.Kind != Float || !b.Admits(c.Float) {
		return b.target
	}
	if math.IsNaN(c.Float) {
		return NaN()
	}
	return c.Float
}

// Integers returns the bounds of the integers that b admits and whose
// magnitude is below 2^24 at width 32 or 2^53 at width 64, each a value of
// the width. It reports false when b admits no such integer.
func (b FloatBounds) Integers() (IntegerBounds, bool) {
	limit := float64(b.width.integerLimit() - 1)
	lo := max(math.Ceil(b.lo), -limit)
	hi := min(math.Floor(b.hi), limit)
	if lo > hi {
		return IntegerBounds{}, false
	}
	return IntegerBounds{lo: IntOf(int64(lo)), hi: IntOf(int64(hi))}, true
}

// simplest returns the value in [lo, hi] with the smallest [FloatKey]. lo
// is at most hi, and both are values of one width.
//
// Every value the search tries has at most as many significant bits as the
// bound it starts from, so it is a value of that width too. The result
// depends on the range alone.
func simplest(lo, hi float64) float64 {
	if lo > 0 {
		return simplestPositive(lo, hi)
	}
	if hi < 0 {
		return -simplestPositive(-hi, -lo)
	}
	return 0
}

// simplestPositive returns the simplest value in [lo, hi], for
// 0 < lo <= hi.
//
// The smallest integer in the range is the simplest value when the range
// contains an integer. Otherwise the search tries each number of
// fractional bits f, from the fewest up to the fractional bits of lo,
// where lo itself is the simplest value. The candidate at f is n / 2^f for
// the smallest odd n with n / 2^f at least lo, and the search returns the
// first candidate at most hi. At f of 0 the candidate is an integer above
// hi. Every numerator is at most the numerator of lo, so the search is
// exact in float64.
func simplestPositive(lo, hi float64) float64 {
	if integral := math.Ceil(lo); integral <= hi {
		return integral
	}
	for fraction := range int(FractionBits(lo)) {
		numerator := math.Ceil(math.Ldexp(lo, fraction))
		if math.Mod(numerator, 2) == 0 {
			numerator++
		}
		if candidate := math.Ldexp(numerator, -fraction); candidate <= hi {
			return candidate
		}
	}
	return lo
}
