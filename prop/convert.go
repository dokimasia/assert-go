// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"math/big"
	"net/netip"
	"sync"
	"time"
	"unicode/utf8"
	"uuid"

	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/shape"
	"go.dokimi.dev/assert/internal/prop/zone"
)

// The lengths that the date and time shapes count in.
const (
	// secondsPerDay is the seconds in a day.
	secondsPerDay = 86_400
	// nanosPerSecond is the nanoseconds in a second.
	nanosPerSecond = 1_000_000_000
	// decimalBase is the base of a decimal's digits.
	decimalBase = 10
	// day is the length of a day, which bounds a time of day.
	day = 24 * time.Hour
)

// timeUnit is the unit of a time shape: the units in a second, and the
// nanoseconds of one unit.
type timeUnit struct {
	// per is the units in a second.
	per int64
	// nanos is the nanoseconds of one unit.
	nanos int64
}

// unitOf returns the unit that a time shape names, and false for a name
// that is no unit of the definition.
func unitOf(name string) (timeUnit, bool) {
	per, ok := shape.UnitsPerSecond(name)
	if !ok {
		return timeUnit{}, false
	}
	return timeUnit{per: per, nanos: nanosPerSecond / per}, true
}

// whole returns the units of ns nanoseconds, and an error that names v for
// an ns that is no whole number of units.
func (u timeUnit) whole(ns int64, v any) (int64, error) {
	if ns%u.nanos != 0 {
		return 0, uninvertible("%v is finer than the shape's unit of %v", v, time.Duration(u.nanos))
	}
	return ns / u.nanos, nil
}

// twoParts returns the record of two named parts, the neutral value of a
// time shape of two parts.
func twoParts(first string, a any, second string, b any) literal.Record {
	return literal.Record{Fields: []literal.Field{{Name: first, Value: a}, {Name: second, Value: b}}}
}

// partsOf returns the values of the two parts of v, the neutral value of a
// time shape of two parts.
func partsOf(v any) (any, any) {
	r := v.(literal.Record)
	return r.Fields[0].Value, r.Fields[1].Value
}

// instant returns the time in UTC of an instant's neutral value at u.
func (u timeUnit) instant(v any) time.Time {
	seconds, units := partsOf(v)
	return time.Unix(seconds.(int64), units.(int64)*u.nanos).UTC()
}

// instantOf returns the neutral value of t as an instant at u, and an error
// for a t finer than u. An instant states no location.
func (u timeUnit) instantOf(t time.Time) (any, error) {
	units, err := u.whole(int64(t.Nanosecond()), t)
	if err != nil {
		return nil, err
	}
	return twoParts(shape.SecondsPart, t.Unix(), shape.UnitsPart, units), nil
}

// date returns the midnight in UTC of a date's neutral value, its days since
// 1970-01-01.
func date(v any) time.Time {
	return time.Unix(v.(int64)*secondsPerDay, 0).UTC()
}

// dateOf returns the neutral value of t as a date, and an error for a t that
// is no midnight in UTC.
func dateOf(t time.Time) (any, error) {
	if t.Location() != time.UTC || t.Unix()%secondsPerDay != 0 || t.Nanosecond() != 0 {
		return nil, uninvertible("%v is no midnight in UTC, which a date is", t)
	}
	return t.Unix() / secondsPerDay, nil
}

// local returns the time in UTC whose fields state a local date and time's
// neutral value at u.
func (u timeUnit) local(v any) time.Time {
	days, units := partsOf(v)
	since := units.(int64)
	return time.Unix(days.(int64)*secondsPerDay+since/u.per, since%u.per*u.nanos).UTC()
}

// localOf returns the neutral value of t, a time in UTC, as a local date and
// time at u, and an error for a t in another location or finer than u.
func (u timeUnit) localOf(t time.Time) (any, error) {
	if t.Location() != time.UTC {
		return nil, uninvertible("%v is not in UTC, whose fields state a local date and time", t)
	}
	units, err := u.whole(int64(t.Nanosecond()), t)
	if err != nil {
		return nil, err
	}
	seconds := t.Unix()
	days := floorDiv(seconds, secondsPerDay)
	return twoParts(shape.DatePart, days, shape.TimeOfDayPart, (seconds-days*secondsPerDay)*u.per+units), nil
}

// duration returns the duration of a neutral value of units at u: a
// duration, or a time of day since midnight.
func (u timeUnit) duration(v any) time.Duration {
	return time.Duration(v.(int64) * u.nanos)
}

// durationOf returns the neutral value of d as a duration at u, and an error
// for a d finer than u.
func (u timeUnit) durationOf(d time.Duration) (any, error) {
	units, err := u.whole(int64(d), d)
	if err != nil {
		return nil, err
	}
	return units, nil
}

// timeOfDayOf returns the neutral value of d as a time of day at u, and an
// error for a d outside [0, 24h) or finer than u.
func (u timeUnit) timeOfDayOf(d time.Duration) (any, error) {
	if d < 0 || d >= day {
		return nil, uninvertible("%v is outside [0, 24h), where a time of day is", d)
	}
	return u.durationOf(d)
}

