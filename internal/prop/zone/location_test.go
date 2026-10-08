// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package zone_test

import (
	"errors"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/zone"
)

// locationsAllocs is the ceiling of the allocations of Locations of a load
// that allocates nothing: the map of sixteen zones.
const locationsAllocs = 5

// errMissing is the error of a load that lacks a zone.
var errMissing = errors.New("zone_test: no such file")

// utc resolves every zone to UTC, and allocates nothing.
func utc(string) (*time.Location, error) { return time.UTC, nil }

// TestLocation checks the resolution of the zones of the list.
func TestLocation(t *testing.T) {
	t.Parallel()

	t.Run("Locations", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the location of each zone of the list in the platform's database", func(t *testing.T) {
			t.Parallel()
			got, err := zone.Locations(time.LoadLocation)
			assert.NoError(t, err, "the platform's database has every zone")
			assert.Length(t, got, len(zone.List()), "one location for each zone")
			assert.Equal(t, got["Europe/Amsterdam"].String(), "Europe/Amsterdam", "the location of a zone's name")
		})

		t.Run("returns a fault at the first zone that the database lacks", func(t *testing.T) {
			t.Parallel()
			lacking := func(name string) (*time.Location, error) {
				if name == "Pacific/Apia" || name == "Pacific/Chatham" {
					return nil, errMissing
				}
				return time.UTC, nil
			}
			_, err := zone.Locations(lacking)
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, f.Path, fault.Path{fault.Key(firstOf(t, "Pacific/Apia", "Pacific/Chatham"))},
				"the first zone of the list that the database lacks")
			assert.Equal(t, f.Reason, "the time-zone database has no such zone", "the reason")
			assert.ErrorIs(t, err, errMissing, "the cause is the error of the load")
		})
	})
}

// firstOf returns the one of names that comes first in the zone list.
func firstOf(t *testing.T, names ...string) string {
	t.Helper()
	for _, z := range zone.List() {
		for _, name := range names {
			if z.Name == name {
				return name
			}
		}
	}
	t.Fatalf("the list has none of %v", names)
	return ""
}

// TestLocationAllocs checks the allocation ceiling of Locations.
func TestLocationAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _, _ = zone.Locations(utc) }, locationsAllocs, "Locations allocates its map")
}

// BenchmarkLocations measures Locations of a load that allocates nothing.
func BenchmarkLocations(b *testing.B) {
	got, err := zone.Locations(utc)
	c := bench.Start(b).MaxAllocs(locationsAllocs)
	defer c.End()
	for c.Loop() {
		got, err = zone.Locations(utc)
	}
	assert.NoError(b, err, "every zone resolves")
	assert.Length(b, got, 16, "a location for each zone")
}
