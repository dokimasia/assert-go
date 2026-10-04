// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	bignum "math/big"
	"net/netip"
	"testing"
	"time"
	"uuid"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// TestConvert checks the Go values of the shapes of a date, a time, a
// zone, an address, a decimal and a char: what each decodes to, what runs
// back, and each value that no neutral value states.
func TestConvert(t *testing.T) {
	t.Parallel()

	t.Run("OfShape", func(t *testing.T) {
		t.Parallel()

		decodes := []struct {
			name  string
			shape string
			give  []choice.Choice
			want  any
		}{
			{
				name:  "decodes a char to a rune",
				shape: `{"shape":"char"}`,
				give:  []choice.Choice{sequence(10)},
				want:  'a',
			},
			{
				name:  "decodes a uuid to a uuid.UUID",
				shape: `{"shape":"uuid"}`,
				give:  []choice.Choice{sequence(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15)},
				want:  uuid.UUID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			},
			{
				name:  "decodes an ip-address of version 4 to a netip.Addr",
				shape: `{"shape":"ip-address","version":4}`,
				give:  []choice.Choice{sequence(127, 0, 0, 1)},
				want:  netip.MustParseAddr("127.0.0.1"),
			},
			{
				name:  "decodes an ip-address of either version to a netip.Addr",
				shape: `{"shape":"ip-address"}`,
				give:  []choice.Choice{integer(1), sequence(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1)},
				want:  netip.MustParseAddr("::1"),
			},
			{
				name:  "decodes a decimal to a *big.Rat of its scale",
				shape: `{"shape":"decimal","scale":2}`,
				give:  []choice.Choice{signed(150)},
				want:  bignum.NewRat(3, 2),
			},
			{
				name:  "decodes an instant to a time.Time in UTC",
				shape: `{"shape":"instant","unit":"ms"}`,
				give:  []choice.Choice{signed(1), signed(500)},
				want:  time.Unix(1, 500_000_000).UTC(),
			},
			{
				name:  "decodes a date to a time.Time at midnight in UTC",
				shape: `{"shape":"date"}`,
				give:  []choice.Choice{signed(1)},
				want:  time.Date(1970, time.January, 2, 0, 0, 0, 0, time.UTC),
			},
			{
				name:  "decodes a time of day to the time.Duration since midnight",
				shape: `{"shape":"time-of-day","unit":"s"}`,
				give:  []choice.Choice{signed(3600)},
				want:  time.Hour,
			},
			{
				name:  "decodes a local date and time to a time.Time in UTC",
				shape: `{"shape":"local-date-time","unit":"s"}`,
				give:  []choice.Choice{signed(1), signed(3600)},
				want:  time.Date(1970, time.January, 2, 1, 0, 0, 0, time.UTC),
			},
			{
				name:  "decodes a local date and time before 1970 by its days and its time of day",
				shape: `{"shape":"local-date-time","unit":"s"}`,
				give:  []choice.Choice{signed(-1), signed(60)},
				want:  time.Date(1969, time.December, 31, 0, 1, 0, 0, time.UTC),
			},
			{
				name:  "decodes a duration to a time.Duration",
				shape: `{"shape":"duration","unit":"ms"}`,
				give:  []choice.Choice{signed(5)},
				want:  5 * time.Millisecond,
			},
			{
				name:  "decodes an offset to an int of seconds",
				shape: `{"shape":"offset"}`,
				give:  []choice.Choice{signed(3600)},
				want:  3600,
			},
			{
				name:  "decodes a zone to its *time.Location",
				shape: `{"shape":"zone"}`,
				give:  []choice.Choice{integer(0)},
				want:  time.UTC,
			},
			{
				name:  "decodes a zoned date and time to a time.Time in its zone",
				shape: `{"shape":"zoned-date-time","unit":"s"}`,
				give:  []choice.Choice{integer(0), signed(60)},
				want:  time.Unix(60, 0).UTC(),
			},
		}
		for _, tt := range decodes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g, err := prop.OfShape(tt.shape)
				assert.NoError(t, err, "the shape reads")
				assert.Equal(t, decodedBy(g, tt.give...), tt.want, "the Go value of the shape")
			})
		}

		runs := []struct {
			name  string
			shape string
			give  any
		}{
			{name: "runs a char back from its string", shape: `{"shape":"char"}`, give: "a"},
			{name: "runs a uuid back from its bytes", shape: `{"shape":"uuid"}`, give: make([]byte, 16)},
			{
				name:  "runs an ip-address back from its bytes",
				shape: `{"shape":"ip-address"}`,
				give:  []byte{10, 0, 0, 1},
			},
			{
				name:  "runs an ip-address of version 6 back from its value",
				shape: `{"shape":"ip-address"}`,
				give:  netip.MustParseAddr("::1"),
			},
			{name: "runs a decimal back from its unscaled integer", shape: `{"shape":"decimal","scale":2}`, give: 150},
			{name: "runs a date back from its days", shape: `{"shape":"date"}`, give: 1},
			{name: "runs a time of day back from its units", shape: `{"shape":"time-of-day","unit":"s"}`, give: 60},
			{name: "runs a duration back from its units", shape: `{"shape":"duration","unit":"s"}`, give: -60},
			{name: "runs a zone back from its name", shape: `{"shape":"zone"}`, give: "UTC"},
			{
				name:  "runs an instant back from the record of its parts",
				shape: `{"shape":"instant","unit":"s"}`,
				give:  literal.Record{Fields: []literal.Field{{Name: "seconds", Value: 60}, {Name: "units", Value: 0}}},
			},
			{
				name:  "runs a local date and time before 1970 back from its value",
				shape: `{"shape":"local-date-time","unit":"s"}`,
				give:  time.Date(1969, time.December, 31, 0, 1, 0, 0, time.UTC),
			},
			{
				name:  "runs a local date and time back from the record of its parts",
				shape: `{"shape":"local-date-time","unit":"s"}`,
				give: literal.Record{
					Fields: []literal.Field{{Name: "date", Value: 1}, {Name: "time-of-day", Value: 60}},
				},
			},
			{
				name:  "runs a zoned date and time back from the record of its parts",
				shape: `{"shape":"zoned-date-time","unit":"s"}`,
				give: literal.Record{Fields: []literal.Field{
					{Name: "instant", Value: literal.Record{Fields: []literal.Field{
						{Name: "seconds", Value: 60}, {Name: "units", Value: 0},
					}}},
					{Name: "zone", Value: "UTC"},
				}},
			},
			{
				name:  "runs a wall time back from the record of its parts",
				shape: `{"shape":"wall-time","unit":"s"}`,
				give: literal.Record{Fields: []literal.Field{
					{Name: "local-date-time", Value: literal.Record{Fields: []literal.Field{
						{Name: "date", Value: 1}, {Name: "time-of-day", Value: 60},
					}}},
					{Name: "zone", Value: "UTC"},
				}},
			},
		}
		for _, tt := range runs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g, err := prop.OfShape(tt.shape)
				assert.NoError(t, err, "the shape reads")
				choices, err := engine.Invert(engine.Generator[any](g), tt.give)
				assert.NoError(t, err, "the value runs back to its choices")
				assert.NotNil(t, choices, "the choices of the value")
			})
		}

		refusals := []struct {
			name       string
			shape      string
			give       any
			wantReason string
		}{
			{
				name:       "refuses a rune that is no character",
				shape:      `{"shape":"char"}`,
				give:       rune(0xD800),
				wantReason: "U+D800 is no character",
			},
			{
				name:       "refuses a nil zone",
				shape:      `{"shape":"zone"}`,
				give:       (*time.Location)(nil),
				wantReason: "a nil *time.Location is no zone",
			},
			{
				name:       "refuses the zero netip.Addr",
				shape:      `{"shape":"ip-address"}`,
				give:       netip.Addr{},
				wantReason: "the zero netip.Addr is no address",
			},
			{
				name:       "refuses an address with a zone",
				shape:      `{"shape":"ip-address"}`,
				give:       netip.MustParseAddr("fe80::1%eth0"),
				wantReason: `fe80::1%eth0 states the zone "eth0", which an ip-address shape has no part for`,
			},
			{
				name:       "refuses a nil decimal",
				shape:      `{"shape":"decimal","scale":1}`,
				give:       (*bignum.Rat)(nil),
				wantReason: "a nil *big.Rat is no decimal",
			},
			{
				name:       "refuses a decimal of more digits than its scale",
				shape:      `{"shape":"decimal","scale":1}`,
				give:       bignum.NewRat(1, 3),
				wantReason: "1/3 has more digits than the scale of 1",
			},
			{
				name:       "refuses a decimal outside the int64 range at its scale",
				shape:      `{"shape":"decimal","scale":2}`,
				give:       new(bignum.Rat).SetInt(new(bignum.Int).Lsh(bignum.NewInt(1), 62)),
				wantReason: "4611686018427387904 is outside the int64 range at the scale of 2",
			},
			{
				name:       "refuses an instant finer than its unit",
				shape:      `{"shape":"instant","unit":"s"}`,
				give:       time.Unix(0, 1).UTC(),
				wantReason: "1970-01-01 00:00:00.000000001 +0000 UTC is finer than the shape's unit of 1s",
			},
			{
				name:       "refuses a date that is no midnight",
				shape:      `{"shape":"date"}`,
				give:       time.Unix(1, 0).UTC(),
				wantReason: "1970-01-01 00:00:01 +0000 UTC is no midnight in UTC, which a date is",
			},
			{
				name:       "refuses a date at midnight in another zone",
				shape:      `{"shape":"date"}`,
				give:       time.Date(2020, 1, 1, 0, 0, 0, 0, time.FixedZone("X", 0)),
				wantReason: "2020-01-01 00:00:00 +0000 X is no midnight in UTC, which a date is",
			},
			{
				name:       "refuses a time of day of a day or more",
				shape:      `{"shape":"time-of-day","unit":"s"}`,
				give:       24 * time.Hour,
				wantReason: "24h0m0s is outside [0, 24h), where a time of day is",
			},
			{
				name:       "refuses a time of day before midnight",
				shape:      `{"shape":"time-of-day","unit":"s"}`,
				give:       -time.Second,
				wantReason: "-1s is outside [0, 24h), where a time of day is",
			},
			{
				name:       "refuses a local date and time outside UTC",
				shape:      `{"shape":"local-date-time","unit":"s"}`,
				give:       time.Unix(0, 0).In(time.FixedZone("X", 60)),
				wantReason: "1970-01-01 00:01:00 +0001 X is not in UTC, whose fields state a local date and time",
			},
			{
				name:       "refuses a local date and time finer than its unit",
				shape:      `{"shape":"local-date-time","unit":"s"}`,
				give:       time.Unix(0, 5).UTC(),
				wantReason: "1970-01-01 00:00:00.000000005 +0000 UTC is finer than the shape's unit of 1s",
			},
			{
				name:       "refuses a duration finer than its unit",
				shape:      `{"shape":"duration","unit":"s"}`,
				give:       time.Millisecond,
				wantReason: "1ms is finer than the shape's unit of 1s",
			},
			{
				name:       "refuses a zoned date and time finer than its unit",
				shape:      `{"shape":"zoned-date-time","unit":"s"}`,
				give:       time.Unix(0, 1).UTC(),
				wantReason: "1970-01-01 00:00:00.000000001 +0000 UTC is finer than the shape's unit of 1s",
			},
			{
				name:       "refuses a wall time without a zone",
				shape:      `{"shape":"wall-time","unit":"s"}`,
				give:       prop.WallTime{Local: time.Unix(0, 0).UTC()},
				wantReason: "a nil *time.Location is no zone",
			},
			{
				name:  "refuses a wall time outside UTC",
				shape: `{"shape":"wall-time","unit":"s"}`,
				give: prop.WallTime{
					Local: time.Date(2020, 1, 1, 0, 0, 0, 0, time.FixedZone("X", 60)),
					Zone:  time.UTC,
				},
				wantReason: "2020-01-01 00:00:00 +0001 X is not in UTC, whose fields state a local date and time",
			},
			{
				name:       "refuses a wall time finer than its unit",
				shape:      `{"shape":"wall-time","unit":"s"}`,
				give:       prop.WallTime{Local: time.Unix(0, 5).UTC(), Zone: time.UTC},
				wantReason: "1970-01-01 00:00:00.000000005 +0000 UTC is finer than the shape's unit of 1s",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, shapeRefusal(t, tt.shape, tt.give), fault.Error{
					Kind: engine.ErrCannotInvert, Reason: tt.wantReason,
				})
			})
		}
	})

	t.Run("Of", func(t *testing.T) {
		t.Parallel()

		refusals := []struct {
			name       string
			give       func() error
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "refuses a zoned date and time in a zone outside the list",
				give: func() error {
					return refusal(struct {
						V time.Time `prop:"zoned-date-time"`
					}{V: time.Unix(0, 0).In(time.FixedZone("Nowhere", 0))})
				},
				wantPath:   fault.Path{fault.Field("V"), fault.Field("zone")},
				wantReason: "Nowhere is no zone of the list",
			},
			{
				name: "refuses an address of the other version",
				give: func() error {
					return refusal(struct {
						V netip.Addr `prop:"version=4"`
					}{V: netip.MustParseAddr("::1")})
				},
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "16 elements are outside the sizes of the sequence",
			},
			{
				name: "refuses an instant finer than its unit, and states its field",
				give: func() error {
					return refusal(struct {
						V time.Time `prop:"unit=ms"`
					}{V: time.Unix(0, 5).UTC()})
				},
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "1970-01-01 00:00:00.000000005 +0000 UTC is finer than the shape's unit of 1ms",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, tt.give(), fault.Error{
					Path: tt.wantPath, Kind: engine.ErrCannotInvert, Reason: tt.wantReason,
				})
			})
		}
	})
}
