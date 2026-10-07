// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ast/inspector"
)

// breakable are the statements that a break without a label ends.
var breakable = []ast.Node{
	(*ast.ForStmt)(nil),
	(*ast.RangeStmt)(nil),
	(*ast.SwitchStmt)(nil),
	(*ast.TypeSwitchStmt)(nil),
	(*ast.SelectStmt)(nil),
}

// eventually reports a loop that waits for a condition. EventuallyTrue and
// Eventually state the wait: they retry within a timeout, and report the last
// failure when it passes.
func eventually(p *pass, cursor inspector.Cursor) {
	if p.waits(cursor) {
		p.report(cursor.Node(), "eventually", "EventuallyTrue or Eventually", nil)
	}
}

// waits reports whether the loop at cursor waits for a condition:
//
//   - It calls time.Sleep.
//   - It can end on a condition before a count of rounds: its body returns or
//     breaks out of it, or it is a for statement whose condition compares no
//     counter.
//   - A test is in scope, which the assertions take.
//
// A loop that sleeps for a count of rounds, such as for range 2, waits for no
// condition.
func (p *pass) waits(loop inspector.Cursor) bool {
	return p.sleeps(loop) && (exits(loop) || p.conditional(loop.Node())) && p.tested(loop)
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

// exits reports whether a return or a break of the body of the loop at
// cursor ends the loop.
func exits(loop inspector.Cursor) bool {
	for exit := range loop.Preorder((*ast.ReturnStmt)(nil), (*ast.BranchStmt)(nil)) {
		if ends(exit, loop) {
			return true
		}
	}
	return false
}

// ends reports whether the statement at exit, a return or a branch inside
// the loop at loop, ends the loop. A return ends it outside a function
// literal of the body. A break without a label ends the innermost loop,
// switch or select around it, and a break with a label ends the statement of
// the label, which can be the loop or a statement around it.
func ends(exit, loop inspector.Cursor) bool {
	branch, isBranch := exit.Node().(*ast.BranchStmt)
	switch {
	case !isBranch:
		lit, inLit := innermost(exit, (*ast.FuncLit)(nil))
		return !inLit || !loop.Contains(lit)
	case branch.Tok != token.BREAK:
		return false
	case branch.Label == nil:
		target, _ := innermost(exit, breakable...)
		return target == loop
	}
	for around := range exit.Enclosing((*ast.LabeledStmt)(nil)) {
		if around.Node().(*ast.LabeledStmt).Label.Name == branch.Label.Name && loop.Contains(around) {
			return false
		}
	}
	return true
}

// innermost returns the nearest node around the node at cursor whose type is
// one of kinds, and false where none is.
func innermost(cursor inspector.Cursor, kinds ...ast.Node) (inspector.Cursor, bool) {
	for around := range cursor.Enclosing(kinds...) {
		return around, true
	}
	return inspector.Cursor{}, false
}

// conditional reports whether the loop n is a for statement whose condition
// is no comparison of a counter, a variable that the post statement or the
// body of the loop changes, as i++ changes i in i < 10.
func (p *pass) conditional(n ast.Node) bool {
	loop, ok := n.(*ast.ForStmt)
	if !ok || loop.Cond == nil {
		return false
	}
	b, isBinary := ast.Unparen(loop.Cond).(*ast.BinaryExpr)
	if !isBinary {
		return true
	}
	_, isComparison := negations[b.Op]
	return !isComparison || !p.counter(loop, b.X) && !p.counter(loop, b.Y)
}

// counter reports whether e names a variable that the post statement or the
// body of loop changes.
func (p *pass) counter(loop *ast.ForStmt, e ast.Expr) bool {
	v := p.variable(e)
	return v != nil && (p.changes(loop.Body, v) || loop.Post != nil && p.changes(loop.Post, v))
}

// tested reports whether a test is in scope at the node at cursor: a
// variable of an enclosing function, such as a parameter, whose method set
// has Helper.
func (p *pass) tested(cursor inspector.Cursor) bool {
	for s := p.Pkg.Scope().Innermost(cursor.Node().Pos()); s != p.Pkg.Scope(); s = s.Parent() {
		for _, name := range s.Names() {
			if v, ok := s.Lookup(name).(*types.Var); ok && types.NewMethodSet(v.Type()).Lookup(nil, "Helper") != nil {
				return true
			}
		}
	}
	return false
}

// polls reports whether the node at cursor is inside a loop that waits for a
// condition. A check there, such as one of a deadline, is part of the wait
// that eventually reports.
func (p *pass) polls(cursor inspector.Cursor) bool {
	for loop := range cursor.Enclosing(loopNodes...) {
		if p.waits(loop) {
			return true
		}
	}
	return false
}
