// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestNaNPolicyString pins the spelling of each NaN policy.
func TestNaNPolicyString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of each policy and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[choice.NaNPolicy]string{
				choice.ExcludeNaN: "exclude-nan", choice.AdmitNaN: "admit-nan", invalidNaNPolicy: "NaNPolicy(2)",
			})
		})
	})
}

// TestNaNPolicyStringAllocs checks that String allocates nothing for a
// policy.
func TestNaNPolicyStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.AdmitNaN.String() }, 0, "String allocates nothing for a policy")
}

// BenchmarkNaNPolicyString measures String under a ceiling of no
// allocation.
func BenchmarkNaNPolicyString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.AdmitNaN.String()
		}
		assert.Equal(b, got, "admit-nan", "the policy's spelling")
	})
}
