// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// The first case of the definition's vector integer/draws-from-a-signed-range.
const (
	// signedGenerator is the integer generator over [-100, 100].
	signedGenerator = `{"gen":"integer","min":-100,"max":100}`
	// firstSigned is the first case that the seed 42 decodes from it.
	firstSigned = `{"choices":[11],"value":{"type":"int","value":11},"rejected":false}`
)

// TestGeneration checks a generation vector: the cases that a seed decodes
// from its generator.
func TestGeneration(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns nil for a vector that states the first case of the seed",
				give: generated(signedGenerator, "42", 1, firstSigned),
			},
			{
				name: "returns an error for a vector that is no JSON object",
				give: `[]`,
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error for a generator of an unknown id",
				give: generated(unknownGenerator, "42", 1, firstSigned),
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for a seed that is no decimal number",
				give: generated(signedGenerator, "forty-two", 1, firstSigned),
				want: "invalid syntax",
			},
			{
				name: "returns an error for a count other than the number of cases",
				give: generated(signedGenerator, "42", 2, firstSigned),
				want: "the vector states 1 cases of 2",
			},
			{
				name: "returns an error for a case of another value",
				give: generated(signedGenerator, "42", 1,
					`{"choices":[11],"value":{"type":"int","value":12},"rejected":false}`),
				want: "case 0: the value is int:11",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Generation, tt.give), tt.want)
			})
		}
	})
}

// generated returns a generation vector of generator that states count
// cases of seed, and the one case first.
func generated(generator, seed string, count int, first string) string {
	return fmt.Sprintf(`{"generator":%s,"seed":%q,"count":%d,"cases":[%s]}`, generator, seed, count, first)
}
