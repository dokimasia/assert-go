// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package filetree_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// TestBits checks the permission bits of a platform whose file systems
// record none.
func TestBits(t *testing.T) {
	t.Parallel()

	t.Run("ModesUnrecorded", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a fault where the file systems record no permission bits", func(t *testing.T) {
			t.Parallel()

			f := assert.ErrorAs[*fault.Error](t, filetree.ModesUnrecorded(), "a fault")
			assert.Equal(t, f.Reason, "the file system records no permission bits", "the reason")
		})
	})
}

// TestBitsAllocs checks the allocation ceiling of ModesUnrecorded.
func TestBitsAllocs(t *testing.T) {
	alloctest.Check(t, bitsCases())
}

// BenchmarkBits measures ModesUnrecorded.
func BenchmarkBits(b *testing.B) {
	for _, c := range bitsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// bitsCases returns a call of ModesUnrecorded, which allocates its fault and
// its reason here.
func bitsCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "ModesUnrecorded", Call: func(assert.TB) { errKept = filetree.ModesUnrecorded() }, Allocs: 2},
	}
}
