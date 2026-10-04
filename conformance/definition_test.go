// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestDefinition checks the readers of the vendored definition, and the
// rules of an overlay.
func TestDefinition(t *testing.T) {
	t.Parallel()

	assertions := conformance.Assertions()
	names := conformance.Names()
	overlay := conformance.Overlay()

	t.Run("Assertions", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a summary and an arity for every assertion", func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, assertions, "the assertion table states something")
			for id, a := range assertions {
				assert.NotEmpty(t, a.Summary, "assertion "+string(id)+" states what it means")
				assert.True(t, a.Arity > 0, "assertion "+string(id)+" states its arity")
			}
		})

		t.Run("returns the detail fields of an assertion in the definition's order", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, assertions["equal"].DetailFields, []string{"want", "got"}, "the fields of equal")
		})
	})

	t.Run("Names", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a Go name for every assertion of the definition", func(t *testing.T) {
			t.Parallel()

			for id := range assertions {
				assert.Contains(t, names, id, "the naming table gives a Go name for "+string(id))
			}
		})

		t.Run("returns no name for an assertion outside the definition", func(t *testing.T) {
			t.Parallel()

			for id := range names {
				assert.Contains(t, assertions, id, "the assertion table declares "+string(id))
			}
		})

		t.Run("returns a name qualified with its subpackage", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, names["equal"], "Equal", "the name of an assertion of the root namespace")
			assert.Equal(t, names["prop-for-all"], "prop.ForAll", "the name of an assertion of prop")
		})
	})

	t.Run("SurfaceNames", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the Go name of a type, a member and a helper", func(t *testing.T) {
			t.Parallel()

			surface := conformance.SurfaceNames()
			assert.Equal(t, surface["seat"], "TB", "the name of a type")
			assert.Equal(t, surface["seat.helper"], "Helper", "the name of a member")
			assert.Equal(t, surface["prop.integer"], "prop.Integer", "the name of a helper")
		})
	})

	t.Run("RelaxationNames", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a Go name for every relaxation that the overlay does not decline", func(t *testing.T) {
			t.Parallel()

			relaxations := conformance.RelaxationNames()
			assert.NotEmpty(t, relaxations, "the definition states relaxations")
			for id, name := range relaxations {
				assert.Equal(t, name == "", overlay.DeclinesRelaxation(id),
					"relaxation "+string(id)+" has a Go name unless the overlay declines it")
			}
		})
	})

	t.Run("Version", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the version of the vendored definition without its line break", func(t *testing.T) {
			t.Parallel()

			assert.Matches(t, conformance.Version(), `^\d+\.\d+\.\d+$`, "a version of three numbers")
		})
	})

	t.Run("Overlay", func(t *testing.T) {
		t.Parallel()

		t.Run("returns no divergence for this library", func(t *testing.T) {
			t.Parallel()

			// An empty overlay states that Go implements every assertion
			// of the standard. A divergence changes the library's
			// contract, and this case with it.
			assert.Empty(t, overlay.Diverge, "Go implements the whole standard, so it declares nothing absent")
		})

		t.Run("returns the definition and the language that the overlay extends", func(t *testing.T) {
			t.Parallel()

			assert.HasPrefix(t, overlay.Extends, "spec://", "the overlay names the definition it extends")
			assert.Equal(t, overlay.Language, "go", "the overlay names the language it speaks for")
		})
	})

	declared := conformance.OverlayDoc{
		Language: "php",
		Diverge: []conformance.Divergence{
			{ID: "bench-max-allocs", Stance: "blocked", Why: "no allocation counter"},
		},
		Relaxations: []conformance.Declined{
			{ID: "equate-nans", Why: "no comparison options"},
		},
		Surface: []conformance.Declined{
			{ID: "recorder-seat", Why: "no recorder"},
		},
	}

	t.Run("Diverges", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for a declared divergence", func(t *testing.T) {
			t.Parallel()

			assert.True(t, declared.Diverges("bench-max-allocs"), "a declared divergence is found")
		})

		t.Run("reports false for an assertion without a declared divergence", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declared.Diverges("equal"), "an assertion nobody declared absent is not a divergence")
		})

		t.Run("reports false for an empty overlay", func(t *testing.T) {
			t.Parallel()

			assert.False(t, conformance.OverlayDoc{}.Diverges("equal"),
				"an overlay without entries declares no divergence")
		})
	})

	t.Run("DeclinesRelaxation", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for a declined relaxation", func(t *testing.T) {
			t.Parallel()

			assert.True(t, declared.DeclinesRelaxation("equate-nans"), "a declined relaxation is found")
		})

		t.Run("reports false for a relaxation that the overlay does not decline", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declared.DeclinesRelaxation("equate-empty"), "a relaxation that nobody declined is offered")
		})
	})

	t.Run("DeclinesSurface", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for a declined id of the surface table", func(t *testing.T) {
			t.Parallel()

			assert.True(t, declared.DeclinesSurface("recorder-seat"), "a declined id is found")
		})

		t.Run("reports false for an id that the overlay does not decline", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declared.DeclinesSurface("seat"), "an id that nobody declined is offered")
		})
	})
}
