// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestCorpus drives every case the definition states against this
// library. It is what checks meaning rather than membership: the
// completeness gate says an assertion exists, and this says it answers
// what the standard says it should.
func TestCorpus(t *testing.T) {
	t.Parallel()

	byAssertion, err := conformance.Cases()
	assert.NoError(t, err, "the corpus can be read")
	assert.NotEmpty(t, byAssertion, "the corpus states something")

	for id, cases := range byAssertion {
		invoke, registered := conformance.Registry[id]
		assert.True(t, registered, "an invoker is registered for "+string(id))

		for _, tc := range cases {
			t.Run(tc.ID, func(t *testing.T) {
				t.Parallel()

				if why, skipped := tc.SkipReason(); skipped {
					t.Skipf("declared skip: %s", why)
				}

				args, err := tc.Decoded()
				assert.NoError(t, err, "the case's arguments decode")

				r := assert.NewRecorder()
				invoke(r, args, tc.ID)

				assert.NoError(t, tc.Check(r), "the outcome is what the case states")
			})
		}
	}
}
