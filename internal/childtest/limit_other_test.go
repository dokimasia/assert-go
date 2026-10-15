// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !unix

package childtest_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
)

func TestLimit(t *testing.T) {
	t.Parallel()

	t.Run("CanLimitFileSize", func(t *testing.T) {
		t.Parallel()

		t.Run("reports that the system sets no limit on the size of a file", func(t *testing.T) {
			t.Parallel()
			assert.False(t, childtest.CanLimitFileSize, "the system sets no limit on the size of a file")
		})
	})

	t.Run("LimitFileSize", func(t *testing.T) {
		t.Parallel()

		t.Run("skips the test on a system without the limit", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				childtest.LimitFileSize(t, 1)
				return
			}
			out, err := childtest.Run(t, t.Name())
			assert.NoError(t, err, "the child passes:\n"+out)
			assert.Contains(t, out, "--- SKIP: "+t.Name()+" ", "the child's test skips")
		})
	})
}
