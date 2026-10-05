// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// TestDrawn checks which values are relevances, and the JSON of a draw of a
// counterexample.
func TestDrawn(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Relevance
			want bool
		}{
			{name: "reports true for Untested", give: prop.Untested, want: true},
			{name: "reports true for ValueMatters", give: prop.ValueMatters, want: true},
			{name: "reports false past ValueMatters", give: invalidRelevance, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a relevance")
			})
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Drawn
			want string
		}{
			{
				name: "returns a draw where any value fails",
				give: prop.Drawn{Label: drawn, Value: 0, Relevance: prop.AnyValueFails},
				want: `{"label":"value","value":{"type":"int","value":0},"any-value-fails":true,"nearest-passing":null}`,
			},
			{
				name: "returns the nearest passing value of a draw whose value matters",
				give: prop.Drawn{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000},
				want: `{"label":"value","value":{"type":"int","value":1001},"any-value-fails":false,` +
					`"nearest-passing":{"type":"int","value":1000}}`,
			},
			{
				name: "returns null for whether any value fails of an untested draw",
				give: prop.Drawn{Label: drawn, Value: 7},
				want: `{"label":"value","value":{"type":"int","value":7},"any-value-fails":null,"nearest-passing":null}`,
			},
			{
				name: "returns the opaque literal of a value that no typed literal states",
				give: prop.Drawn{Label: drawn, Value: complex(1, 2)},
				want: `{"label":"value","value":{"type":"opaque","text":"(1+2i)"},"any-value-fails":null,"nearest-passing":null}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := json.Marshal(tt.give)
				assert.NoError(t, err, "the draw is JSON")
				assert.Equal(t, string(got), tt.want, "the draw as the record of a run states it")
			})
		}
	})
}

// drawnJSONAllocs are the allocations of MarshalJSON on a draw of an
// integer whose value matters, measured.
const drawnJSONAllocs = 20

// TestDrawnAllocs checks that Valid allocates nothing, and the ceiling
// of MarshalJSON.
func TestDrawnAllocs(t *testing.T) {
	d := prop.Drawn{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000}
	assert.MaxAllocs(t, func() { _ = prop.ValueMatters.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = d.MarshalJSON() }, drawnJSONAllocs, "MarshalJSON allocates its JSON")
}

// BenchmarkDrawn measures Valid under a ceiling of no allocation, and
// MarshalJSON.
func BenchmarkDrawn(b *testing.B) {
	b.Run("MarshalJSON", func(b *testing.B) {
		d := prop.Drawn{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000}
		got, _ := d.MarshalJSON()
		c := bench.Start(b).MaxAllocs(drawnJSONAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = d.MarshalJSON()
		}
		assert.Contains(b, string(got), `"nearest-passing":{"type":"int","value":1000}`, "the nearest passing value")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.ValueMatters.Valid()
		}
		assert.True(b, got, "ValueMatters is a relevance")
	})
}
