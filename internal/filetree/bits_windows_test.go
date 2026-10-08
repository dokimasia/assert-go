// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build windows

package filetree_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// TestBits checks the permission bits of Windows, whose file systems record
// the owner's write bit of a file alone, as its read-only attribute.
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

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		t.Run("sets no read-only attribute on a directory, where Windows does not honour it", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			assert.NoError(t, filetree.Write(dir, filetree.Tree{"ro": dirWith(0o555)}), "the tree is written")
			info, err := os.Stat(filepath.Join(dir, "ro"))
			assert.NoError(t, err, "the directory is there")
			assert.Equal(t, info.Mode().Perm()&ownerWrite, ownerWrite, "the directory has no read-only attribute")
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
		{Name: "ModesUnrecorded", Call: func(assert.TB) { errKept = filetree.ModesUnrecorded() }, Allocs: 3},
	}
}
