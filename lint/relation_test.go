// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestRelation(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "suggests Commutative for an equality of f(a, b) and f(b, a)", give: "./commutative"},
			{
				name: "suggests Associative for an equality of the two groupings of three operands",
				give: "./associative",
			},
			{name: "reports an equality of a value and its conversion back as RoundTrip", give: "./round-trip"},
			{
				name: "reports an equality of two consecutive calls with one input as Deterministic",
				give: "./deterministic",
			},
			{name: "reports an equality of two consecutive listings as StableOrder", give: "./stable-order"},
			{name: "reports an equality of two readings around a call as Pure", give: "./pure"},
			{name: "reports a difference of two readings around a call as NotPure", give: "./not-pure"},
			{name: "reports an equality of two readings after one call each as Idempotent", give: "./idempotent"},
			{name: "reports a loop that checks and assigns seen[x] as NoDuplicates", give: "./no-duplicates"},
			{name: "reports a loop that keeps the value of the step before as Monotonic", give: "./monotonic"},
			{name: "suggests Total for a loop whose body is one NoError of a call of the element", give: "./total"},
			{
				name: "reports a failure of a call after a statement that calls Close as FailsAfterClose",
				give: "./after-close",
			},
			{name: "reports a counted loop whose body is one HasError of a call as Poisoned", give: "./poisoned"},
		}
		analyzeEach(t, tests)
	})
}
