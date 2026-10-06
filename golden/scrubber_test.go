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

		tests := []struct {
			name       string
			giveFields []string
			give       string
			want       string
		}{
			{
				name:       "replaces a named field's string, and leaves a field nobody named",
				giveFields: []string{"token"},
				give:       `{"token":"secret","name":"kept"}`,
				want:       `{"token":"SCRUBBED","name":"kept"}`,
			},
			{
				name:       "replaces a string with escaped quotes whole",
				giveFields: []string{"token"},
				give:       `{"token":"a \"quoted\" \\ word","name":"kept"}`,
				want:       `{"token":"SCRUBBED","name":"kept"}`,
			},
			{
				name:       "replaces a value of each scalar type",
				giveFields: []string{"name", "seconds", "bytes", "done", "failed", "peak"},
				give: `{"name": "fixture", "seconds": 0.25, "bytes": 512, "done": true, "failed": false, ` +
					`"peak": null}`,
				want: `{"name": "SCRUBBED", "seconds": "SCRUBBED", "bytes": "SCRUBBED", "done": "SCRUBBED", ` +
					`"failed": "SCRUBBED", "peak": "SCRUBBED"}`,
			},
			{
				name:       "replaces a negative number with a fraction and an exponent",
				giveFields: []string{"drift"},
				give:       `{"drift":-12.5e-3,"kept":1}`,
				want:       `{"drift":"SCRUBBED","kept":1}`,
			},
			{
				name:       "leaves an object, an array and a word that only starts as a literal",
				giveFields: []string{"object", "list", "word"},
				give:       `{"object":{"a":1},"list":[1,2],"word":nullish}`,
				want:       `{"object":{"a":1},"list":[1,2],"word":nullish}`,
			},
			{
				name:       "returns the input for no named field",
				giveFields: nil,
				give:       `{"a":"one"}`,
				want:       `{"a":"one"}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, golden.ScrubJSONFields(tt.giveFields...)(tt.give), tt.want, "the scrubbed text")
			})
		}
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
		{Name: "ScrubJSONFields", Call: func(assert.TB) { scrubbing = golden.ScrubJSONFields("token") }, Allocs: 113},
		{Name: "ScrubJSONFields of no field", Call: func(assert.TB) { scrubbing = golden.ScrubJSONFields() }},
	}
}
