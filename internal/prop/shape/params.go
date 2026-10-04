// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"encoding/json"
	"maps"
	"math"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"time"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// The bounds of the date and time shapes by default: 0001-01-01T00:00:00Z
// and 9999-12-31T23:59:59Z in seconds since the epoch, and the same dates
// in days.
const (
	// firstSecond is the first second of the year 1.
	firstSecond = -62_135_596_800
	// lastSecond is the last second of the year 9999.
	lastSecond = 253_402_300_799
	// secondsPerDay is the seconds in a day.
	secondsPerDay = 86_400
	// firstDay is the day of firstSecond, which starts a day.
	firstDay = firstSecond / secondsPerDay
	// lastDay is the day of lastSecond.
	lastDay = lastSecond / secondsPerDay
)

// The calendar that a bound of a date or a time states.
const (
	// firstYear and lastYear are the years that the date and time shapes
	// state.
	firstYear = 1
	lastYear  = 9999
	// lastHour, lastMinute and lastSecondOfMinute are the last hour, minute
	// and second of a day.
	lastHour           = 23
	lastMinute         = 59
	lastSecondOfMinute = 59
	// secondsPerHour and secondsPerMinute are the seconds in an hour and in
	// a minute.
	secondsPerHour   = 3_600
	secondsPerMinute = 60
	// decimalBase is the base of a decimal's digits.
	decimalBase = 10
)

// units are the units of a time shape, as the count of them in a second.
var units = map[string]int64{"s": 1, "ms": 1_000, "us": 1_000_000, "ns": 1_000_000_000}

// safeInteger is 2^53 - 1, the largest integer that a JSON integer states.
// A larger one is a decimal string.
var safeInteger = big.NewInt(1<<53 - 1)

// The text forms of a date, a time of day, a date and time, and a decimal.
// A date and time ends in Z when it is an instant in UTC.
var (
	datePattern     = regexp.MustCompile(`^([0-9]{4})-([0-9]{2})-([0-9]{2})$`)
	timePattern     = regexp.MustCompile(`^([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,9}))?$`)
	dateTimePattern = regexp.MustCompile(
		`^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,9}))?(Z?)$`)
	decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
)

// The parts of a match of dateTimePattern: the whole match and its eight
// groups, of which the last is the Z of an instant in UTC.
const (
	dateTimeParts = 9
	zonePart      = 8
)

// integer returns the integer that v states as a typed literal states one:
// a JSON integer within 2^53 - 1 in magnitude, or the canonical decimal
// string of one beyond it. It returns nil for any other value.
func integer(v any) *big.Int {
	switch v := v.(type) {
	case json.Number:
		if n, ok := new(big.Int).SetString(string(v), decimalBase); ok && n.CmpAbs(safeInteger) <= 0 {
			return n
		}
	case string:
		if n, ok := new(big.Int).SetString(v, decimalBase); ok && n.String() == v && n.CmpAbs(safeInteger) > 0 {
			return n
		}
	}
	return nil
}

// intParam returns the integer of the parameter key of n, and a fault at
// the key for a value that is no integer.
func intParam(n node, key string) (*big.Int, error) {
	if v := integer(n[key]); v != nil {
		return v, nil
	}
	return nil, fault.At(unreadable("%s is no integer", show(n[key])), fault.Field(key))
}

// count returns the count that the parameter key of n states, and false
// when n states none. It returns a fault at the key for a value that is no
// count of an int.
func count(n node, key string) (int, bool, error) {
	if n[key] == nil {
		return 0, false, nil
	}
	v, err := intParam(n, key)
	if err != nil {
		return 0, false, err
	}
	if v.Sign() < 0 {
		return 0, false, fault.At(unreadable("%s is below 0", v), fault.Field(key))
	}
	if !v.IsInt64() || v.Int64() > math.MaxInt {
		return 0, false, fault.At(unreadable("%s is more than an int counts", v), fault.Field(key))
	}
	return int(v.Int64()), true, nil
}

// sizes returns the sizes that min_size, 0 by default, and max_size,
// unbounded by default, state.
func sizes(n node) (choice.Sizes, error) {
	least, _, err := count(n, minSizeKey)
	if err != nil {
		return choice.Sizes{}, err
	}
	most, bounded, err := count(n, maxSizeKey)
	if err != nil {
		return choice.Sizes{}, err
	}
	if !bounded {
		unbounded, _ := choice.NewUnboundedSizes(least)
		return unbounded, nil
	}
	s, err := choice.NewSizes(least, most)
	if err != nil {
		return choice.Sizes{}, unreadable("the sizes [%d, %d] are empty", least, most)
	}
	return s, nil
}

// unit returns the units in a second of the unit that a time shape states.
func unit(n node) (int64, error) {
	name, _ := n[unitKey].(string)
	if per, ok := UnitsPerSecond(name); ok {
		return per, nil
	}
	return 0, fault.At(unreadable("%s is none of %s", show(n[unitKey]), show(slices.Sorted(maps.Keys(units)))),
		fault.Field(unitKey))
}

