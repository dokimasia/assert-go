// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"

	"golang.org/x/tools/go/ast/inspector"
)

// eventually reports a loop that calls time.Sleep, which polls a condition.
// EventuallyTrue and Eventually state the wait: they retry within a timeout,
// and report the last failure when it passes.
func eventually(p *pass, cursor inspector.Cursor) {
	if p.sleeps(cursor) {
		p.report(cursor.Node(), "eventually", "EventuallyTrue or Eventually", nil)
	}
}

// sleeps reports whether the loop at cursor calls time.Sleep.
func (p *pass) sleeps(loop inspector.Cursor) bool {
	for inner := range loop.Preorder((*ast.CallExpr)(nil)) {
		if p.callee(inner.Node().(*ast.CallExpr)) == "time.Sleep" {
			return true
		}
	}
	return false
}

// polls reports whether the node at cursor is inside a loop that calls
// time.Sleep. A check there, such as one of a deadline, is part of the wait
// that eventually reports.
func (p *pass) polls(cursor inspector.Cursor) bool {
	for loop := range cursor.Enclosing((*ast.ForStmt)(nil), (*ast.RangeStmt)(nil)) {
		if p.sleeps(loop) {
			return true
		}
	}
	return false
}
