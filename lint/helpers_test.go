// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import (
	"os"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"go.dokimi.dev/assert/lint"
)

// fixture is a case of a spec: a package of the test module under testdata,
// and the name of the subtest that analyzes it.
type fixture struct {
	name string
	give string
}

// analyzeEach analyzes the package of each case in a subtest of the case's
// name.
func analyzeEach(t *testing.T, tests []fixture) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analyze(t, tt.give)
		})
	}
}

// analyze runs the analyzer over the packages of the test module under
// testdata. Each diagnostic must match a want comment on its line, each want
// comment a diagnostic, and the fixes of each file must turn it into the
// file's golden file.
//
// RunWithSuggestedFixes reads the golden file of a file that receives a fix
// and no other, so analyze also requires a fix in each file that has a golden
// file.
func analyze(t *testing.T, packages ...string) {
	t.Helper()
	for _, result := range analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), lint.Analyzer, packages...) {
		fixed := make(map[string]bool)
		for _, diagnostic := range result.Action.Diagnostics {
			for _, fix := range diagnostic.SuggestedFixes {
				for _, edit := range fix.TextEdits {
					fixed[result.Action.Package.Fset.File(edit.Pos).Name()] = true
				}
			}
		}
		for _, file := range result.Action.Package.GoFiles {
			if _, err := os.Stat(file + ".golden"); err == nil && !fixed[file] {
				t.Errorf("%s has a golden file, and the analyzer suggests no fix in it", file)
			}
		}
	}
}
