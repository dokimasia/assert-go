// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// This package is a consumer of the library, so its tests are written
// with it. The assertion core cannot do the same: a package that tests
// itself with itself lets one bug hide another.

// abortingOnly names members the recording surface is not expected to
// carry, with the reason. An entry here is a claim someone can argue
// with, which a silent omission is not.
var abortingOnly = map[string]string{
	"Rejects":     "drives a check to failure, which needs a seat that stops",
	"Recorder":    "the seat both surfaces report through, declared once",
	"NewRecorder": "the seat both surfaces report through, declared once",
	"TB":          "the seat interface, declared once and used by both",
}

func TestSurface(t *testing.T) {
	t.Parallel()

	aborting, err := conformance.Members(conformance.Aborting)
	assert.NoError(t, err, "the aborting surface can be read")

	recording, err := conformance.Members(conformance.Recording)
	assert.NoError(t, err, "the recording surface can be read")

	t.Run("Members", func(t *testing.T) {
		t.Parallel()

		t.Run("both surfaces declare something", func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, aborting, "the aborting surface declares members")
			assert.NotEmpty(t, recording, "the recording surface declares members")
		})

		t.Run("every aborting member has a recording twin", func(t *testing.T) {
			t.Parallel()

			for _, name := range aborting {
				if _, excused := abortingOnly[name]; excused {
					continue
				}
				assert.Contains(t, recording, name,
					"the recording surface carries "+name)
			}
		})

		t.Run("the recording surface adds nothing of its own", func(t *testing.T) {
			t.Parallel()

			for _, name := range recording {
				assert.Contains(t, aborting, name,
					"the aborting surface carries "+name)
			}
		})

		t.Run("every excused member is absent as claimed", func(t *testing.T) {
			t.Parallel()

			for name, why := range abortingOnly {
				assert.Contains(t, aborting, name,
					"the aborting surface declares "+name+", excused because it "+why)
				assert.NotContains(t, recording, name,
					"the recording surface omits "+name+", excused because it "+why)
			}
		})
	})
}