// UnitsPerSecond returns the units in a second of the unit of a time shape:
// 1 for s, 1,000 for ms, 1,000,000 for us and 1,000,000,000 for ns. It
// reports false for any other unit.
func UnitsPerSecond(unit string) (int64, bool) {
	per, ok := units[unit]
	return per, ok
}

// number returns the float that v states as a typed literal states one: a
// JSON number, or one of the names NaN, Inf and -Inf. It reports false for
// any other value.
func number(v any) (float64, bool) {
	switch v := v.(type) {
	case json.Number:
		f, err := strconv.ParseFloat(string(v), 64)
		return f, err == nil || math.IsInf(f, 0)
	case string:
		switch v {
		case "NaN":
			return math.NaN(), true
		case "Inf":
			return math.Inf(1), true
		case "-Inf":
			return math.Inf(-1), true
		}
	}
	return 0, false
}

// digits returns the integer of a run of decimal digits, which a pattern of
// this file matched.
func digits(text string) int64 {
	n, _ := strconv.ParseInt(text, decimalBase, 64)
	return n
}

// day returns the days since 1970-01-01 of a date of the proleptic
// Gregorian calendar, for a year from 1 to 9999.
func day(year, month, date string) (int64, error) {
	y, m, d := int(digits(year)), time.Month(digits(month)), int(digits(date))
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	if y < firstYear || y > lastYear || t.Month() != m || t.Day() != d {
		return 0, unreadable("%s-%s-%s is no date", year, month, date)
	}
	return t.Unix() / secondsPerDay, nil
}

// clock returns the seconds since midnight of a time of day.
func clock(hour, minute, second string) (int64, error) {
	h, m, s := digits(hour), digits(minute), digits(second)
	if h > lastHour || m > lastMinute || s > lastSecondOfMinute {
		return 0, unreadable("%s:%s:%s is no time", hour, minute, second)
	}
	return h*secondsPerHour + m*secondsPerMinute + s, nil
}

// fraction returns the units within a second that the digits of a fraction
// of a second state, at per units in a second.
func fraction(text string, per int64) (int64, error) {
	if text == "" {
		return 0, nil
	}
	scale := int64(math.Pow10(len(text)))
	scaled := digits(text) * per
	if scaled%scale != 0 {
		return 0, unreadable("the fraction .%s is finer than the unit", text)
	}
	return scaled / scale, nil
}

// bounds returns the integers that the min and the max of n state, as parse
// reads the value of each key, inside [lo, hi], which are the bounds by
// default. The fault of parse is at its key.
func bounds(n node, lo, hi int64, parse func(v any) (int64, error)) (int64, int64, error) {
	least, most := lo, hi
	var err error
	if v := n[minKey]; v != nil {
		if least, err = parse(v); err != nil {
			return 0, 0, fault.At(err, fault.Field(minKey))
		}
	}
	if v := n[maxKey]; v != nil {
		if most, err = parse(v); err != nil {
			return 0, 0, fault.At(err, fault.Field(maxKey))
		}
	}
	if least < lo || least > most || most > hi {
		return 0, 0, unreadable("the bounds [%d, %d] are empty or outside [%d, %d]", least, most, lo, hi)
	}
	return least, most, nil
}

// literalInteger returns the integer that a bound states as a typed literal
// does.
func literalInteger(v any) (int64, error) {
	n := integer(v)
	if n == nil || !n.IsInt64() {
		return 0, unreadable("%s is no integer", show(v))
	}
	return n.Int64(), nil
}

// pair is a bound of a two-choice shape: its first part, such as the
// seconds or the date, and its second part.
type pair [2]int64

// before reports whether p comes before q: an earlier first part, or the
// same first part and an earlier second part.
func (p pair) before(q pair) bool {
	return p[0] < q[0] || p[0] == q[0] && p[1] < q[1]
}

// pairBounds returns the bounds that the min and the max of n state, as
// parse reads each, inside [lo, hi], which are the bounds by default. The
// fault of parse is at its key.
func pairBounds(n node, lo, hi pair, parse func(any) (pair, error)) (pair, pair, error) {
	least, most := lo, hi
	var err error
	if v := n[minKey]; v != nil {
		if least, err = parse(v); err != nil {
			return pair{}, pair{}, fault.At(err, fault.Field(minKey))
		}
	}
	if v := n[maxKey]; v != nil {
		if most, err = parse(v); err != nil {
			return pair{}, pair{}, fault.At(err, fault.Field(maxKey))
		}
	}
	if least.before(lo) || most.before(least) || hi.before(most) {
		return pair{}, pair{}, unreadable("the bounds %v to %v are empty or out of range", least, most)
	}
	return least, most, nil
}

// secondPart returns the bounds of the second choice of a two-choice shape,
// whose first choice is first: [0, top], except where first equals the
// first part of a bound, where it stops at that bound's second part.
func secondPart(first int64, lo, hi pair, top int64) choice.IntegerBounds {
	least, most := int64(0), top
	if first == lo[0] {
		least = lo[1]
	}
	if first == hi[0] {
		most = hi[1]
	}
	return choice.MustIntegerBounds(choice.IntOf(least), choice.IntOf(most))
}
