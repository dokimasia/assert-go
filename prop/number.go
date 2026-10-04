// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The odds of true of a [Boolean] when [Odds] states none: 1/2.
const (
	// evenNumerator is the numerator of the odds.
	evenNumerator = 1
	// evenDenominator is the denominator of the odds.
	evenDenominator = 2
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

// FloatOption configures [Float]. The zero FloatOption changes nothing.
type FloatOption struct {
	// allowNaN makes NaN a value of the generator.
	allowNaN bool
}

// AllowNaN makes [Float] return NaN among its values, as the canonical
// quiet NaN. NaN is the simplest value after the infinities.
func AllowNaN() FloatOption {
	return FloatOption{allowNaN: true}
}

// BooleanOption configures [Boolean]. The zero BooleanOption changes
// nothing.
type BooleanOption struct {
	// num and den are the odds of true, and den is 0 for the zero option.
	num, den uint64
}

// Odds makes [Boolean] return true with probability num/den during the
// random phase. A replayed or shrunk case is unaffected. It panics when den
// is 0 or num exceeds den.
func Odds(num, den uint64) BooleanOption {
	if den == 0 || num > den {
		panic(fmt.Sprintf("prop: Odds(%d, %d) states no probability", num, den))
	}
	return BooleanOption{num: num, den: den}
}

// Integer returns a generator of the integers of T in [lo, hi]. Its
// simplest value is the one closest to zero, and above zero at equal
// distance. The random phase draws an earlier value of the case with the
// same bounds in one case in four, so a body sees two equal values often.
// Integer[uint64] covers the whole unsigned range. It panics when lo
// exceeds hi.
func Integer[T Integral](lo, hi T) Generator[T] {
	return Generator[T](engine.Integer(lo, hi))
}

// Duration returns a generator of the durations in [lo, hi], decoded as an
// integer of nanoseconds is. It panics when lo exceeds hi.
func Duration(lo, hi time.Duration) Generator[time.Duration] {
	return Generator[time.Duration](engine.Duration(lo, hi))
}

// Float returns a generator of the floats of T's width in [lo, hi], which
// may be infinite. A float32 draws choices of width 32. Its simplest value
// has the fewest fractional bits, then the smallest magnitude, and is
// positive at equal magnitude. It panics when a bound is NaN, lo exceeds
// hi, or a bound is not a value of T's width.
func Float[T Floating](lo, hi T, opts ...FloatOption) Generator[T] {
	nan := choice.ExcludeNaN
	for _, o := range opts {
		if o.allowNaN {
			nan = choice.AdmitNaN
		}
	}
	return Generator[T](engine.Float(lo, hi, nan))
}

// Boolean returns a generator of true and false, true with probability
// 1/2 during the random phase unless [Odds] states other odds. Its simplest
// value is false.
func Boolean(opts ...BooleanOption) Generator[bool] {
	num, den := uint64(evenNumerator), uint64(evenDenominator)
	for _, o := range opts {
		if o.den != 0 {
			num, den = o.num, o.den
		}
	}
	return Generator[bool](engine.Boolean(num, den))
}
