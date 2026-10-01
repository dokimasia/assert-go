// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/random"
)

// The pinned mix of one contract, from the definition's executable
// reference, so that a change to the mixing fails here before a vector
// moves.
const (
	// pinnedContract is the contract whose mix is pinned.
	pinnedContract = "decoding undoes encoding"
	// pinnedMix is the mix of pinnedContract.
	pinnedMix = 14330315428228886101
)

// TestMix checks the fold of bytes into a seed.
func TestMix(t *testing.T) {
	t.Parallel()

	t.Run("Mix", func(t *testing.T) {
		t.Parallel()

		t.Run("returns 0 for no bytes", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, random.Mix(""), uint64(0), "the fold starts at 0")
		})

		t.Run("returns the first value of the stream of the value XOR each byte", func(t *testing.T) {
			t.Parallel()
			first := random.New(0 ^ 'a')
			second := random.New(first.Next() ^ 'b')
			assert.Equal(t, random.Mix("ab"), second.Next(), "the fold of two bytes")
		})

		t.Run("returns the pinned mix of a contract", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, random.Mix(pinnedContract), uint64(pinnedMix), "the mix the reference computes")
		})
	})
}

// TestMixZeroAlloc checks that Mix allocates nothing.
func TestMixZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = random.Mix(pinnedContract) }, 0, "Mix allocates nothing")
}

// BenchmarkMix measures Mix over a contract under a ceiling of no
// allocation. Each byte costs one initialisation of the stream.
func BenchmarkMix(b *testing.B) {
	b.Run("Mix", func(b *testing.B) {
		var got uint64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.Mix(pinnedContract)
		}
		assert.Equal(b, got, uint64(pinnedMix), "the pinned mix")
	})
}
