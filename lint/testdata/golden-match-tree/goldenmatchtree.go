// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package goldenmatchtree

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/golden"
)

func correct(t *testing.T, dir string) {
	golden.MatchTree(t, "generated", os.DirFS(dir), golden.ShouldUpdate())
}
