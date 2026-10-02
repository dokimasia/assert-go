// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestDefinition is the completeness gate: every assertion of the
// definition is present under the name that the naming table gives Go, or
// the overlay declares it absent.
func TestDefinition(t *testing.T) {
	t.Parallel()

	assertions, err := conformance.Assertions()
	assert.NoError(t, err, "the assertion table can be read")
	assert.NotEmpty(t, assertions, "the assertion table states something")

	names, err := conformance.Names()
	assert.NoError(t, err, "the naming table can be read")

	overlay, err := conformance.Overlay()
	assert.NoError(t, err, "this language's overlay can be read")

	t.Run("Assertions", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a summary and an arity for every assertion", func(t *testing.T) {
			t.Parallel()

			for id, a := range assertions {
				assert.NotEmpty(t, a.Summary, "assertion "+string(id)+" states what it means")
				assert.True(t, a.Arity > 0, "assertion "+string(id)+" states its arity")
			}
		})
	})

	t.Run("Names", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a Go name for every assertion of the definition", func(t *testing.T) {
			t.Parallel()

			for id := range assertions {
				assert.Contains(t, names, id,
					"the naming table gives a Go name for "+string(id))
			}
		})

		t.Run("returns no name for an assertion outside the definition", func(t *testing.T) {
			t.Parallel()

			for id := range names {
				assert.Contains(t, assertions, id,
					"the assertion table declares "+string(id))
			}
		})
	})

	t.Run("RelaxationNames", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a Go name for every relaxation that the overlay does not decline", func(t *testing.T) {
			t.Parallel()

			relaxations, err := conformance.RelaxationNames()
			assert.NoError(t, err, "the relaxations can be read")
			assert.NotEmpty(t, relaxations, "the definition states relaxations")

			members, err := conformance.Members(conformance.Aborting)
			assert.NoError(t, err, "the surface of the relaxations can be read")

			for id, name := range relaxations {
				declined := overlay.DeclinesRelaxation(id)

				switch {
				case name == "" && !declined:
					t.Errorf("%s: the table gives no Go name and the overlay does not decline it",
						id)
				case name != "" && declined:
					t.Errorf("%s: the table names %s and the overlay declines it, which is a contradiction",
						id, name)
				case name != "" && !declares(members, name):
					t.Errorf("%s: %s is named and not implemented", id, name)
				}
			}
		})
	})

	t.Run("Members", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the Go name of every assertion that the overlay does not declare absent", func(t *testing.T) {
			t.Parallel()

			for id, a := range assertions {
				name := names[id]
				where, member := split(name)

				surface, ok := resolve(a.Package, where)
				assert.True(t, ok,
					"assertion "+string(id)+" names a package this library has")

				members, err := conformance.Members(surface)
				assert.NoError(t, err, "the surface of "+string(id)+" can be read")

				present := declares(members, member)
				declared := overlay.Diverges(id)

				switch {
				case present && declared:
					t.Errorf("%s: the overlay declares it absent, but %s is implemented",
						id, name)
				case !present && !declared:
					t.Errorf("%s: %s is not implemented and no overlay entry declares why",
						id, name)
				}
			}
		})
	})

	t.Run("Overlay", func(t *testing.T) {
		t.Parallel()

		t.Run("returns no divergence for this library", func(t *testing.T) {
			t.Parallel()

			// An empty overlay states that Go implements every assertion
			// of the standard. A divergence changes the library's
			// contract, and this case with it.
			assert.Empty(t, overlay.Diverge,
				"Go implements the whole standard, so it declares nothing absent")
		})

		t.Run("returns the definition and the language that the overlay extends", func(t *testing.T) {
			t.Parallel()

			assert.HasPrefix(t, overlay.Extends, "spec://",
				"the overlay names the definition it extends")
			assert.Equal(t, overlay.Language, "go",
				"the overlay names the language it speaks for")
		})
	})

	t.Run("Version", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the version of the vendored definition", func(t *testing.T) {
			t.Parallel()

			version, err := conformance.Version()
			assert.NoError(t, err, "the version can be read")
			assert.NotEmpty(t, version, "the version is stated")
		})
	})
}

// TestOverlayRules drives the rules of an overlay that this library's empty
// overlay cannot: the declared divergences and the declined relaxations of
// another language.
func TestOverlayRules(t *testing.T) {
	t.Parallel()

	declared := conformance.OverlayDoc{
		Language: "php",
		Diverge: []conformance.Divergence{
			{ID: "bench-max-allocs", Stance: "blocked", Why: "no allocation counter"},
		},
		Relaxations: []conformance.Declined{
			{ID: "equate-nans", Why: "no comparison options"},
		},
	}

	t.Run("Diverges", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for a declared divergence", func(t *testing.T) {
			t.Parallel()

			assert.True(t, declared.Diverges("bench-max-allocs"),
				"a declared divergence is found")
		})

		t.Run("reports false for an assertion without a declared divergence", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declared.Diverges("equal"),
				"an assertion nobody declared absent is not a divergence")
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

			assert.True(t, declared.DeclinesRelaxation("equate-nans"),
				"a declined relaxation is found")
		})

		t.Run("reports false for a relaxation that the overlay does not decline", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declared.DeclinesRelaxation("equate-empty"),
				"a relaxation that nobody declined is offered")
		})
	})
}

// split separates a qualified name into the package it names and the
// member within it. An unqualified name has no package.
func split(name string) (pkg, member string) {
	where, rest, qualified := strings.Cut(name, ".")
	if !qualified {
		return "", name
	}
	return where, rest
}

// resolve returns the surface that an assertion's package names.
func resolve(declared, qualified string) (conformance.Surface, bool) {
	if declared == "" && qualified == "" {
		return conformance.Aborting, true
	}
	if declared == "" {
		declared = qualified
	}
	return conformance.Subpackage(declared)
}

// declares reports whether members contains name, or for a method the type
// that the method belongs to.
func declares(members []string, name string) bool {
	owner, _, isMethod := strings.Cut(name, ".")
	if isMethod {
		name = owner
	}
	return slices.Contains(members, name)
}
