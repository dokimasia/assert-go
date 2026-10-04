// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"math"
	"math/big"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/zone"
)

// The ids of the domain shapes whose generators record a span of their own,
// labelled with the id.
const (
	instantID = "instant"
	localID   = "local-date-time"
	zonedID   = "zoned-date-time"
	wallID    = "wall-time"
	ipID      = "ip-address"
)

// The names of the fields of the neutral values of the date and time
// shapes, each a [literal.Record] of two fields in this order: an instant of
// SecondsPart and UnitsPart, a local-date-time of DatePart and
// TimeOfDayPart, a zoned-date-time of InstantPart and ZonePart, and a
// wall-time of LocalPart and ZonePart.
const (
	// SecondsPart is an instant's seconds since 1970-01-01T00:00:00Z.
	SecondsPart = "seconds"
	// UnitsPart is an instant's units within its second.
	UnitsPart = "units"
	// DatePart is a local date and time's days since 1970-01-01.
	DatePart = "date"
	// TimeOfDayPart is a local date and time's units since midnight.
	TimeOfDayPart = "time-of-day"
	// InstantPart is a zoned date and time's instant.
	InstantPart = "instant"
	// LocalPart is a wall time's local date and time.
	LocalPart = "local-date-time"
	// ZonePart is the name of the zone of a zoned date and time or of a
	// wall time.
	ZonePart = "zone"
)

// The parameters of the domain shapes.
const (
	// uuidBytes, ipv4Bytes and ipv6Bytes are the bytes of a UUID and of an
	// address of each IP version.
	uuidBytes = 16
	ipv4Bytes = 4
	ipv6Bytes = 16
	// ipv4 and ipv6 are the versions of an address.
	ipv4 = 4
	ipv6 = 6
	// maxOffset is the largest offset from UTC, in seconds: 18 hours.
	maxOffset = 64_800
	// nanosPerSecond is the nanoseconds in a second.
	nanosPerSecond = 1_000_000_000
	// nearOdds is the denominator of the odds, 1 in 4, that a zoned value
	// comes from its zone's offset changes, when the zone has one.
	nearOdds = 4
	// afterOdds is the denominator of the odds, 1 in 2, that a wall time
	// near a change takes the offset after it.
	afterOdds = 2
)

// stepBounds are the bounds [-1, 1] of the step from an offset change.
var stepBounds = choice.MustIntegerBounds(choice.IntOf(-1), choice.IntOf(1))

// zoneBounds returns the bounds of an index into the zone list.
func zoneBounds() choice.IntegerBounds {
	return choice.MustIntegerBounds(choice.Int{}, choice.UintOf(uint64(len(zone.List())-1)))
}

// wideInt returns the generator of the 128-bit integers in [lo, hi], as a
// *big.Int: its high 64 bits, signed when lo is negative, then its low 64
// bits, unsigned, in a span labelled int. The low half ranges over the
// whole unsigned range, except where the high half equals the high half of
// a bound, where it stops at that bound's low half. Each half is a value
// choice that the random phase may reuse.
//
// It runs backwards from an integer of any type, a *big.Int and a big.Int
// included, through its two halves.
func wideInt(lo, hi *big.Int) engine.Generator[any] {
	least, _ := halvesOf(lo)
	most, _ := halvesOf(hi)
	highs := choice.MustIntegerBounds(least[0], most[0])
	lows := func(high choice.Int) choice.IntegerBounds {
		l, h := choice.Int{}, lowMask
		if high == least[0] {
			l = least[1]
		}
		if high == most[0] {
			h = most[1]
		}
		return choice.MustIntegerBounds(l, h)
	}
	decode := func(c *engine.Case) any {
		var high, low choice.Int
		c.Span(intID, func() {
			high = c.Reusable(highs)
			low = c.Reusable(lows(high))
		})
		return joined(high, low)
	}
	return engine.NewInvertible(intID, decode, func(v any) ([]engine.Step, any, error) {
		n, ok := bigOf(v)
		if !ok {
			return nil, nil, uninvertible("%v is no integer", v)
		}
		halves, ok := halvesOf(n)
		if !ok {
			return nil, nil, uninvertible("%v is outside [%s, %s]", n, lo, hi)
		}
		first, err := engine.IntegerStep(highs, halves[0])
		if err != nil {
			return nil, nil, err
		}
		second, err := engine.IntegerStep(lows(halves[0]), halves[1])
		if err != nil {
			return nil, nil, err
		}
		return []engine.Step{first, second}, joined(halves[0], halves[1]), nil
	})
}

