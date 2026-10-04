// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestDriver checks RunSubject, which calls an assertion with a built
// behaviour. The corpus cases run every pair of an assertion and a
// behaviour that they state.
func TestDriver(t *testing.T) {
	t.Parallel()

	t.Run("RunSubject", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			surface   string
			assertion string
			kind      string
			want      bool
		}{
			{
				name:      "reports true for a behaviour that it drives through an assertion",
				surface:   "check",
				assertion: "throws",
				kind:      "raises",
				want:      true,
			},
			{
				name:      "reports false for a behaviour that this language cannot build",
				surface:   "check",
				assertion: "throws",
				kind:      "sleeps",
			},
			{
				name:      "reports false for an assertion that takes no behaviour",
				surface:   "check",
				assertion: "equal",
				kind:      "raises",
			},
			{
				name:      "reports false for a surface that the standard does not state",
				surface:   "assume",
				assertion: "throws",
				kind:      "raises",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r := assert.NewRecorder()
				if got := conformance.RunSubject(tt.surface, tt.assertion, tt.kind, r, tt.name); got != tt.want {
					t.Fatalf("RunSubject reports %t, want %t", got, tt.want)
				}
				if r.Failed() {
					t.Fatalf("the run reported a failure: %s", r.Message())
				}
			})
		}
	})

	t.Run("SubjectDrivers", func(t *testing.T) {
		t.Parallel()

		t.Run("drives one set of assertions on both surfaces", func(t *testing.T) {
			t.Parallel()
			checks, expects := conformance.SubjectDrivers["check"], conformance.SubjectDrivers["expect"]
			if len(checks) == 0 || len(checks) != len(expects) {
				t.Fatalf("the surfaces drive %d and %d assertions, want one set", len(checks), len(expects))
			}
			for id := range checks {
				if _, ok := expects[id]; !ok {
					t.Fatalf("the recording surface drives no %s", id)
				}
			}
		})
	})
}