// locations returns the location of each zone of the list by name, which
// the platform's time-zone database resolves once per process, and the
// fault of the first zone that the database lacks.
var locations = sync.OnceValues(func() (map[string]*time.Location, error) {
	return zone.Locations(time.LoadLocation)
})

// location returns the location of a zone's neutral value, the name of a
// zone of the list. The read of a shape with a zone resolves each zone of
// the list first.
func location(name any) *time.Location {
	resolved, _ := locations()
	return resolved[name.(string)]
}

// zoneOf returns the neutral value of loc, its name, and an error for a nil
// loc.
func zoneOf(loc *time.Location) (any, error) {
	if loc == nil {
		return nil, uninvertible("a nil %T is no zone", loc)
	}
	return loc.String(), nil
}

// zoned returns the time in its zone of a zoned date and time's neutral
// value at u.
func (u timeUnit) zoned(v any) time.Time {
	instant, name := partsOf(v)
	return u.instant(instant).In(location(name))
}

// zonedOf returns the neutral value of t as a zoned date and time at u, and
// an error for a t finer than u.
func (u timeUnit) zonedOf(t time.Time) (any, error) {
	instant, err := u.instantOf(t)
	if err != nil {
		return nil, err
	}
	return twoParts(shape.InstantPart, instant, shape.ZonePart, t.Location().String()), nil
}

// wall returns the wall time of a wall time's neutral value at u.
func (u timeUnit) wall(v any) WallTime {
	local, name := partsOf(v)
	return WallTime{Local: u.local(local), Zone: location(name)}
}

// wallOf returns the neutral value of w as a wall time at u, and an error
// for a w whose local time is not in UTC or is finer than u, and for a nil
// zone.
func (u timeUnit) wallOf(w WallTime) (any, error) {
	local, err := u.localOf(w.Local)
	if err != nil {
		return nil, err
	}
	name, err := zoneOf(w.Zone)
	if err != nil {
		return nil, err
	}
	return twoParts(shape.LocalPart, local, shape.ZonePart, name), nil
}

// uuidValue returns the UUID of a uuid shape's neutral value, its 16 bytes.
func uuidValue(v any) uuid.UUID {
	return uuid.UUID(v.([]byte))
}

// uuidOf returns the neutral value of u, its 16 bytes.
func uuidOf(u uuid.UUID) any {
	return u[:]
}

// address returns the address of an ip-address shape's neutral value, its 4
// or 16 bytes in network order.
func address(v any) netip.Addr {
	a, _ := netip.AddrFromSlice(v.([]byte))
	return a
}

// addressOf returns the neutral value of a, its 4 or 16 bytes in network
// order, and an error for the zero Addr and for an address with a zone.
func addressOf(a netip.Addr) (any, error) {
	if !a.IsValid() {
		return nil, uninvertible("the zero %T is no address", a)
	}
	if a.Zone() != "" {
		return nil, uninvertible("%v states the zone %q, which an ip-address shape has no part for", a, a.Zone())
	}
	if a.Is4() {
		b := a.As4()
		return b[:], nil
	}
	b := a.As16()
	return b[:], nil
}

// scale is the scale of a decimal shape: its digits after the point, and
// ten to their power.
type scale struct {
	// digits are the digits after the point.
	digits int
	// factor is ten to the power of digits.
	factor *big.Int
}

// scaleOf returns the scale of digits after the point.
func scaleOf(digits int) scale {
	return scale{digits: digits, factor: new(big.Int).Exp(big.NewInt(decimalBase), big.NewInt(int64(digits)), nil)}
}

// decimal returns the decimal of a neutral value, its unscaled integer.
func (s scale) decimal(v any) *big.Rat {
	return new(big.Rat).SetFrac(big.NewInt(v.(int64)), s.factor)
}

// decimalOf returns the neutral value of r, its unscaled integer, and an
// error for a nil r, an r with more digits than the scale, and an r whose
// unscaled integer is outside the int64 range.
func (s scale) decimalOf(r *big.Rat) (any, error) {
	if r == nil {
		return nil, uninvertible("a nil %T is no decimal", r)
	}
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(s.factor))
	if !scaled.IsInt() {
		return nil, uninvertible("%s has more digits than the scale of %d", r.RatString(), s.digits)
	}
	if !scaled.Num().IsInt64() {
		return nil, uninvertible("%s is outside the int64 range at the scale of %d", r.RatString(), s.digits)
	}
	return scaled.Num().Int64(), nil
}

// char returns the character of a char shape's neutral value, a string of
// one character.
func char(v any) rune {
	r, _ := utf8.DecodeRuneInString(v.(string))
	return r
}

// charOf returns the neutral value of r, the string of one character, and
// an error for an r that is no character.
func charOf(r rune) (any, error) {
	if !utf8.ValidRune(r) {
		return nil, uninvertible("%U is no character", r)
	}
	return string(r), nil
}

// floorDiv returns a divided by b, rounded towards negative infinity, for a
// b above 0.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b < 0 {
		q--
	}
	return q
}
