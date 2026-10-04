// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/shape"
)

// units is the reason of a unit outside the definition's units.
const units = ` is none of ["ms","ns","s","us"]`

// TestParams checks how the shapes read their parameters: the units of
// the time shapes, counts and sizes, and the bounds of the numbers, dates
// and times, with each parameter that fails to read and where the fault
// is.
func TestParams(t *testing.T) {
	t.Parallel()

	t.Run("UnitsPerSecond", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the units in a second of each unit of the time shapes", func(t *testing.T) {
			t.Parallel()
			got := make(map[string]int64)
			for _, unit := range []string{"s", "ms", "us", "ns"} {
				per, ok := shape.UnitsPerSecond(unit)
				assert.True(t, ok, "a unit of the definition")
				got[unit] = per
			}
			assert.Equal(t, got, map[string]int64{"s": 1, "ms": 1_000, "us": 1_000_000, "ns": 1_000_000_000},
				"the units in a second")
		})

		t.Run("reports false for a unit outside the definition", func(t *testing.T) {
			t.Parallel()
			_, ok := shape.UnitsPerSecond("min")
			assert.False(t, ok, "minutes are no unit")
		})
	})

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		start := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
		values := []struct {
			name  string
			shape string
			give  []choice.Choice
			want  any
		}{
			{
				name:  "returns a date of its bounds",
				shape: `{"shape":"date","min":"1970-01-02","max":"1970-01-31"}`,
				give:  []choice.Choice{n(30)},
				want:  int64(30),
			},
			{
				name:  "returns the target of a date's bounds",
				shape: `{"shape":"date","min":"1970-01-02","max":"1970-01-31"}`,
				give:  []choice.Choice{n(0)},
				want:  int64(1),
			},
			{
				name:  "returns a time of day of its minimum",
				shape: `{"shape":"time-of-day","unit":"s","min":"23:00:00"}`,
				give:  []choice.Choice{n(0)},
				want:  int64(82_800),
			},
			{
				name:  "returns a time of day of a fraction of its unit",
				shape: `{"shape":"time-of-day","unit":"ms","min":"00:00:00.25"}`,
				give:  []choice.Choice{n(0)},
				want:  int64(250),
			},
			{
				name:  "returns an instant's units of the second where a bound stops them",
				shape: `{"shape":"instant","unit":"ms","min":"2020-01-01T00:00:00.5Z"}`,
				give:  []choice.Choice{n(start), n(3)},
				want:  record("seconds", start, "units", int64(500)),
			},
			{
				name:  "returns an instant's units of a later second",
				shape: `{"shape":"instant","unit":"ms","min":"2020-01-01T00:00:00.5Z"}`,
				give:  []choice.Choice{n(start + 1), n(3)},
				want:  record("seconds", start+1, "units", int64(3)),
			},
			{
				name:  "returns a local time of day that a bound stops on its own date",
				shape: `{"shape":"local-date-time","unit":"s","max":"1970-01-02T12:00:00"}`,
				give:  []choice.Choice{n(1), n(86_399)},
				want:  record("date", int64(1), "time-of-day", int64(0)),
			},
			{
				name:  "returns a local time of day on another date",
				shape: `{"shape":"local-date-time","unit":"s","max":"1970-01-02T12:00:00"}`,
				give:  []choice.Choice{n(0), n(86_399)},
				want:  record("date", int64(0), "time-of-day", int64(86_399)),
			},
			{
				name:  "returns a duration of its bounds",
				shape: `{"shape":"duration","unit":"ms","min":-5,"max":5}`,
				give:  []choice.Choice{n(-5)},
				want:  int64(-5),
			},
		}
		for _, tt := range values {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, tt.shape, tt.give...)
				assert.Equal(t, got, tt.want, "the value")
			})
		}

		atMin, atMax := fault.Path{fault.Field("min")}, fault.Path{fault.Field("max")}
		atUnit := fault.Path{fault.Field("unit")}
		faults := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns an error at a width that is no integer",
				give:       `{"shape":"int","width":"x","signed":true}`,
				wantPath:   fault.Path{fault.Field("width")},
				wantReason: `"x" is no integer`,
			},
			{
				name:       "returns an error at a minimum of an int that is no integer",
				give:       `{"shape":"int","width":8,"signed":false,"min":1.5}`,
				wantPath:   atMin,
				wantReason: "1.5 is no integer",
			},
			{
				name:       "returns an error at a scale that is no integer",
				give:       `{"shape":"decimal","scale":"x"}`,
				wantPath:   fault.Path{fault.Field("scale")},
				wantReason: `"x" is no integer`,
			},
			{
				name:       "returns an error at a negative size of a fixed-list",
				give:       `{"shape":"fixed-list","of":{"shape":"bool"},"size":-1}`,
				wantPath:   fault.Path{fault.Field("size")},
				wantReason: "-1 is below 0",
			},
			{
				name:       "returns an error at a negative minimum size",
				give:       `{"shape":"list","of":{"shape":"bool"},"min_size":-1}`,
				wantPath:   fault.Path{fault.Field("min_size")},
				wantReason: "-1 is below 0",
			},
			{
				name:       "returns an error at a maximum size that is no integer",
				give:       `{"shape":"bytes","max_size":"x"}`,
				wantPath:   fault.Path{fault.Field("max_size")},
				wantReason: `"x" is no integer`,
			},
			{
				name:       "returns an error at a size beyond the int64 range",
				give:       `{"shape":"string","max_size":"99999999999999999999"}`,
				wantPath:   fault.Path{fault.Field("max_size")},
				wantReason: "99999999999999999999 is more than an int counts",
			},
			{
				name:       "returns an error at a minimum size of a set that is no integer",
				give:       `{"shape":"set","of":{"shape":"bool"},"min_size":"x"}`,
				wantPath:   fault.Path{fault.Field("min_size")},
				wantReason: `"x" is no integer`,
			},
			{
				name:       "returns an error for list sizes of no length",
				give:       `{"shape":"list","of":{"shape":"bool"},"min_size":3,"max_size":2}`,
				wantReason: "the sizes [3, 2] are empty",
			},
			{
				name:       "returns an error for map sizes of no length",
				give:       `{"shape":"map","key":{"shape":"bool"},"of":{"shape":"bool"},"min_size":2,"max_size":1}`,
				wantReason: "the sizes [2, 1] are empty",
			},
			{
				name:       "returns an error for string sizes of no length",
				give:       `{"shape":"string","min_size":2,"max_size":1}`,
				wantReason: "the sizes [2, 1] are empty",
			},
			{
				name:       "returns an error at an unknown unit of an instant",
				give:       `{"shape":"instant","unit":"min"}`,
				wantPath:   atUnit,
				wantReason: `"min"` + units,
			},
			{
				name:       "returns an error at an unknown unit of a time of day",
				give:       `{"shape":"time-of-day","unit":"min"}`,
				wantPath:   atUnit,
				wantReason: `"min"` + units,
			},
			{
				name:       "returns an error at a unit of a local date and time that is no string",
				give:       `{"shape":"local-date-time","unit":1}`,
				wantPath:   atUnit,
				wantReason: "1" + units,
			},
			{
				name:       "returns an error at an unknown unit of a duration",
				give:       `{"shape":"duration","unit":"h"}`,
				wantPath:   atUnit,
				wantReason: `"h"` + units,
			},
			{
				name:       "returns an error at an unknown unit of a zoned date and time",
				give:       `{"shape":"zoned-date-time","unit":"m"}`,
				wantPath:   atUnit,
				wantReason: `"m"` + units,
			},
			{
				name:       "returns an error at an unknown unit of a wall time",
				give:       `{"shape":"wall-time","unit":"m"}`,
				wantPath:   atUnit,
				wantReason: `"m"` + units,
			},
			{
				name:       "returns an error at an instant bound finer than its unit",
				give:       `{"shape":"instant","unit":"s","min":"2020-01-01T00:00:00.5Z"}`,
				wantPath:   atMin,
				wantReason: "the fraction .5 is finer than the unit",
			},
			{
				name:       "returns an error at an instant bound of no date",
				give:       `{"shape":"instant","unit":"s","min":"2020-02-30T00:00:00Z"}`,
				wantPath:   atMin,
				wantReason: "2020-02-30 is no date",
			},
			{
				name:       "returns an error at an instant bound of no time",
				give:       `{"shape":"instant","unit":"s","max":"2020-01-01T25:00:00Z"}`,
				wantPath:   atMax,
				wantReason: "25:00:00 is no time",
			},
			{
				name:       "returns an error for instant bounds out of order",
				give:       `{"shape":"instant","unit":"s","min":"2020-01-02T00:00:00Z","max":"2020-01-01T00:00:00Z"}`,
				wantReason: "the bounds [1577923200 0] to [1577836800 0] are empty or out of range",
			},
			{
				name:       "returns an error at a date that does not exist",
				give:       `{"shape":"date","min":"2021-02-29"}`,
				wantPath:   atMin,
				wantReason: "2021-02-29 is no date",
			},
			{
				name:       "returns an error at a date of the year 0",
				give:       `{"shape":"date","max":"0000-12-31"}`,
				wantPath:   atMax,
				wantReason: "0000-12-31 is no date",
			},
			{
				name:       "returns an error at a date of month 13",
				give:       `{"shape":"date","max":"2020-13-01"}`,
				wantPath:   atMax,
				wantReason: "2020-13-01 is no date",
			},
			{
				name:       "returns an error at a time of day of hour 24",
				give:       `{"shape":"time-of-day","unit":"s","max":"24:00:00"}`,
				wantPath:   atMax,
				wantReason: "24:00:00 is no time",
			},
			{
				name:       "returns an error at a time of day of minute 60",
				give:       `{"shape":"time-of-day","unit":"s","max":"23:60:00"}`,
				wantPath:   atMax,
				wantReason: "23:60:00 is no time",
			},
			{
				name:       "returns an error at a time of day of second 60",
				give:       `{"shape":"time-of-day","unit":"s","max":"23:59:60"}`,
				wantPath:   atMax,
				wantReason: "23:59:60 is no time",
			},
			{
				name:       "returns an error at a time of day finer than its unit",
				give:       `{"shape":"time-of-day","unit":"ms","min":"00:00:00.0005"}`,
				wantPath:   atMin,
				wantReason: "the fraction .0005 is finer than the unit",
			},
			{
				name:       "returns an error at a local bound of no date",
				give:       `{"shape":"local-date-time","unit":"s","max":"2020-04-31T00:00:00"}`,
				wantPath:   atMax,
				wantReason: "2020-04-31 is no date",
			},
			{
				name:       "returns an error at a duration bound that is no integer",
				give:       `{"shape":"duration","unit":"s","min":"x"}`,
				wantPath:   atMin,
				wantReason: `"x" is no integer`,
			},
			{
				name:       "returns an error at a duration bound beyond the int64 range",
				give:       `{"shape":"duration","unit":"s","max":"99999999999999999999"}`,
				wantPath:   atMax,
				wantReason: `"99999999999999999999" is no integer`,
			},
			{
				name:       "returns an error for an offset beyond 18 hours",
				give:       `{"shape":"offset","max":64801}`,
				wantReason: "the bounds [-64800, 64801] are empty or outside [-64800, 64800]",
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
}

// TestParamsAllocs checks that UnitsPerSecond allocates nothing.
func TestParamsAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _, _ = shape.UnitsPerSecond("us") }, 0, "UnitsPerSecond allocates nothing")
}

// BenchmarkParams measures UnitsPerSecond under a ceiling of no
// allocation.
func BenchmarkParams(b *testing.B) {
	b.Run("UnitsPerSecond", func(b *testing.B) {
		var got int64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = shape.UnitsPerSecond("us")
		}
		assert.Equal(b, got, int64(1_000_000), "the microseconds in a second")
	})
}
