// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/types/typeutil"
)

// count is a comparison of a length with a constant, as len(x) > 0 is.
type count struct {
	op    token.Token
	value int64
}

// emptiness maps each comparison of a length with a constant that Empty or
// NotEmpty states to that assertion.
var emptiness = map[count]string{
	{token.EQL, 0}: "Empty", {token.LEQ, 0}: "Empty", {token.LSS, 1}: "Empty",
	{token.NEQ, 0}: "NotEmpty", {token.GTR, 0}: "NotEmpty", {token.GEQ, 1}: "NotEmpty",
}

// length reports a check of a length that Length, Empty or NotEmpty states:
// a comparison of len(x) with a count, Equal of len(x) and a count, NotEqual
// of len(x) and 0, and Length of x and 0.
//
// It suggests Length for a slice, an array, a map or a channel, whose length
// both len and Length count as their elements or entries. A string is none:
// len counts its bytes, and Length counts its Unicode scalar values. Empty and
// NotEmpty apply to a string as well, because a string without bytes has no
// scalar values. The rule leaves out a pointer to an array, whose len is the
// array's length and which has no length for Length.
func length(p *pass, c check) bool {
	x, n, op, viaLen := p.lengthCheck(c)
	if x == nil {
		return false
	}
	name := ""
	if value := p.TypesInfo.Types[n].Value; value != nil {
		v, _ := constant.Int64Val(constant.ToInt(value))
		name = emptiness[count{op, v}]
	}
	if name == "" && op == token.EQL {
		name = "Length"
	}
	if name == "" || c.is(name) || viaLen && !p.counted(x, name) {
		return false
	}
	args := p.source(x)
	if name == "Length" {
		args += ", " + p.source(n)
	}
	p.report(c.node, "length", name, c.rewrite(name, args))
	return true
}

// lengthCheck returns the check of a length that c states, as len(x) op n,
// and whether c reads the length through len. x is nil for a check that
// states no check of a length. len(x) is the value under test: the first of
// the two values, or the second where the first is a constant other than a
// length. In Equal(t, n, len(p)), n is the value under test.
func (p *pass) lengthCheck(c check) (x, n ast.Expr, op token.Token, viaLen bool) {
	var a, b ast.Expr
	switch {
	case c.is("Length"):
		return c.call.Args[1], c.call.Args[2], token.EQL, false
	case c.is("Equal", "NotEqual"):
		// A call with options compares as no check of a length does.
		if len(c.call.Args) != 4 {
			return nil, nil, 0, false
		}
		a, b, op = c.call.Args[1], c.call.Args[2], token.EQL
		if c.is("NotEqual") {
			op = token.NEQ
		}
	default:
		a, b, op, _ = c.comparison()
	}
	if p.lenArg(a) == nil && p.constant(a) {
		a, b, op = b, a, mirrors[op]
	}
	return p.lenArg(a), b, op, true
}

// lenArg returns x of the expression len(x), and nil for any other
// expression.
func (p *pass) lenArg(e ast.Expr) ast.Expr {
	n, ok := ast.Unparen(e).(*ast.CallExpr)
	if !ok {
		return nil
	}
	if b, ok := typeutil.Callee(p.TypesInfo, n).(*types.Builtin); ok && b.Name() == "len" {
		return n.Args[0]
	}
	return nil
}

// counted reports whether the assertion name counts the length of x as len
// counts it.
func (p *pass) counted(x ast.Expr, name string) bool {
	switch p.TypesInfo.TypeOf(x).Underlying().(type) {
	case *types.Slice, *types.Array, *types.Map, *types.Chan:
		return true
	case *types.Basic:
		return name != "Length"
	}
	return false
}
