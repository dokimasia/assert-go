// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"slices"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestSubject checks the table of built behaviours against the
// definition's subjects table, and the shapes of a behaviour that no
// corpus case and no vector observes.
func TestSubject(t *testing.T) {
	t.Parallel()

	t.Run("Subjects", func(t *testing.T) {
		t.Parallel()

		t.Run("builds every kind of the definition's subjects table and no other", func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile("spec/assertions.json")
			assert.NoError(t, err, "the vendored assertion table can be read")
			var doc struct {
				Subjects map[string]json.RawMessage `json:"subjects"`
			}
			assert.NoError(t, json.Unmarshal(raw, &doc), "the vendored assertion table parses")
			assert.NotEmpty(t, doc.Subjects, "the definition states subjects")
			assert.Equal(t, slices.Sorted(maps.Keys(conformance.Subjects)), slices.Sorted(maps.Keys(doc.Subjects)),
				"the kinds that the table builds")
		})

		t.Run("builds a subject with state of its own on each call", func(t *testing.T) {
			t.Parallel()
			first, second := conformance.Subjects["accumulates"](), conformance.Subjects["accumulates"]()
			assert.NoError(t, first.Call(nil), "a call of accumulates succeeds")
			assert.Equal(t, first.Observe(), 1, "the first subject counts its call")
			assert.Equal(t, second.Observe(), 0, "the second subject counts none")
		})

		t.Run("returns a reads-handle that returns success for an absent handle", func(t *testing.T) {
			t.Parallel()
			var absent context.Context
			assert.NoError(
				t,
				conformance.Subjects["reads-handle"]().Ctx(absent, nil),
				"an absent handle gives no reason",
			)
		})

		t.Run("returns a settles-after that fails twice before it passes", func(t *testing.T) {
			t.Parallel()
			seated := conformance.Subjects["settles-after"]().Seated
			for attempt, want := range []bool{true, true, false} {
				r := assert.NewRecorder()
				seated(r)
				assert.Equal(t, r.Failed(), want, "attempt "+strconv.Itoa(attempt+1)+" of the subject")
			}
		})
	})
}
