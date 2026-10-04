// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// TestDecoding checks a decoding vector: the replay of its choices into its
// generator. Written with testing rather than with this library, because a
// verdict is not written with the subject.
func TestDecoding(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for a vector that states the replayed case",
				give: decoded(digitGenerator, `[7]`, `[7]`, `{"type":"int","value":7}`),
			},
			{
				name:       "returns a fault at the generator for a generator of an unknown id",
				give:       decoded(unknownGenerator, `[]`, `[]`, null),
				wantPath:   inVector(fault.Field(generatorAt), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the choice for a choice of no stated kind",
				give:       decoded(digitGenerator, `[7,{"integer":7}]`, `[7]`, `{"type":"int","value":7}`),
				wantPath:   inVector(fault.Field(choicesAt), fault.Index(1)),
				wantReason: `{"integer":7} is no choice`,
			},
			{
				name:       "returns a fault at recorded for recorded choices other than the replay's",
				give:       decoded(digitGenerator, `[7]`, `[6]`, `{"type":"int","value":7}`),
				wantPath:   inVector(fault.Field(recordedAt)),
				wantReason: "the case records prop1:AAc, want [6]",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Decoding, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}
