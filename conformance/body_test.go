// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// TestBody checks the refusal of a behaviour vector's body that names no
// body of the vocabulary, or states a draw or a predicate that the
// vocabulary does not. The definition's vectors run every body of the
// vocabulary.
func TestBody(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns an error for a body that is no JSON object",
				give: `3`,
				want: "parse body",
			},
			{
				name: "returns an error for a body of an unknown kind",
				give: `{"kind":"sleeps"}`,
				want: `"sleeps" names no body`,
			},
			{
				name: "returns an error for a draw of a generator of an unknown id",
				give: `{"draw":` + unknownGenerator + `}`,
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for a label that a predicate of an unknown kind classifies",
				give: `{"draw":` + digitGenerator + `,"classify":{"small":{"kind":"most"}}}`,
				want: `"most" names no predicate`,
			},
			{
				name: "returns an error for a rejection under a predicate of an unknown kind",
				give: `{"draw":` + digitGenerator + `,"rejects-when":{"kind":"least"}}`,
				want: `"least" names no predicate`,
			},
			{
				name: "returns an error for a failure under a predicate of an unknown kind",
				give: `{"draw":` + digitGenerator + `,"fails":[{"identity":"big","when":{"kind":"some"}}]}`,
				want: `"some" names no predicate`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Behaviour, behaving(tt.give, seven, passedDetail)), tt.want)
			})
		}
	})
}
