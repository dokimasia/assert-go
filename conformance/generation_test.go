// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The first case of the definition's vector integer/draws-from-a-signed-range.
const (
	// signedGenerator is the integer generator over [-100, 100].
	signedGenerator = `{"gen":"integer","min":-100,"max":100}`
	// firstSigned is the first case that the seed 42 decodes from it.
	firstSigned = `{"choices":[11],"value":{"type":"int","value":11},"rejected":false}`
)

// TestGeneration checks a generation vector and a shapes vector: the cases
// that a seed decodes from a generator or a shape. Written with testing
// rather than with this library, because a verdict is not written with the
// subject.
func TestGeneration(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			kind       conformance.VectorKind
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for a vector that states the first case of the seed",
				kind: conformance.Generation,
				give: generated(signedGenerator, "42", 1, firstSigned),
			},
			{
				name:       "returns a fault at the generator for a generator of an unknown id",
				kind:       conformance.Generation,
				give:       generated(unknownGenerator, "42", 1, firstSigned),
				wantPath:   inVector(fault.Field(generatorAt), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the shape for a shape of no shape of the vocabulary",
				kind:       conformance.Shapes,
				give:       `{"shape":{"shape":"widget"},"seed":"1","count":0,"cases":[]}`,
				wantPath:   inVector(fault.Field(shapeAt), fault.Field(shapeAt)),
				wantReason: `"widget" is no shape of the vocabulary`,
			},
			{
				name:       "returns a fault at the cases for a count other than the number of cases",
				kind:       conformance.Generation,
				give:       generated(signedGenerator, "42", 2, firstSigned),
				wantPath:   inVector(fault.Field(casesAt)),
				wantReason: "the vector states 1 cases of 2",
			},
			{
				name: "returns a fault at the value of the case for a case of another value",
				kind: conformance.Generation,
				give: generated(signedGenerator, "42", 1,
					`{"choices":[11],"value":{"type":"int","value":12},"rejected":false}`),
				wantPath:   inVector(fault.Field(casesAt), fault.Index(0), fault.Field(valueAt)),
				wantReason: `the value is int:11, want {"type":"int","value":12}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, tt.kind, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// generated returns a generation vector of generator that states count
// cases of seed, and the one case first.
func generated(generator, seed string, count int, first string) string {
	return fmt.Sprintf(`{"generator":%s,"seed":%q,"count":%d,"cases":[%s]}`, generator, seed, count, first)
}
