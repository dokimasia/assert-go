// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"math"
	"math/big"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/shape"
	"go.dokimi.dev/assert/internal/prop/zone"
)

// The first second and the first day of the year 1, and the last of 9999.
const (
	firstSecond = -62_135_596_800
	lastSecond  = 253_402_300_799
	firstDay    = -719_162
	lastDay     = 2_932_896
)

// Shapes that the domain tests decode.
const (
	// bounded is a 128-bit int from 2^64 + 10 to 2^65.
	bounded = `{"shape":"int","width":128,"signed":false,"min":"18446744073709551626","max":"36893488147419103232"}`
	// late are the instants from half a second past 2020-01-01 in UTC.
	late = `{"shape":"instant","unit":"ms","min":"2020-01-01T00:00:00.5Z"}`
	// lateLocal are the local dates and times from half a second past
	// 2020-01-01.
	lateLocal = `{"shape":"local-date-time","unit":"ms","min":"2020-01-01T00:00:00.5"}`
)

// domainTrips are a shape of each domain kind, whose generated values each
// run back to choices that decode to them.
var domainTrips = []string{
	`{"shape":"int","width":128,"signed":true}`,
	bounded,
	`{"shape":"uuid"}`,
	`{"shape":"ip-address"}`,
	`{"shape":"decimal","scale":2,"min":"0.00","max":"100.00"}`,
	`{"shape":"instant","unit":"ns"}`,
	`{"shape":"instant","unit":"s"}`,
	`{"shape":"instant","unit":"ms","min":"2020-01-01T00:00:00.5Z","max":"2020-01-01T00:00:01.25Z"}`,
	`{"shape":"date"}`,
	`{"shape":"time-of-day","unit":"us"}`,
	`{"shape":"local-date-time","unit":"ms","min":"2020-01-01T00:00:00.5","max":"2020-01-02T00:00:00.25"}`,
	`{"shape":"duration","unit":"ms"}`,
	`{"shape":"offset"}`,
	`{"shape":"zone"}`,
	`{"shape":"zoned-date-time","unit":"ms"}`,
	`{"shape":"wall-time","unit":"s"}`,
}

