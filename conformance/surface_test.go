// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert/conformance"
)

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
	if err != nil {
		t.Fatalf("read the aborting surface: %v", err)
	}
	recording, err := conformance.Members(conformance.Recording)
	if err != nil {
		t.Fatalf("read the recording surface: %v", err)
	}

	t.Run("Members", func(t *testing.T) {
		t.Parallel()

		t.Run("both surfaces declare something", func(t *testing.T) {
			t.Parallel()

			if len(aborting) == 0 || len(recording) == 0 {
				t.Fatalf("read %d and %d members; this test would pass having checked nothing",
					len(aborting), len(recording))
			}
		})

		t.Run("every aborting member has a recording twin", func(t *testing.T) {
			t.Parallel()

			for _, name := range aborting {
				if _, excused := abortingOnly[name]; excused {
					continue
				}
				if !slices.Contains(recording, name) {
					t.Errorf("%s is missing from the recording surface", name)
				}
			}
		})

		t.Run("the recording surface adds nothing of its own", func(t *testing.T) {
			t.Parallel()

			for _, name := range recording {
				if !slices.Contains(aborting, name) {
					t.Errorf("%s is in the recording surface but not the aborting one", name)
				}
			}
		})

		t.Run("every excused member is absent as claimed", func(t *testing.T) {
			t.Parallel()

			for name, why := range abortingOnly {
				if !slices.Contains(aborting, name) {
					t.Errorf("%s is excused (%s) but the aborting surface does not declare it",
						name, why)
				}
				if slices.Contains(recording, name) {
					t.Errorf("%s is excused (%s) but the recording surface declares it anyway",
						name, why)
				}
			}
		})
	})
}
