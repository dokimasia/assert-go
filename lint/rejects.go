// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/types"
)

// rejects reports True of Failed of a test other than the check's own, such
// as an assert.Recorder that a check ran on. Rejects runs the check against
// the implementation that it must reject, and returns the failure records.
func rejects(p *pass, c check) bool {
	n, ok := ast.Unparen(c.cond).(*ast.CallExpr)
	if !ok || !c.holds {
		return false
	}
	sel, ok := n.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Failed" || p.source(sel.X) == p.source(c.tb) {
		return false
	}
	methods := types.NewMethodSet(p.TypesInfo.TypeOf(sel.X))
	if methods.Lookup(nil, "Helper") == nil {
		return false
	}
	p.report(c.node, "rejects", "Rejects", nil)
	return true
}
