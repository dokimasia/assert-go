// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"go.dokimi.dev/assert/lint"
)

func main() {
	singlechecker.Main(lint.Analyzer)
}
