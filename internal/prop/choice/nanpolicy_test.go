// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// invalidNaNPolicy is the first value past the two policies.
const invalidNaNPolicy = choice.NaNPolicy(2)

// TestNaNPolicy checks which values are NaN policies, and pins each
// policy's spelling.
func TestNaNPolicy(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for both policies", func(t *testing.T) {
			t.Parallel()
			for _, p := range []choice.NaNPolicy{choice.ExcludeNaN, choice.AdmitNaN} {
				assert.True(t, p.Valid(), p.String()+" is a policy")
			}
		})

		t.Run("reports false past the two policies", func(t *testing.T) {
			t.Parallel()
			assert.False(t, invalidNaNPolicy.Valid(), "policy 2 is no policy")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.NaNPolicy
			want string
		}{
			{name: "returns exclude-nan for ExcludeNaN", give: choice.ExcludeNaN, want: "exclude-nan"},
			{name: "returns admit-nan for AdmitNaN", give: choice.AdmitNaN, want: "admit-nan"},
			{name: "returns NaNPolicy(2) for a value past the policies", give: invalidNaNPolicy, want: "NaNPolicy(2)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the policy's spelling")
			})
		}
	})
}

// TestNaNPolicyZeroAlloc checks that no method of NaNPolicy allocates.
func TestNaNPolicyZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.AdmitNaN.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.AdmitNaN.String() }, 0, "String allocates nothing")
}

// BenchmarkNaNPolicy measures each method of NaNPolicy under a ceiling of
// no allocation.
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
