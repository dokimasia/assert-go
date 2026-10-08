// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"

	"go.dokimi.dev/assert/lint"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		t.Run("is a valid analyzer named assertlint", func(t *testing.T) {
			t.Parallel()
			if err := analysis.Validate([]*analysis.Analyzer{lint.Analyzer}); err != nil {
				t.Errorf("Validate returns %v, want nil", err)
			}
			if lint.Analyzer.Name != "assertlint" {
				t.Errorf("Name is %q, want %q", lint.Analyzer.Name, "assertlint")
			}
		})
		uncovered := []string{
			"accumulates", "bench-max-latency", "golden-match-json-field", "golden-match-tree",
			"tree-equal", "tree-contains", "tree-unchanged", "linearizable", "serializable", "snapshot-isolation",
		}
		tests := make([]fixture, 0, len(uncovered))
		for _, assertion := range uncovered {
			tests = append(tests, fixture{
				name: "reports no call of " + assertion + ", which no rule covers",
				give: "./" + assertion,
			})
		}
		analyzeEach(t, tests)
	})
}