// halvesOf returns the high 64 bits of n, rounded towards negative infinity
// as a shift rounds, and the low 64 bits, unsigned. It reports false for an
// n whose high half fits neither 64-bit range.
func halvesOf(n *big.Int) ([2]choice.Int, bool) {
	high := new(big.Int).Rsh(n, half)
	low := new(big.Int).And(n, new(big.Int).SetUint64(math.MaxUint64))
	h, ok := intOf(high)
	return [2]choice.Int{h, choice.UintOf(low.Uint64())}, ok
}

// intOf returns n as a choice's integer, and false for an n outside both
// 64-bit ranges.
func intOf(n *big.Int) (choice.Int, bool) {
	if n.IsInt64() {
		return choice.IntOf(n.Int64()), true
	}
	return choice.UintOf(n.Uint64()), n.IsUint64()
}

// joined returns the integer of a high and a low half: high * 2^64 + low.
func joined(high, low choice.Int) *big.Int {
	n := new(big.Int).SetUint64(high.Magnitude())
	if high.Negative() {
		n.Neg(n)
	}
	n.Lsh(n, half)
	return n.Add(n, new(big.Int).SetUint64(low.Magnitude()))
}

// bigOf returns v as an integer: a *big.Int, a big.Int, or a value of any
// integer type. It reports false for any other value.
func bigOf(v any) (*big.Int, bool) {
	switch v := v.(type) {
	case *big.Int:
		return v, v != nil
	case big.Int:
		return &v, true
	}
	i, ok := engine.IntegerValue(v)
	if !ok {
		return nil, false
	}
	n := new(big.Int).SetUint64(i.Magnitude())
	if i.Negative() {
		n.Neg(n)
	}
	return n, true
}

// uuidShape returns the generator of a UUID's 16 bytes.
func uuidShape(*reader, node) (engine.Generator[any], error) {
	return engine.Erase(fixedBytes(uuidBytes)), nil
}

// fixedBytes returns the generator of byte strings of exactly count bytes.
func fixedBytes(count int) engine.Generator[[]byte] {
	exactly, _ := choice.NewSizes(count, count)
	return engine.Bytes(exactly)
}

// ipShape returns the generator of an IP address of the stated version, or
// of either.
func ipShape(_ *reader, n node) (engine.Generator[any], error) {
	version := n[versionKey]
	if version == nil {
		return ipOf(), nil
	}
	if v := integer(version); v != nil && v.IsInt64() {
		switch v.Int64() {
		case ipv4:
			return engine.Erase(fixedBytes(ipv4Bytes)), nil
		case ipv6:
			return engine.Erase(fixedBytes(ipv6Bytes)), nil
		}
	}
	return engine.Generator[any]{}, fault.At(unreadable("%s is neither 4 nor 6", show(version)),
		fault.Field(versionKey))
}

// ipOf returns the generator of an address of either version: an index
// that decides structure, 0 for version 4 and 1 for version 6, in a span
// labelled ip-address, then the address's 4 or 16 bytes. It runs backwards
// from a []byte of 4 or 16 bytes.
func ipOf() engine.Generator[any] {
	versions := [...]engine.Generator[[]byte]{fixedBytes(ipv4Bytes), fixedBytes(ipv6Bytes)}
	decode := func(c *engine.Case) any {
		var address []byte
		c.Span(ipID, func() {
			address = versions[c.Structure(bit, 0).Magnitude()].Decode(c)
		})
		return address
	}
	return engine.NewInvertible(ipID, decode, func(v any) ([]engine.Step, any, error) {
		address, ok := v.([]byte)
		index := slices.Index([]int{ipv4Bytes, ipv6Bytes}, len(address))
		if !ok || index < 0 {
			return nil, nil, uninvertible("%v is no address of 4 or 16 bytes", v)
		}
		// The bytes of a version produce every address of its length.
		steps, value, _ := versions[index].Inverse(address)
		return append([]engine.Step{indexStep(bit, index)}, steps...), value, nil
	})
}

