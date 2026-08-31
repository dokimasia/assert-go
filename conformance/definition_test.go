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

// TestDefinition is the completeness gate: every assertion the
// definition states must be present under the name the naming table
// gives it, or declared absent in the overlay.
//
// It is what would have caught the definition drifting from the
// implementation, which it did for a while: the definition described
// two assertions while the library shipped forty-one.
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

		t.Run("every assertion states its message fields", func(t *testing.T) {
			t.Parallel()

			for id, a := range assertions {
				assert.NotEmpty(t, a.Summary, "assertion "+string(id)+" states what it means")
				assert.True(t, a.Arity > 0, "assertion "+string(id)+" states its arity")
			}
		})
	})

	t.Run("Names", func(t *testing.T) {
		t.Parallel()

		t.Run("every assertion has a name in this language", func(t *testing.T) {
			t.Parallel()

			for id := range assertions {
				assert.Contains(t, names, id,
					"the naming table gives a Go name for "+string(id))
			}
		})

		t.Run("the naming table names nothing the definition does not", func(t *testing.T) {
			t.Parallel()

			for id := range names {
				assert.Contains(t, assertions, id,
					"the assertion table declares "+string(id))
			}
		})
	})

	t.Run("relaxations", func(t *testing.T) {
		t.Parallel()

		relaxations, err := conformance.RelaxationNames()
		assert.NoError(t, err, "the relaxations can be read")
		assert.NotEmpty(t, relaxations, "the definition states relaxations")

		members, err := conformance.Members(conformance.Aborting)
		assert.NoError(t, err, "the surface holding the relaxations can be read")

		for id, name := range relaxations {
			declined := overlay.DeclinesRelaxation(id)

			switch {
			case name == "" && !declined:
				t.Errorf("%s: the table gives no Go name and the overlay does not decline it",
					id)
			case name != "" && declined:
				t.Errorf("%s: the table names %s and the overlay declines it, which is a contradiction",
					id, name)
			case name != "" && !holds(members, name):
				t.Errorf("%s: %s is named and not implemented", id, name)
			}
		}
	})

	t.Run("completeness", func(t *testing.T) {
		t.Parallel()

		t.Run("every assertion is implemented or declared absent", func(t *testing.T) {
			t.Parallel()

			for id, a := range assertions {
				name := names[id]
				where, member := split(name)

				surface, ok := resolve(a.Package, where)
				assert.True(t, ok,
					"assertion "+string(id)+" names a package this library has")

				members, err := conformance.Members(surface)
				assert.NoError(t, err, "the surface holding "+string(id)+" can be read")

				present := holds(members, member)
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

		t.Run("this library declares no divergence", func(t *testing.T) {
			t.Parallel()

			// An empty overlay is a claim, not an absence: this
			// language can express every assertion the standard states.
			// A divergence appearing here is a deliberate change to
			// what the library promises, so it changes this case too.
			assert.Empty(t, overlay.Diverge,
				"Go implements the whole standard, so it declares nothing absent")
		})

		t.Run("it names the definition it was written against", func(t *testing.T) {
			t.Parallel()

			assert.HasPrefix(t, overlay.Extends, "spec://",
				"the overlay names the definition it extends")
			assert.Equal(t, overlay.Language, "go",
				"the overlay names the language it speaks for")
		})
	})

	t.Run("Version", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the definition this library implements", func(t *testing.T) {
			t.Parallel()

			version, err := conformance.Version()
			assert.NoError(t, err, "the version can be read")
			assert.NotEmpty(t, version, "the version is stated")
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

// resolve answers the surface an assertion's package names.
func resolve(declared, qualified string) (conformance.Surface, bool) {
	if declared == "" && qualified == "" {
		return conformance.Aborting, true
	}
	if declared == "" {
		declared = qualified
	}
	return conformance.Subpackage(declared)
}

// holds reports whether members carries name, which for a method is
// the type it hangs off.
func holds(members []string, name string) bool {
	owner, _, isMethod := strings.Cut(name, ".")
	if isMethod {
		name = owner
	}
	return slices.Contains(members, name)
}

// TestOverlayRules drives the overlay's own rule, which this
// library's empty overlay cannot: a divergence has to be found when
// one is declared.
func TestOverlayRules(t *testing.T) {
	t.Parallel()

	declared := conformance.OverlayDoc{
		Language: "php",
		Diverge: []conformance.Divergence{
			{ID: "bench-max-allocs", Stance: "blocked", Why: "no allocation counter"},
		},
	}

	t.Run("Diverges", func(t *testing.T) {
		t.Parallel()

		t.Run("finds a declared divergence", func(t *testing.T) {
			t.Parallel()

			assert.True(t, declared.Diverges("bench-max-allocs"),
				"a declared divergence is found")
		})

		t.Run("does not find one that was not declared", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declared.Diverges("equal"),
				"an assertion nobody declared absent is not a divergence")
		})

		t.Run("an empty overlay declares nothing", func(t *testing.T) {
			t.Parallel()

			assert.False(t, conformance.OverlayDoc{}.Diverges("equal"),
				"an overlay with no entries declares no divergence")
		})
	})
}
