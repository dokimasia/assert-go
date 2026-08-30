// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
)

// This package is a consumer of the library, so its tests are written
// with it. The assertion core cannot do the same: a package that tests
// itself with itself lets one bug hide another.

func TestScrubber(t *testing.T) {
	t.Parallel()

	t.Run("ScrubTimestamps", func(t *testing.T) {
		t.Parallel()

		stamps := []string{
			"2026-08-30T12:00:00Z",
			"2026-08-30T12:00:00.123456Z",
			"2026-08-30T12:00:00+02:00",
			"2026-08-30 12:00:00",
		}
		for _, stamp := range stamps {
			t.Run("replaces "+stamp, func(t *testing.T) {
				t.Parallel()

				assert.NotContains(t, golden.ScrubTimestamps()("at "+stamp+" exactly"), stamp,
					"the timestamp is replaced")
			})
		}

		t.Run("leaves a bare date alone", func(t *testing.T) {
			t.Parallel()

			const in = "released 2026-08-30"
			assert.Equal(t, golden.ScrubTimestamps()(in), in,
				"a date without a time is not a timestamp")
		})
	})

	t.Run("ScrubHashes", func(t *testing.T) {
		t.Parallel()

		t.Run("replaces a hex digest", func(t *testing.T) {
			t.Parallel()

			const digest = "d41d8cd98f00b204e9800998ecf8427e"
			assert.NotContains(t, golden.ScrubHashes()("sum "+digest), digest,
				"the digest is replaced")
		})

		t.Run("leaves a short hex string alone", func(t *testing.T) {
			t.Parallel()

			const in = "colour #ff8800"
			assert.Equal(t, golden.ScrubHashes()(in), in,
				"six hex characters are too few to be a digest")
		})
	})

	t.Run("ScrubRunIDs", func(t *testing.T) {
		t.Parallel()

		t.Run("replaces a run identifier", func(t *testing.T) {
			t.Parallel()

			const id = "run_abcdef0123456789"
			assert.NotContains(t, golden.ScrubRunIDs()("id "+id), id,
				"the run identifier is replaced")
		})
	})

	t.Run("ScrubJSONFields", func(t *testing.T) {
		t.Parallel()

		t.Run("replaces a named field's value", func(t *testing.T) {
			t.Parallel()

			got := golden.ScrubJSONFields("token")(`{"token":"secret","name":"kept"}`)

			assert.NotContains(t, got, "secret", "the named field's value is replaced")
			assert.Contains(t, got, "kept", "a field nobody named is left alone")
		})

		t.Run("replaces several named fields", func(t *testing.T) {
			t.Parallel()

			got := golden.ScrubJSONFields("a", "b")(`{"a":"one","b":"two","c":"three"}`)

			assert.NotContains(t, got, "one", "the first named field is replaced")
			assert.NotContains(t, got, "two", "the second named field is replaced")
			assert.Contains(t, got, "three", "a field nobody named is left alone")
		})

		t.Run("naming no field changes nothing", func(t *testing.T) {
			t.Parallel()

			const in = `{"a":"one"}`
			assert.Equal(t, golden.ScrubJSONFields()(in), in,
				"a scrubber naming no field is the identity")
		})
	})
}
