// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestProp(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		forms := []string{
			"accumulates", "associative", "close-to", "commutative", "contains", "contains-in-order",
			"deterministic", "empty", "equal", "err-absent", "err-as", "err-is", "err-is-not", "err-present",
			"false", "has-prefix", "has-suffix", "honours-cancellation", "honours-deadline", "idempotent",
			"in-range", "length", "matches", "max-allocs", "max-allocs-with-setup", "nil", "nil-context-safe",
			"not-contains", "not-empty", "not-equal", "not-nil", "not-pure", "not-throws", "pairwise",
			"permutation", "pure", "round-trip", "throws", "true",
		}
		tests := make([]fixture, 0, 2+len(forms))
		tests = append(tests,
			fixture{
				name: "reports a loop over random values and a call of testing/quick as prop.ForAll",
				give: "./prop-for-all",
			},
			fixture{name: "reports a ForAll that loops over drawn actions as stateful.Steps", give: "./machine"},
		)
		for _, form := range forms {
			tests = append(tests, fixture{
				name: "reports a ForAll of one draw and one assertion as the form prop-" + form,
				give: "./prop-" + form,
			})
		}
		analyzeEach(t, tests)
	})
}