// TestDomain checks the domain shapes: the value that each decodes from
// stated choices in the form that the definition states it, the choices
// that each value runs back to, and each shape and each value that a shape
// refuses, with where the fault is.
func TestDomain(t *testing.T) {
	t.Parallel()

	amsterdam := zone.List()[1]
	change := amsterdam.Changes[0]

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		values := []struct {
			name  string
			shape string
			give  []choice.Choice
			want  any
		}{
			{
				name:  "returns a UUID's 16 bytes",
				shape: `{"shape":"uuid"}`,
				give:  []choice.Choice{seq(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15)},
				want:  []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			},
			{
				name:  "returns an address of version 4",
				shape: `{"shape":"ip-address","version":4}`,
				give:  []choice.Choice{seq(127, 0, 0, 1)},
				want:  []byte{127, 0, 0, 1},
			},
			{
				name:  "returns an address of version 6",
				shape: `{"shape":"ip-address","version":6}`,
				want:  make([]byte, 16),
			},
			{
				name:  "returns an address of either version",
				shape: `{"shape":"ip-address"}`,
				give:  []choice.Choice{n(1), seq(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1)},
				want:  []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			},
			{
				name:  "returns a decimal's bound rounded inward",
				shape: `{"shape":"decimal","scale":2,"min":"0.005","max":"1.999"}`,
				give:  []choice.Choice{n(0)},
				want:  int64(1),
			},
			{
				name:  "returns a decimal's unscaled value",
				shape: `{"shape":"decimal","scale":2,"min":"0.005","max":"1.999"}`,
				give:  []choice.Choice{n(199)},
				want:  int64(199),
			},
			{
				name:  "returns the target for a decimal past its bounds",
				shape: `{"shape":"decimal","scale":2,"min":"-0.995","max":"1.999"}`,
				give:  []choice.Choice{n(200)},
				want:  int64(0),
			},
			{
				name:  "returns the largest decimal of scale 0",
				shape: `{"shape":"decimal","scale":0}`,
				give:  []choice.Choice{n(math.MaxInt64)},
				want:  int64(math.MaxInt64),
			},
			{
				name:  "returns an instant's seconds and nanoseconds",
				shape: `{"shape":"instant","unit":"ns"}`,
				give:  []choice.Choice{n(-1), n(999_999_999)},
				want:  record("seconds", int64(-1), "units", int64(999_999_999)),
			},
			{
				name:  "returns the first instant of the year 1",
				shape: `{"shape":"instant","unit":"us"}`,
				give:  []choice.Choice{n(firstSecond), n(0)},
				want:  record("seconds", int64(firstSecond), "units", int64(0)),
			},
			{
				name:  "returns the target for an instant before the year 1",
				shape: `{"shape":"instant","unit":"s"}`,
				give:  []choice.Choice{n(firstSecond - 1)},
				want:  record("seconds", int64(0), "units", int64(0)),
			},
			{
				name:  "returns the first date of the year 1",
				shape: `{"shape":"date"}`,
				give:  []choice.Choice{n(firstDay)},
				want:  int64(firstDay),
			},
			{
				name:  "returns the last date of the year 9999",
				shape: `{"shape":"date"}`,
				give:  []choice.Choice{n(lastDay)},
				want:  int64(lastDay),
			},
			{
				name:  "returns the last millisecond of a day",
				shape: `{"shape":"time-of-day","unit":"ms"}`,
				give:  []choice.Choice{n(86_399_999)},
				want:  int64(86_399_999),
			},
			{
				name:  "returns a local date and time of a minimum before the epoch",
				shape: `{"shape":"local-date-time","unit":"ms","min":"1969-12-31T23:59:59.5"}`,
				give:  []choice.Choice{n(-1), n(0)},
				want:  record("date", int64(-1), "time-of-day", int64(86_399_500)),
			},
			{
				name:  "returns the largest duration in nanoseconds",
				shape: `{"shape":"duration","unit":"ns"}`,
				give:  []choice.Choice{n(math.MaxInt64)},
				want:  int64(math.MaxInt64),
			},
			{
				name:  "returns the least duration in seconds",
				shape: `{"shape":"duration","unit":"s"}`,
				give:  []choice.Choice{n(-9_223_372_036)},
				want:  int64(-9_223_372_036),
			},
			{
				name:  "returns the target of a duration past the range in seconds",
				shape: `{"shape":"duration","unit":"s"}`,
				give:  []choice.Choice{n(9_223_372_037)},
				want:  int64(0),
			},
			{
				name:  "returns 18 hours west",
				shape: `{"shape":"offset"}`,
				give:  []choice.Choice{n(-64_800)},
				want:  int64(-64_800),
			},
			{
				name:  "returns the target past 18 hours",
				shape: `{"shape":"offset"}`,
				give:  []choice.Choice{n(64_801)},
				want:  int64(0),
			},
			{name: "returns UTC first", shape: `{"shape":"zone"}`, want: "UTC"},
			{
				name:  "returns the zone of an index",
				shape: `{"shape":"zone"}`,
				give:  []choice.Choice{n(1)},
				want:  amsterdam.Name,
			},
			{
				name:  "returns an instant in UTC, which makes no choice of a change",
				shape: `{"shape":"zoned-date-time","unit":"s"}`,
				give:  []choice.Choice{n(0), n(42)},
				want:  record("instant", record("seconds", int64(42), "units", int64(0)), "zone", "UTC"),
			},
			{
				name:  "returns an instant one unit before a change",
				shape: `{"shape":"zoned-date-time","unit":"ns"}`,
				give:  []choice.Choice{n(1), n(1), n(0), n(-1)},
				want: record(
					"instant",
					record("seconds", change.At-1, "units", int64(999_999_999)),
					"zone",
					amsterdam.Name,
				),
			},
			{
				name:  "returns an instant away from the changes",
				shape: `{"shape":"zoned-date-time","unit":"ms"}`,
				give:  []choice.Choice{n(1), n(0), n(7), n(8)},
				want:  record("instant", record("seconds", int64(7), "units", int64(8)), "zone", amsterdam.Name),
			},
			{
				name:  "returns a wall time at a change in the offset before it",
				shape: `{"shape":"wall-time","unit":"s"}`,
				give:  []choice.Choice{n(1), n(1), n(0), n(0), n(0)},
				want:  record("local-date-time", local(change.At+change.Before), "zone", amsterdam.Name),
			},
			{
				name:  "returns a wall time at a change in the offset after it",
				shape: `{"shape":"wall-time","unit":"s"}`,
				give:  []choice.Choice{n(1), n(1), n(0), n(0), n(1)},
				want:  record("local-date-time", local(change.At+change.After), "zone", amsterdam.Name),
			},
			{
				name:  "returns a wall time away from the changes",
				shape: `{"shape":"wall-time","unit":"s"}`,
				give:  []choice.Choice{n(1), n(0), n(5), n(6)},
				want: record(
					"local-date-time",
					record("date", int64(5), "time-of-day", int64(6)),
					"zone",
					amsterdam.Name,
				),
			},
		}
		for _, tt := range values {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, tt.shape, tt.give...)
				assert.Equal(t, got, tt.want, "the value")
			})
		}

		wide := []struct {
			name  string
			shape string
			give  []choice.Choice
			want  *big.Int
		}{
			{
				name:  "returns -1 of its two halves",
				shape: `{"shape":"int","width":128,"signed":true}`,
				give:  []choice.Choice{n(-1), allOnes},
				want:  big.NewInt(-1),
			},
			{
				name:  "returns 2^64 of its two halves",
				shape: `{"shape":"int","width":128,"signed":true}`,
				give:  []choice.Choice{n(1), n(0)},
				want:  power(64, 0),
			},
			{
				name:  "returns the least 128-bit int",
				shape: `{"shape":"int","width":128,"signed":true}`,
				give:  []choice.Choice{n(math.MinInt64), n(0)},
				want:  new(big.Int).Neg(power(127, 0)),
			},
			{
				name:  "returns the largest unsigned 128-bit int",
				shape: `{"shape":"int","width":128,"signed":false}`,
				give:  []choice.Choice{allOnes, allOnes},
				want:  power(128, -1),
			},
			{
				name:  "returns a low half that a minimum stops",
				shape: bounded,
				give:  []choice.Choice{n(1), n(3)},
				want:  power(64, 10),
			},
			{
				name:  "returns a low half that a maximum stops",
				shape: bounded,
				give:  []choice.Choice{n(2), n(9)},
				want:  power(65, 0),
			},
			{
				name:  "returns a low half inside a minimum's second",
				shape: bounded,
				give:  []choice.Choice{n(1), n(11)},
				want:  power(64, 11),
			},
		}
		for _, tt := range wide {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, tt.shape, tt.give...)
				assert.Equal(t, literal.Canonical(got), literal.Canonical(tt.want), "the 128-bit int")
			})
		}

		t.Run("returns an instant at seconds of one choice", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, `{"shape":"instant","unit":"s"}`, n(lastSecond))
			assert.Equal(t, got, any(record("seconds", int64(lastSecond), "units", int64(0))), "the last second")
			assert.Length(t, e.Case.Choices(), 1, "no choice of the units")
		})

		t.Run("returns a zoned value near a change in about one case in four", func(t *testing.T) {
			t.Parallel()
			g := read(t, `{"shape":"zoned-date-time","unit":"s"}`)
			near, changed := 0, 0
			for index := range uint64(4000) {
				e := engine.Generate(func(c *engine.Case) { engine.Draw(c, g, drawn) }, 11, index, nil)
				choices := e.Case.Choices()
				if len(zone.List()[choices[0].Integer.Magnitude()].Changes) > 0 {
					changed++
					if choices[1].Integer == choice.UintOf(1) {
						near++
					}
				}
			}
			assert.True(t, changed > 3000, "most zones have a change")
			assert.InRange(t, float64(near)/float64(changed), 0.22, 0.28, "about a quarter come from the changes")
		})

		atMin := fault.Path{fault.Field("min")}
		faults := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns an error at the null scale of a decimal",
				give:       `{"shape":"decimal","scale":null}`,
				wantPath:   fault.Path{fault.Field("scale")},
				wantReason: "null is no scale",
			},
			{
				name:       "returns an error at a decimal bound in an exponent",
				give:       `{"shape":"decimal","scale":2,"min":"1e3"}`,
				wantPath:   atMin,
				wantReason: `"1e3" is no decimal`,
			},
			{
				name:       "returns an error at a decimal maximum in an exponent",
				give:       `{"shape":"decimal","scale":2,"max":"-1e3"}`,
				wantPath:   fault.Path{fault.Field("max")},
				wantReason: `"-1e3" is no decimal`,
			},
			{
				name: "returns an error for decimal bounds outside the int64 range",
				give: `{"shape":"decimal","scale":2,"max":"92233720368547758.08"}`,
				wantReason: "the bounds [-9223372036854775808, 9223372036854775808] are empty or outside " +
					"[-9223372036854775808, 9223372036854775807]",
			},
			{
				name:       "returns an error at an address of version 5",
				give:       `{"shape":"ip-address","version":5}`,
				wantPath:   fault.Path{fault.Field("version")},
				wantReason: "5 is neither 4 nor 6",
			},
			{
				name:       "returns an error at a version that is no integer",
				give:       `{"shape":"ip-address","version":"4"}`,
				wantPath:   fault.Path{fault.Field("version")},
				wantReason: `"4" is neither 4 nor 6`,
			},
			{
				name:       "returns an error at an instant bound without a Z",
				give:       `{"shape":"instant","unit":"s","min":"2020-01-01T00:00:00"}`,
				wantPath:   atMin,
				wantReason: `"2020-01-01T00:00:00" is no instant in UTC`,
			},
			{
				name:       "returns an error at a date in another form",
				give:       `{"shape":"date","min":"2020/01/01"}`,
				wantPath:   atMin,
				wantReason: `"2020/01/01" is no date`,
			},
			{
				name:       "returns an error at a time of day in another form",
				give:       `{"shape":"time-of-day","unit":"s","min":"noon"}`,
				wantPath:   atMin,
				wantReason: `"noon" is no time of day`,
			},
			{
				name:       "returns an error at a local bound with a Z",
				give:       `{"shape":"local-date-time","unit":"s","min":"2020-01-01T00:00:00Z"}`,
				wantPath:   atMin,
				wantReason: `"2020-01-01T00:00:00Z" is no local date and time`,
			},
		}
		for _, tt := range faults {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := shape.Read([]byte(tt.give))
				isFault(t, err, shape.ErrShape, tt.wantPath, tt.wantReason)
			})
		}
	})

	t.Run("Invert", func(t *testing.T) {
		t.Parallel()

		t.Run("returns choices that decode to each value that a domain shape generates", func(t *testing.T) {
			t.Parallel()
			for _, text := range domainTrips {
				roundTrips(t, text)
			}
		})

		tests := []struct {
			name  string
			shape string
			give  any
			want  []choice.Choice
		}{
			{
				name:  "returns a 128-bit int of a big.Int",
				shape: `{"shape":"int","width":128,"signed":true}`,
				give:  *big.NewInt(-1),
				want:  []choice.Choice{n(-1), allOnes},
			},
			{
				name:  "returns a 128-bit int of an int",
				shape: `{"shape":"int","width":128,"signed":false}`,
				give:  7,
				want:  []choice.Choice{n(0), n(7)},
			},
			{
				name:  "returns an address of version 6",
				shape: `{"shape":"ip-address"}`,
				give:  make([]byte, 16),
				want:  []choice.Choice{n(1), seq(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := engine.Invert(read(t, tt.shape), tt.give)
				assert.NoError(t, err, "the shape produces the value")
				assert.True(t, slices.EqualFunc(got, tt.want, choice.Choice.Equal), "the choices")
			})
		}

		atSeconds, atUnits := fault.Path{fault.Field("seconds")}, fault.Path{fault.Field("units")}
		atZone := fault.Path{fault.Field("zone")}
		refusals := []struct {
			name       string
			shape      string
			give       any
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns an error for a 128-bit int's value that is no integer",
				shape:      bounded,
				give:       "x",
				wantReason: "x is no integer",
			},
			{
				name:       "returns an error for a nil *big.Int",
				shape:      bounded,
				give:       (*big.Int)(nil),
				wantReason: "<nil> is no integer",
			},
			{
				name:       "returns an error for an integer past 128 bits",
				shape:      bounded,
				give:       power(200, 0),
				wantReason: power(200, 0).String() + " is outside [18446744073709551626, 36893488147419103232]",
			},
			{
				name:       "returns an error for a high half outside the bounds",
				shape:      bounded,
				give:       -1,
				wantReason: "-1 is outside [1, 2]",
			},
			{
				name:       "returns an error for a low half outside the bounds",
				shape:      bounded,
				give:       power(64, 3),
				wantReason: "3 is outside [10, 18446744073709551615]",
			},
			{
				name:       "returns an error for an instant's value that is no record",
				shape:      late,
				give:       3,
				wantReason: "3 is no record of the fields [seconds units]",
			},
			{
				name:       "returns an error at an instant's seconds that are no integer",
				shape:      late,
				give:       record("seconds", "x", "units", 0),
				wantPath:   atSeconds,
				wantReason: "x is no integer",
			},
			{
				name:       "returns an error at an instant's units that are no integer",
				shape:      late,
				give:       record("seconds", 0, "units", "x"),
				wantPath:   atUnits,
				wantReason: "x is no integer",
			},
			{
				name:       "returns an error at an instant's seconds before its minimum",
				shape:      late,
				give:       record("seconds", 0, "units", 0),
				wantPath:   atSeconds,
				wantReason: "0 is outside [1577836800, 253402300799]",
			},
			{
				name:       "returns an error at an instant's units before its minimum",
				shape:      late,
				give:       record("seconds", 1577836800, "units", 0),
				wantPath:   atUnits,
				wantReason: "0 is outside [500, 999]",
			},
			{
				name:       "returns an error at the units of an instant at seconds",
				shape:      `{"shape":"instant","unit":"s"}`,
				give:       record("seconds", 0, "units", 1),
				wantPath:   atUnits,
				wantReason: "1 is not 0, the units of an instant at seconds",
			},
			{
				name:       "returns an error for a local's value of its parts in another order",
				shape:      lateLocal,
				give:       record("time-of-day", 0, "date", 0),
				wantReason: "{[{time-of-day 0} {date 0}]} is no record of the fields [date time-of-day]",
			},
			{
				name:       "returns an error at a local date before its minimum",
				shape:      lateLocal,
				give:       record("date", 0, "time-of-day", 0),
				wantPath:   fault.Path{fault.Field("date")},
				wantReason: "0 is outside [18262, 2932896]",
			},
			{
				name:       "returns an error at a local time of day before its minimum",
				shape:      lateLocal,
				give:       record("date", 18262, "time-of-day", 0),
				wantPath:   fault.Path{fault.Field("time-of-day")},
				wantReason: "0 is outside [500, 86399999]",
			},
			{
				name:       "returns an error for a zoned value that is no record",
				shape:      `{"shape":"zoned-date-time","unit":"s"}`,
				give:       3,
				wantReason: "3 is no record of the fields [instant zone]",
			},
			{
				name:       "returns an error at a zone outside the list",
				shape:      `{"shape":"zoned-date-time","unit":"s"}`,
				give:       record("instant", record("seconds", 0, "units", 0), "zone", "Mars/Olympus"),
				wantPath:   atZone,
				wantReason: "Mars/Olympus is no zone of the list",
			},
			{
				name:       "returns an error at a zone that is no string",
				shape:      `{"shape":"wall-time","unit":"s"}`,
				give:       record("local-date-time", record("date", 0, "time-of-day", 0), "zone", 3),
				wantPath:   atZone,
				wantReason: "3 is no zone of the list",
			},
			{
				name:       "returns an error at the seconds of a zoned instant outside the range",
				shape:      `{"shape":"zoned-date-time","unit":"s"}`,
				give:       record("instant", record("seconds", lastSecond+1, "units", 0), "zone", "UTC"),
				wantPath:   fault.Path{fault.Field("instant"), fault.Field("seconds")},
				wantReason: "253402300800 is outside [-62135596800, 253402300799]",
			},
			{
				name:       "returns an error at the date of a wall time outside the range",
				shape:      `{"shape":"wall-time","unit":"s"}`,
				give:       record("local-date-time", record("date", lastDay+1, "time-of-day", 0), "zone", "UTC"),
				wantPath:   fault.Path{fault.Field("local-date-time"), fault.Field("date")},
				wantReason: "2932897 is outside [-719162, 2932896]",
			},
			{
				name:       "returns an error for an address that is no bytes",
				shape:      `{"shape":"ip-address"}`,
				give:       "127.0.0.1",
				wantReason: "127.0.0.1 is no address of 4 or 16 bytes",
			},
			{
				name:       "returns an error for an address of 5 bytes",
				shape:      `{"shape":"ip-address"}`,
				give:       make([]byte, 5),
				wantReason: "[0 0 0 0 0] is no address of 4 or 16 bytes",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := engine.Invert(read(t, tt.shape), tt.give)
				isFault(t, err, engine.ErrCannotInvert, tt.wantPath, tt.wantReason)
			})
		}
	})
}

// power returns 2^n plus delta.
func power(n uint, delta int64) *big.Int {
	return new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), n), big.NewInt(delta))
}

// local returns the local date and time of the seconds since the epoch.
func local(seconds int64) literal.Record {
	day := seconds / 86_400
	if seconds%86_400 < 0 {
		day--
	}
	return record("date", day, "time-of-day", seconds-day*86_400)
}
