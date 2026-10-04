// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package zone_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/zone"
)

// manifest is the vendored definition's manifest, which states the digest of
// every file the definition vendors.
var manifest = filepath.Join("..", "..", "..", "conformance", "spec", "manifest.json")

// TestZone checks the zone list that the embedded table states.
func TestZone(t *testing.T) {
	t.Parallel()

	t.Run("List", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the sixteen zones with UTC first and unchanged", func(t *testing.T) {
			t.Parallel()
			zones := zone.List()
			assert.Length(t, zones, 16, "the definition lists sixteen zones")
			assert.Equal(t, zones[0].Name, "UTC", "UTC comes first, so a zone shrinks to it")
			assert.Empty(t, zones[0].Changes, "UTC changes no offset")
		})

		t.Run("returns 2,588 offset changes", func(t *testing.T) {
			t.Parallel()
			total := 0
			for _, z := range zone.List() {
				total += len(z.Changes)
			}
			assert.Equal(t, total, 2588, "the table states every change from 1900 until 2100")
		})

		t.Run("returns a change's instant and the offsets before and after it", func(t *testing.T) {
			t.Parallel()
			amsterdam := zone.List()[1]
			assert.Equal(t, amsterdam.Name, "Europe/Amsterdam", "Amsterdam is the second zone")
			assert.Equal(t, amsterdam.Changes[0], zone.Change{At: -1740355200, Before: 0, After: 3600},
				"the first change moves from UTC to an hour east")
		})

		t.Run("returns each zone's changes in time order, each from the offset before it", func(t *testing.T) {
			t.Parallel()
			for _, z := range zone.List() {
				for i := 1; i < len(z.Changes); i++ {
					assert.True(t, z.Changes[i].At > z.Changes[i-1].At, z.Name+" changes in time order")
					assert.Equal(t, z.Changes[i].Before, z.Changes[i-1].After,
						z.Name+" changes from the offset that the change before left")
				}
			}
		})

		t.Run("returns the same list on every call", func(t *testing.T) {
			t.Parallel()
			first, second := zone.List(), zone.List()
			assert.True(t, &first[0] == &second[0], "the table is parsed once")
		})
	})
}

// TestZoneTable checks that the embedded table is the vendored definition's,
// by the digest that the vendored manifest states for it.
func TestZoneTable(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("zones.json")
	assert.NoError(t, err, "the embedded table can be read")
	raw, err := os.ReadFile(manifest)
	assert.NoError(t, err, "the vendored manifest can be read")
	var doc struct {
		Files map[string]string `json:"files"`
	}
	assert.NoError(t, json.Unmarshal(raw, &doc), "the manifest parses")
	sum := sha256.Sum256(data)
	assert.Equal(t, "sha256:"+hex.EncodeToString(sum[:]), doc.Files["spec/zones.json"],
		"make spec-sync copied the table of the vendored definition")
}

// BenchmarkList measures List once the table is parsed.
func BenchmarkList(b *testing.B) {
	got := zone.List()
	c := bench.Start(b).MaxAllocs(0)
	defer c.End()
	for c.Loop() {
		got = zone.List()
	}
	assert.Length(b, got, 16, "the list")
}
