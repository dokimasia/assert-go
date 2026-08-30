// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"math"
	"reflect"
)

// CloseTo reports when got is further than tolerance from want,
// comparing by absolute difference: abs(got-want) <= tolerance passes.
//
// got is any numeric type, read as a float64. Values beyond 2^53 lose
// precision in that conversion, so this is the wrong assertion for
// large integers; compare those exactly.
//
// A NaN anywhere fails, on either side or as the tolerance, because no
// tolerance contains it. Note that this needs stating in the code: a
// bare comparison would pass a NaN, since every comparison against one
// is false.
func CloseTo(seat Seat, mode Mode, got any, want, tolerance float64, msg string) {
	seat.Helper()

	f, ok := floatOf(got)
	if !ok {
		Report(seat, mode, "%s: tolerance requires a number, got %T", msg, got)
		return
	}

	// Every comparison against NaN is false, so a bare `diff > tolerance`
	// would pass a NaN rather than reject it. Name the case instead.
	diff := math.Abs(f - want)
	if math.IsNaN(diff) || math.IsNaN(tolerance) {
		Report(seat, mode, "%s: %v is not within %v of %v: NaN is outside every tolerance",
			msg, f, tolerance, want)
		return
	}
	if diff > tolerance {
		Report(seat, mode, "%s: %v is not within %v of %v", msg, f, tolerance, want)
	}
}

// InRange reports when got falls outside the closed interval
// [low, high]. Both ends are included.
//
// got is any numeric type, read as a float64, with the precision limit
// [CloseTo] describes. Passing low above high always fails, and says
// so rather than reporting the value.
func InRange(seat Seat, mode Mode, got any, low, high float64, msg string) {
	seat.Helper()

	if low > high {
		Report(seat, mode, "%s: empty range [%v, %v]", msg, low, high)
		return
	}

	f, ok := floatOf(got)
	if !ok {
		Report(seat, mode, "%s: range requires a number, got %T", msg, got)
		return
	}

	// NaN compares false against both bounds, so testing the bounds alone
	// would admit it. See [CloseTo].
	if math.IsNaN(f) {
		Report(seat, mode, "%s: NaN is not in [%v, %v]", msg, low, high)
		return
	}
	if f < low || f > high {
		Report(seat, mode, "%s: %v is not in [%v, %v]", msg, f, low, high)
	}
}

// floatOf reads any numeric value as a float64, so one comparison
// serves every width and signedness. A float64 holds every int64
// exactly up to 2^53; beyond that this loses precision, which the
// numeric assertions state.
func floatOf(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0, false
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}