// decimalShape returns the generator of a decimal's unscaled value at the
// stated scale, as an int64. A bound with more digits than the scale rounds
// inward.
func decimalShape(_ *reader, n node) (engine.Generator[any], error) {
	scale, stated, err := count(n, scaleKey)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if !stated {
		return engine.Generator[any]{}, fault.At(unreadable("null is no scale"), fault.Field(scaleKey))
	}
	factor := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(decimalBase), big.NewInt(int64(scale)), nil))
	bound := func(key string, round func(*big.Rat) *big.Int, fallback int64) (*big.Int, error) {
		v := n[key]
		if v == nil {
			return big.NewInt(fallback), nil
		}
		text, ok := v.(string)
		if !ok || !decimalPattern.MatchString(text) {
			return nil, fault.At(unreadable("%s is no decimal", show(v)), fault.Field(key))
		}
		r, _ := new(big.Rat).SetString(text)
		return round(r.Mul(r, factor)), nil
	}
	least, err := bound(minKey, ceiling, math.MinInt64)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	most, err := bound(maxKey, floor, math.MaxInt64)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if !least.IsInt64() || !most.IsInt64() || least.Cmp(most) > 0 {
		return engine.Generator[any]{}, unreadable("the bounds [%s, %s] are empty or outside [%d, %d]",
			least, most, int64(math.MinInt64), int64(math.MaxInt64))
	}
	return engine.Erase(engine.Integer(least.Int64(), most.Int64())), nil
}

// floor returns the largest integer at most r.
func floor(r *big.Rat) *big.Int {
	return new(big.Int).Div(r.Num(), r.Denom())
}

// ceiling returns the smallest integer at least r.
func ceiling(r *big.Rat) *big.Int {
	return new(big.Int).Neg(floor(new(big.Rat).Neg(r)))
}

