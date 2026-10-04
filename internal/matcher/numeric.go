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
// precision in that conversion, so compare large integers exactly
// instead.
//
// A NaN fails, on either side or as the tolerance, because no tolerance
// contains it.
//
// # Allocation contract
//
// A passing call on a float64 allocates nothing.
func CloseTo(seat Seat, mode Mode, got any, want, tolerance float64, msg string) {
	seat.Helper()

	f, ok := floatOf(got)
	if !ok {
		Fail(seat, mode, "close-to", msg,
			map[string]any{"got": got, "want": want, "tolerance": tolerance})
		return
	}

	// Every comparison against a NaN is false, so diff > tolerance alone
	// passes a NaN. The NaN case is checked first.
	diff := math.Abs(f - want)
	if math.IsNaN(diff) || math.IsNaN(tolerance) {
		Fail(seat, mode, "close-to", msg,
			map[string]any{"got": got, "want": want, "tolerance": tolerance})
		return
	}
	if diff > tolerance {
		Fail(seat, mode, "close-to", msg,
			map[string]any{"got": got, "want": want, "tolerance": tolerance})
		return
	}
	Pass(seat, mode, "close-to", msg)
}

// InRange reports when got is outside the closed interval [low, high].
// Both ends are included.
//
// got is any numeric type, read as a float64, with the precision limit
// [CloseTo] describes. A range whose low is above its high, or whose
// bound is NaN, contains no number, so it fails whatever got is. Every
// comparison against NaN is false, so a NaN bound tested like any other
// would admit every value.
//
// # Allocation contract
//
// A passing call on a float64 allocates nothing.
func InRange(seat Seat, mode Mode, got any, low, high float64, msg string) {
	seat.Helper()

	if math.IsNaN(low) || math.IsNaN(high) || low > high {
		Fail(seat, mode, "in-range", msg,
			map[string]any{"got": got, "low": low, "high": high})
		return
	}

	f, ok := floatOf(got)
	if !ok {
		Fail(seat, mode, "in-range", msg,
			map[string]any{"got": got, "low": low, "high": high})
		return
	}

	// NaN compares false against both bounds, so testing the bounds alone
	// would admit it. See [CloseTo].
	if math.IsNaN(f) {
		Fail(seat, mode, "in-range", msg,
			map[string]any{"got": got, "low": low, "high": high})
		return
	}
	if f < low || f > high {
		Fail(seat, mode, "in-range", msg,
			map[string]any{"got": got, "low": low, "high": high})
		return
	}
	Pass(seat, mode, "in-range", msg)
}

// floatOf reads any numeric value as a float64, so one comparison
// serves every width and signedness. A float64 represents every int64 of
// magnitude up to 2^53 exactly, and loses precision above it, which the
// numeric assertions state. A nil value has the kind [reflect.Invalid],
// and is no number.
func floatOf(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
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
