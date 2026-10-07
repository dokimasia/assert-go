// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"slices"
)

// sortedFuncs are the functions that report whether values are sorted.
var sortedFuncs = []string{
	"slices.IsSorted", "slices.IsSortedFunc",
	"sort.IsSorted", "sort.SliceIsSorted", "sort.IntsAreSorted", "sort.StringsAreSorted", "sort.Float64sAreSorted",
}

// pairwise reports True of a function of sortedFuncs. Pairwise states that
// every adjacent pair of a sequence satisfies a predicate, such as an order.
func pairwise(p *pass, c check) bool {
	n, ok := ast.Unparen(c.cond).(*ast.CallExpr)
	if !ok || !c.holds || !slices.Contains(sortedFuncs, p.callee(n)) {
		return false
	}
	p.report(c.node, "pairwise", "Pairwise", nil)
	return true
}
