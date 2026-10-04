// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestNaNPolicy checks which values are NaN policies.
func TestNaNPolicy(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for both policies and false past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t, []choice.NaNPolicy{choice.ExcludeNaN, choice.AdmitNaN},
				[]choice.NaNPolicy{invalidNaNPolicy})
		})
	})
}

// TestNaNPolicyAllocs checks that Valid allocates nothing.
func TestNaNPolicyAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.AdmitNaN.Valid() }, 0, "Valid allocates nothing")
}

// BenchmarkNaNPolicy measures Valid under a ceiling of no allocation.
func BenchmarkNaNPolicy(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.AdmitNaN.Valid()
		}
		assert.True(b, got, "AdmitNaN is a policy")
	})
}
