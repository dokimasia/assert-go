// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/types/typeutil"
)

// panics reports a check of the result of recover(), which a deferred
// function makes. Panics states a check that the result is present, and
// NotPanics a check that it is nil. The rule reads Nil, NotNil, and a
// comparison of the result with nil.
func panics(p *pass, c check) bool {
	value, isNil, ok := p.nilness(c)
	if !ok || !p.recovers(value) {
		return false
	}
	if isNil {
		p.report(c.node, "not-panics", "NotPanics", nil)
	} else {
		p.report(c.node, "panics", "Panics", nil)
	}
	return true
}

// nilness returns the value of a check that a value is nil, or that it is
// not where isNil is false: Nil, NotNil, and a comparison with nil. It
// returns false for any other check.
func (p *pass) nilness(c check) (value ast.Expr, isNil, ok bool) {
	if c.is("Nil", "NotNil") {
		return c.call.Args[1], c.is("Nil"), true
	}
	x, y, op, ok := c.comparison()
	if ok && p.TypesInfo.Types[x].IsNil() {
		x, y = y, x
	}
	return x, op == token.EQL, ok && p.TypesInfo.Types[y].IsNil()
}

// recovers reports whether a source of e is a call of recover.
func (p *pass) recovers(e ast.Expr) bool {
	for _, n := range p.sources(e) {
		if b, ok := typeutil.Callee(p.TypesInfo, n).(*types.Builtin); ok && b.Name() == "recover" {
			return true
		}
	}
	return false
}