// instantShape returns the generator of an instant at the stated unit, from
// 0001-01-01T00:00:00Z to the last unit of 9999 unless min and max narrow
// it. A bound is a date and time in UTC.
func instantShape(_ *reader, n node) (engine.Generator[any], error) {
	per, err := unit(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	parse := func(v any) (pair, error) {
		text, _ := v.(string)
		m := dateTimePattern.FindStringSubmatch(text)
		if len(m) != dateTimeParts || m[zonePart] != "Z" {
			return pair{}, unreadable("%s is no instant in UTC", show(v))
		}
		return dateTime(m, per)
	}
	lo, hi, err := pairBounds(n, pair{firstSecond, 0}, pair{lastSecond, per - 1}, parse)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return instantOf(per, lo, hi), nil
}

// dateTime returns the seconds since the epoch and the units within the
// second that the parts of a date and time state.
func dateTime(m []string, per int64) (pair, error) {
	d, err := day(m[1], m[2], m[3])
	if err != nil {
		return pair{}, err
	}
	s, err := clock(m[4], m[5], m[6])
	if err != nil {
		return pair{}, err
	}
	u, err := fraction(m[7], per)
	if err != nil {
		return pair{}, err
	}
	return pair{d*secondsPerDay + s, u}, nil
}

// instantOf returns the generator of the instants from lo to hi at per
// units in a second, as a [literal.Record] of its seconds and its units:
// the seconds, then the units within the second when per is above 1, each a
// value choice that the random phase may reuse, in a span labelled instant.
// It runs backwards from such a record, and the fault of a part is at the
// part.
func instantOf(per int64, lo, hi pair) engine.Generator[any] {
	seconds := choice.MustIntegerBounds(choice.IntOf(lo[0]), choice.IntOf(hi[0]))
	decode := func(c *engine.Case) any {
		var s, u int64
		c.Span(instantID, func() {
			s = int64Of(c.Reusable(seconds))
			if per > 1 {
				u = int64Of(c.Reusable(secondPart(s, lo, hi, per-1)))
			}
		})
		return twoParts(SecondsPart, s, UnitsPart, u)
	}
	return engine.NewInvertible(instantID, decode, func(v any) ([]engine.Step, any, error) {
		s, u, err := integerParts(v, SecondsPart, UnitsPart)
		if err != nil {
			return nil, nil, err
		}
		first, err := engine.IntegerStep(seconds, s)
		if err != nil {
			return nil, nil, fault.At(err, fault.Field(SecondsPart))
		}
		if per == 1 {
			if u != (choice.Int{}) {
				return nil, nil, fault.At(uninvertible("%v is not 0, the units of an instant at seconds", u),
					fault.Field(UnitsPart))
			}
			return []engine.Step{first}, twoParts(SecondsPart, int64Of(s), UnitsPart, int64(0)), nil
		}
		second, err := engine.IntegerStep(secondPart(int64Of(s), lo, hi, per-1), u)
		if err != nil {
			return nil, nil, fault.At(err, fault.Field(UnitsPart))
		}
		return []engine.Step{first, second}, twoParts(SecondsPart, int64Of(s), UnitsPart, int64Of(u)), nil
	})
}

// int64Of returns i, which a bound inside the int64 range admits, as an
// int64.
func int64Of(i choice.Int) int64 {
	n, _ := i.Int64()
	return n
}

// twoParts returns the [literal.Record] of two named parts.
func twoParts(first string, a any, second string, b any) literal.Record {
	return literal.Record{Fields: []literal.Field{{Name: first, Value: a}, {Name: second, Value: b}}}
}

// integerParts returns the two integers of v, a [literal.Record] of the two
// named parts in order, and a fault at a part that is no integer.
func integerParts(v any, first, second string) (choice.Int, choice.Int, error) {
	values, err := recordValues(v, []string{first, second})
	if err != nil {
		return choice.Int{}, choice.Int{}, err
	}
	a, ok := engine.IntegerValue(values[0])
	if !ok {
		return choice.Int{}, choice.Int{}, fault.At(uninvertible("%v is no integer", values[0]), fault.Field(first))
	}
	b, ok := engine.IntegerValue(values[1])
	if !ok {
		return choice.Int{}, choice.Int{}, fault.At(uninvertible("%v is no integer", values[1]), fault.Field(second))
	}
	return a, b, nil
}

// dateShape returns the generator of a date, in days since 1970-01-01, as
// an int64. A bound is a date.
func dateShape(_ *reader, n node) (engine.Generator[any], error) {
	parse := func(v any) (int64, error) {
		text, _ := v.(string)
		m := datePattern.FindStringSubmatch(text)
		if m == nil {
			return 0, unreadable("%s is no date", show(v))
		}
		return day(m[1], m[2], m[3])
	}
	lo, hi, err := bounds(n, firstDay, lastDay, parse)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return engine.Erase(engine.Integer(lo, hi)), nil
}

// timeOfDayShape returns the generator of a time of day, in units since
// midnight, as an int64. A bound is a time of day.
func timeOfDayShape(_ *reader, n node) (engine.Generator[any], error) {
	per, err := unit(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	parse := func(v any) (int64, error) {
		text, _ := v.(string)
		m := timePattern.FindStringSubmatch(text)
		if m == nil {
			return 0, unreadable("%s is no time of day", show(v))
		}
		seconds, bad := clock(m[1], m[2], m[3])
		if bad != nil {
			return 0, bad
		}
		units, bad := fraction(m[4], per)
		return seconds*per + units, bad
	}
	lo, hi, err := bounds(n, 0, secondsPerDay*per-1, parse)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return engine.Erase(engine.Integer(lo, hi)), nil
}

// localShape returns the generator of a date and a time of day without a
// zone. A bound is a date and time without a Z.
func localShape(_ *reader, n node) (engine.Generator[any], error) {
	per, err := unit(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	parse := func(v any) (pair, error) {
		text, _ := v.(string)
		m := dateTimePattern.FindStringSubmatch(text)
		if len(m) != dateTimeParts || m[zonePart] != "" {
			return pair{}, unreadable("%s is no local date and time", show(v))
		}
		at, bad := dateTime(m, per)
		return pair{floorDiv(at[0], secondsPerDay), mod(at[0], secondsPerDay)*per + at[1]}, bad
	}
	lo, hi, err := pairBounds(n, pair{firstDay, 0}, pair{lastDay, secondsPerDay*per - 1}, parse)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return localOf(per, lo, hi), nil
}

// localOf returns the generator of the local dates and times from lo to hi
// at per units in a second, as a [literal.Record] of its date and its time
// of day: the date, then the time of day, each a value choice that the
// random phase may reuse, in a span labelled local-date-time. It runs
// backwards from such a record, and the fault of a part is at the part.
func localOf(per int64, lo, hi pair) engine.Generator[any] {
	days := choice.MustIntegerBounds(choice.IntOf(lo[0]), choice.IntOf(hi[0]))
	top := secondsPerDay*per - 1
	decode := func(c *engine.Case) any {
		var d, t int64
		c.Span(localID, func() {
			d = int64Of(c.Reusable(days))
			t = int64Of(c.Reusable(secondPart(d, lo, hi, top)))
		})
		return twoParts(DatePart, d, TimeOfDayPart, t)
	}
	return engine.NewInvertible(localID, decode, func(v any) ([]engine.Step, any, error) {
		d, t, err := integerParts(v, DatePart, TimeOfDayPart)
		if err != nil {
			return nil, nil, err
		}
		first, err := engine.IntegerStep(days, d)
		if err != nil {
			return nil, nil, fault.At(err, fault.Field(DatePart))
		}
		second, err := engine.IntegerStep(secondPart(int64Of(d), lo, hi, top), t)
		if err != nil {
			return nil, nil, fault.At(err, fault.Field(TimeOfDayPart))
		}
		return []engine.Step{first, second}, twoParts(DatePart, int64Of(d), TimeOfDayPart, int64Of(t)), nil
	})
}

// durationShape returns the generator of a duration, in units, as an int64:
// at most the nanoseconds of the largest Go time.Duration, in the unit.
func durationShape(_ *reader, n node) (engine.Generator[any], error) {
	per, err := unit(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	largest := int64(math.MaxInt64) / (nanosPerSecond / per)
	lo, hi, err := bounds(n, -largest, largest, literalInteger)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return engine.Erase(engine.Integer(lo, hi)), nil
}

// offsetShape returns the generator of an offset from UTC, in seconds east
// of it, as an int64: at most 18 hours either way.
func offsetShape(_ *reader, n node) (engine.Generator[any], error) {
	lo, hi, err := bounds(n, -maxOffset, maxOffset, literalInteger)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return engine.Erase(engine.Integer(lo, hi)), nil
}

// zoneShape returns the generator of a zone of the list, by name, with UTC
// first.
func zoneShape(*reader, node) (engine.Generator[any], error) {
	listed := zone.List()
	names := make([]string, len(listed))
	for i, z := range listed {
		names[i] = z.Name
	}
	return engine.Erase(engine.SampledFrom(names...)), nil
}

// zonedShape returns the generator of an instant in a zone.
func zonedShape(_ *reader, n node) (engine.Generator[any], error) {
	per, err := unit(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return zonedOf(instantOf(per, pair{firstSecond, 0}, pair{lastSecond, per - 1}), per), nil
}

// zonedOf returns the generator of an instant in a zone, as a
// [literal.Record] of the instant and the zone's name, in a span labelled
// zoned-date-time: the zone, then, for a zone with an offset change, a coin
// of 1 in 4 that decides whether the value comes from the changes. Such a
// value is one of the zone's changes, through an index that decides
// structure, moved by a step from -1 to 1 units. Any other value takes the
// instant from instant, over the whole range.
//
// It runs backwards through the zone and the whole range, so a value at an
// offset change has the same one sequence of choices as any other value.
// The fault of a part is at the part.
func zonedOf(instant engine.Generator[any], per int64) engine.Generator[any] {
	decode := func(c *engine.Case) any {
		var value any
		var name string
		c.Span(zonedID, func() {
			z := pickZone(c)
			name = z.Name
			if !near(c, z) {
				value = instant.Decode(c)
				return
			}
			change, step := changeAndStep(c, z)
			total := change.At*per + step
			value = twoParts(SecondsPart, floorDiv(total, per), UnitsPart, mod(total, per))
		})
		return twoParts(InstantPart, value, ZonePart, name)
	}
	return engine.NewInvertible(zonedID, decode, func(v any) ([]engine.Step, any, error) {
		inner, name, head, err := zoneParts(v, InstantPart)
		if err != nil {
			return nil, nil, err
		}
		steps, value, err := instant.Inverse(inner)
		if err != nil {
			return nil, nil, fault.At(err, fault.Field(InstantPart))
		}
		return slices.Concat(head, steps), twoParts(InstantPart, value, ZonePart, name), nil
	})
}

// wallShape returns the generator of a wall time in a zone.
func wallShape(_ *reader, n node) (engine.Generator[any], error) {
	per, err := unit(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return wallOf(localOf(per, pair{firstDay, 0}, pair{lastDay, secondsPerDay*per - 1}), per), nil
}

// wallOf returns the generator of a wall time in a zone, which the code
// under test resolves, as a [literal.Record] of the local date and time and
// the zone's name, in a span labelled wall-time: the zone, then, for a zone
// with an offset change, a coin of 1 in 4 that decides whether the value
// comes from the changes. Such a value is one of the zone's changes, a step
// from -1 to 1 units, and a coin of 1 in 2 that picks the offset after the
// change over the one before it: the change's instant in that offset,
// moved by the step. Any other value takes the local date and time from
// local, over the whole range.
//
// It runs backwards through the zone and the whole range. The fault of a
// part is at the part.
func wallOf(local engine.Generator[any], per int64) engine.Generator[any] {
	decode := func(c *engine.Case) any {
		var value any
		var name string
		c.Span(wallID, func() {
			z := pickZone(c)
			name = z.Name
			if !near(c, z) {
				value = local.Decode(c)
				return
			}
			change, step := changeAndStep(c, z)
			offset := change.Before
			if c.Coin(1, afterOdds) {
				offset = change.After
			}
			total := (change.At+offset)*per + step
			value = twoParts(DatePart, floorDiv(total, secondsPerDay*per), TimeOfDayPart, mod(total, secondsPerDay*per))
		})
		return twoParts(LocalPart, value, ZonePart, name)
	}
	return engine.NewInvertible(wallID, decode, func(v any) ([]engine.Step, any, error) {
		inner, name, head, err := zoneParts(v, LocalPart)
		if err != nil {
			return nil, nil, err
		}
		steps, value, err := local.Inverse(inner)
		if err != nil {
			return nil, nil, fault.At(err, fault.Field(LocalPart))
		}
		return slices.Concat(head, steps), twoParts(LocalPart, value, ZonePart, name), nil
	})
}

// pickZone returns the zone of the list that an index that decides
// structure chooses.
func pickZone(c *engine.Case) zone.Zone {
	return zone.List()[c.Structure(zoneBounds(), 0).Magnitude()]
}

// near reports whether a value of z comes from one of its offset changes:
// a coin of 1 in 4 for a zone with a change, and no choice for a zone
// without one, such as UTC.
func near(c *engine.Case, z zone.Zone) bool {
	return len(z.Changes) > 0 && c.Coin(1, nearOdds)
}

// changeAndStep returns one of z's changes, through an index that decides
// structure, and a step from -1 to 1, a value choice that the random phase
// may reuse.
func changeAndStep(c *engine.Case, z zone.Zone) (zone.Change, int64) {
	changes := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(uint64(len(z.Changes)-1)))
	change := z.Changes[c.Structure(changes, 0).Magnitude()]
	return change, int64Of(c.Reusable(stepBounds))
}

// zoneParts returns the part of v, a [literal.Record] of the part and a
// zone, the zone's name, and the steps of the zone's index and of the
// whole range when the zone has a change. It returns a fault at the zone
// for a zone outside the list.
func zoneParts(v any, part string) (any, string, []engine.Step, error) {
	values, err := recordValues(v, []string{part, ZonePart})
	if err != nil {
		return nil, "", nil, err
	}
	name, _ := values[1].(string)
	listed := zone.List()
	index := slices.IndexFunc(listed, func(z zone.Zone) bool { return z.Name == name })
	if index < 0 {
		return nil, "", nil, fault.At(uninvertible("%v is no zone of the list", values[1]), fault.Field(ZonePart))
	}
	steps := []engine.Step{indexStep(zoneBounds(), index)}
	if len(listed[index].Changes) > 0 {
		steps = append(steps, bitStep(0))
	}
	return values[0], name, steps, nil
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

// mod returns the remainder of a divided by b that floorDiv leaves, from 0
// to b - 1, for a b above 0.
func mod(a, b int64) int64 {
	return a - floorDiv(a, b)*b
}
