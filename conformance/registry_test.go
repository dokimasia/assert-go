// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/expect"
)

// TestRegistry checks the registry of the invokers of each assertion in
// each form, and the options of the relaxations.
func TestRegistry(t *testing.T) {
	t.Parallel()

	byAssertion := conformance.Cases()

	t.Run("Registry", func(t *testing.T) {
		t.Parallel()

		t.Run("registers only assertions that the definition states", func(t *testing.T) {
			t.Parallel()

			assertions := conformance.Assertions()
			for id := range conformance.Registry {
				assert.Contains(t, assertions, id, "the definition states "+string(id))
			}
		})

		t.Run("registers only assertions that corpus cases cover", func(t *testing.T) {
			t.Parallel()

			for id := range conformance.Registry {
				assert.Contains(t, byAssertion, id, "the corpus covers "+string(id))
			}
		})

		t.Run("drives both function forms of every assertion that a case states values for", func(t *testing.T) {
			t.Parallel()

			for id, cases := range byAssertion {
				if statesOnlySubjects(cases) {
					continue
				}
				for _, form := range []conformance.Form{conformance.AbortingCall, conformance.RecordingCall} {
					assert.Contains(t, conformance.Registry[id], form,
						"the registry drives "+string(id)+" as "+string(form))
				}
			}
		})

		t.Run("drives each chain form of every assertion whose name the chain declares", func(t *testing.T) {
			t.Parallel()

			names := conformance.Names()
			chains := map[conformance.Form]reflect.Type{
				conformance.AbortingChain:  reflect.TypeFor[*assert.Assertion[any]](),
				conformance.RecordingChain: reflect.TypeFor[*expect.Assertion[any]](),
			}
			for id, cases := range byAssertion {
				if statesOnlySubjects(cases) {
					continue
				}
				for form, chain := range chains {
					_, declared := chain.MethodByName(names[id])
					_, driven := conformance.Registry[id][form]
					assert.Equal(t, driven, declared,
						"the registry drives "+string(id)+" as "+string(form)+" when the chain declares "+names[id])
				}
			}
		})

		t.Run("drives the assertion that an invoker names", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			conformance.Registry["equal"][conformance.AbortingCall](r, []any{1, 2}, "the values match", nil)

			assert.True(t, r.Failed(), "the equal invoker reports on differing values")
			assert.Equal(t, r.Failures()[0].Assertion, "equal", "the record is of equal")
		})
	})

	t.Run("Relaxations", func(t *testing.T) {
		t.Parallel()

		t.Run("maps every relaxation that the overlay does not decline to an option", func(t *testing.T) {
			t.Parallel()

			overlay := conformance.Overlay()
			for id := range conformance.RelaxationNames() {
				_, mapped := conformance.Relaxations[id]
				assert.Equal(t, mapped, !overlay.DeclinesRelaxation(id),
					"the registry maps "+string(id)+" unless the overlay declines it")
			}
		})
	})
}
