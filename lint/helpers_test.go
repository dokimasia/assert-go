// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// errorList keeps the errors that analysistest reports, in place of a test.
type errorList []string

// Errorf keeps the error.
func (l *errorList) Errorf(format string, args ...any) {
	*l = append(*l, fmt.Sprintf(format, args...))
}

// analyzeEach analyzes the packages of every case in one run, and reports
// the errors that name a case's package in a subtest of the case's name.
// Each diagnostic must match a want comment on its line, each want comment a
// diagnostic, and the fixes of each file must turn it into the file's golden
// file. analysistest parses and type-checks every dependency from source in
// each run, so one run for the cases loads the dependencies once. The test
// reports each error that names no package of a case.
func analyzeEach(t *testing.T, tests []fixture) {
	t.Helper()
	packages := make([]string, len(tests))
	for i, tt := range tests {
		packages[i] = tt.give
	}
	errs := analyze(packages...)
	claimed := make([]bool, len(errs))
	for _, tt := range tests {
		dir := filepath.Join(analysistest.TestData(), filepath.FromSlash(tt.give)) + string(filepath.Separator)
		var own []string
		for i, err := range errs {
			if strings.Contains(err, dir) {
				own, claimed[i] = append(own, err), true
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, err := range own {
				t.Error(err)
			}
		})
	}
	for i, err := range errs {
		if !claimed[i] {
			t.Error(err)
		}
	}
}

// analyze runs the analyzer over the packages of the test module under
// testdata, and returns the errors of the run.
//
// RunWithSuggestedFixes reads the golden file of a file that receives a fix
// and no other, so analyze also requires a fix in each file that has a golden
// file.
func analyze(packages ...string) []string {
	var errs errorList
	for _, result := range analysistest.RunWithSuggestedFixes(&errs, analysistest.TestData(), lint.Analyzer, packages...) {
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
				errs.Errorf("%s has a golden file, and the analyzer suggests no fix in it", file)
			}
		}
	}
	return errs
}
