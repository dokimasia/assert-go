// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// TestBody checks the refusal of a behaviour vector's body that names no
// body of the vocabulary, or states a draw or a predicate that the
// vocabulary does not. The definition's vectors run every body of the
// vocabulary. Written with testing rather than with this library, because
// a verdict is not written with the subject.
func TestBody(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		at := func(segs ...fault.Segment) fault.Path {
			return inVector(append([]fault.Segment{fault.Field("body")}, segs...)...)
		}
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault for a body that is no JSON object",
				give:       `3`,
				wantPath:   at(),
				wantReason: "the body does not parse",
			},
			{
				name:       "returns a fault at the kind for a body of an unknown kind",
				give:       `{"kind":"sleeps"}`,
				wantPath:   at(fault.Field(kindAt)),
				wantReason: `"sleeps" names no body`,
			},
			{
				name:       "returns a fault at the draw for a draw of a generator of an unknown id",
				give:       `{"draw":` + unknownGenerator + `}`,
				wantPath:   at(fault.Field("draw"), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the label for a label that a predicate of an unknown kind classifies",
				give:       `{"draw":` + digitGenerator + `,"classify":{"small":{"kind":"most"}}}`,
				wantPath:   at(fault.Field("classify"), fault.Key("small"), fault.Field(kindAt)),
				wantReason: `"most" names no predicate`,
			},
			{
				name:       "returns a fault at rejects-when for a rejection under a predicate of an unknown kind",
				give:       `{"draw":` + digitGenerator + `,"rejects-when":{"kind":"least"}}`,
				wantPath:   at(fault.Field("rejects-when"), fault.Field(kindAt)),
				wantReason: `"least" names no predicate`,
			},
			{
				name: "returns a fault at the failure for a failure under a predicate of an unknown kind",
				give: `{"draw":` + digitGenerator + `,"fails":[{"identity":"big","when":{"kind":"never"}},` +
					`{"identity":"odd","when":{"kind":"some"}}]}`,
				wantPath:   at(fault.Field("fails"), fault.Index(1), fault.Field("when"), fault.Field(kindAt)),
				wantReason: `"some" names no predicate`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := check(t, conformance.Behaviour, behaving(tt.give, seven, passedDetail))
				expectFault(t, err, tt.wantPath, tt.wantReason)
			})
		}
	})
}
