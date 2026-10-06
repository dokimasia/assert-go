// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/alloctest"
)

// scrubbing keeps the scrubber that a case returns, so the compiler keeps
// the call.
var scrubbing golden.Scrubber

// TestScrubber checks what each scrubber replaces and what it leaves
// alone. The package uses this module's assertions, as every package
// outside internal/matcher does.
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

			assert.Equal(t, golden.ScrubRunIDs()("id run_abcdef0123456789"), "id SCRUBBED_RUN_ID",
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

		t.Run("returns the input for no named field", func(t *testing.T) {
			t.Parallel()

			const in = `{"a":"one"}`
			assert.Equal(t, golden.ScrubJSONFields()(in), in,
				"a scrubber naming no field is the identity")
		})
	})
}

// TestScrubberAllocs checks the allocation ceiling of each constructor of
// a scrubber.
func TestScrubberAllocs(t *testing.T) {
	alloctest.Check(t, scrubberCases())
}

// BenchmarkScrubber measures each constructor of a scrubber.
func BenchmarkScrubber(b *testing.B) {
	for _, c := range scrubberCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// scrubberCases returns a call of each constructor of a scrubber, with its
// allocation ceiling, measured.
func scrubberCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "ScrubTimestamps", Call: func(assert.TB) { scrubbing = golden.ScrubTimestamps() }},
		{Name: "ScrubHashes", Call: func(assert.TB) { scrubbing = golden.ScrubHashes() }},
		{Name: "ScrubRunIDs", Call: func(assert.TB) { scrubbing = golden.ScrubRunIDs() }},
		{Name: "ScrubJSONFields", Call: func(assert.TB) { scrubbing = golden.ScrubJSONFields("token") }, Allocs: 46},
		{Name: "ScrubJSONFields of no field", Call: func(assert.TB) { scrubbing = golden.ScrubJSONFields() }},
	}
}
