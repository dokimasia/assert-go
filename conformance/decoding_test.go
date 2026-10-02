// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// The generators and the literal that the tests of every kind build their
// vectors from.
const (
	// digitGenerator is the integer generator over [0, 9].
	digitGenerator = `{"gen":"integer","min":0,"max":9}`
	// rejectingGenerator is a filter that rejects its one value at every
	// attempt, and so rejects the case.
	rejectingGenerator = `{"gen":"filter","of":{"gen":"just","value":{"type":"int","value":1}},` +
		`"keep":{"kind":"never"}}`
	// null is the JSON text of no value.
	null = "null"
)

// TestDecoding checks a decoding vector: the replay of its choices into its
// generator.
func TestDecoding(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns nil for a vector that states the replayed case",
				give: decoded(digitGenerator, `[7]`, `[7]`, `{"type":"int","value":7}`),
			},
			{
				name: "returns an error for a vector that is no JSON object",
				give: `[]`,
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error for a generator of an unknown id",
				give: decoded(`{"gen":"widget"}`, `[]`, `[]`, null),
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for a choice of no stated kind",
				give: decoded(digitGenerator, `[{"integer":7}]`, `[7]`, `{"type":"int","value":7}`),
				want: "is no choice",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Decoding, tt.give), tt.want)
			})
		}
	})
}

// decoded returns a decoding vector of generator that replays choices,
// records recorded, and decodes the typed literal value, or rejects the
// case for a value of null.
func decoded(generator, choices, recorded, value string) string {
	return fmt.Sprintf(`{"generator":%s,"choices":%s,"recorded":%s,"value":%s,"rejected":%t}`,
		generator, choices, recorded, value, value == null)
}
