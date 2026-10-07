// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// containsFuncs are the packages whose function Contains reports whether a
// haystack contains a needle.
var containsFuncs = []string{"strings", "bytes", "slices"}

// sorts are the functions that sort the slice of their first argument in
// place.
var sorts = []string{
	"slices.Sort", "slices.SortFunc", "slices.SortStableFunc",
	"sort.Ints", "sort.Strings", "sort.Float64s", "sort.Slice", "sort.SliceStable",
}

// contains reports True or False of strings.Contains, bytes.Contains and
// slices.Contains, and of the ok of a lookup of a map. Contains and
// NotContains state the check.
//
// It suggests the assertion for every function but slices.Contains over
// bytes, which Contains reads as text, and over elements that are no
// booleans, numbers or strings, which Contains compares as Equal does and ==
// does not. A lookup gets no fix, because its key is in another statement.
func contains(p *pass, c check) bool {
	name := "Contains"
	if !c.holds {
		name = "NotContains"
	}
	if lookup := p.lookups[p.variable(c.cond)]; lookup != nil {
		p.report(c.node, "contains", name+" of "+p.brief(lookup), nil)
		return true
	}
	for _, path := range containsFuncs {
		n, ok := p.callOf(c.cond, path, "Contains")
		if !ok {
			continue
		}
		var fixes []analysis.SuggestedFix
		if path != "slices" || textless(p.TypesInfo.TypeOf(n.Args[1])) {
			fixes = c.rewrite(name, p.source(n.Args[0])+", "+p.source(n.Args[1]))
		}
		p.report(c.node, "contains", name, fixes)
		return true
	}
	return false
}

// textless reports whether t is a boolean, numeric or string type other than
// byte itself, whose slice Contains reads as text. A type defined over uint8,
// such as an enumeration, is a number.
func textless(t types.Type) bool {
	return basic(t) && !types.Identical(t, types.Typ[types.Byte])
}

// membership reports a check that one variable equals one of several
// values: True of x == a || x == b, or False of x != a && x != b. Contains
// states it with the members in a slice, and the rule suggests
// Contains(t, []T{a, b}, x, msg) where x is a boolean, number or string other
// than a byte, whose type T the file can write, and no member calls a
// function: the slice evaluates every member, and || stops at the first
// equal one. Contains reads a slice of bytes as text.
func membership(p *pass, c check) bool {
	chain, comparison := token.LOR, token.EQL
	if !c.holds {
		chain, comparison = token.LAND, token.NEQ
	}
	terms := operands(c.cond, chain)
	first, ok := terms[0].(*ast.BinaryExpr)
	if len(terms) < 2 || !ok {
		return false
	}
	for _, side := range []ast.Expr{first.X, first.Y} {
		v := p.variable(side)
		members, ok := p.members(terms, v, comparison)
		if v == nil || !ok {
			continue
		}
		var fixes []analysis.SuggestedFix
		text, spelled := p.spell(c.node.Pos(), v.Type())
		if textless(v.Type()) && spelled && !slices.ContainsFunc(members, func(e ast.Expr) bool { return !pure(e) }) {
			values := make([]string, len(members))
			for i, member := range members {
				values[i] = p.source(member)
			}
			fixes = c.rewrite("Contains", "[]"+text+"{"+strings.Join(values, ", ")+"}, "+v.Name())
		}
		p.report(c.node, "membership", "Contains", fixes)
		return true
	}
	return false
}

// members returns the values that the terms compare the variable v with, and
// false where a term is no comparison of v through op.
func (p *pass) members(terms []ast.Expr, v *types.Var, op token.Token) ([]ast.Expr, bool) {
	members := make([]ast.Expr, 0, len(terms))
	for _, term := range terms {
		b, ok := term.(*ast.BinaryExpr)
		switch {
		case !ok || b.Op != op:
			return nil, false
		case p.variable(b.X) == v:
			members = append(members, b.Y)
		case p.variable(b.Y) == v:
			members = append(members, b.X)
		default:
			return nil, false
		}
	}
	return members, true
}

// containsInOrder reports an order of two indices of strings.Index in one
// text, as strings.Index(s, a) < strings.Index(s, b). ContainsInOrder states
// it, and requires each needle to be present: a missing needle has the index
// -1, which is below every index.
func containsInOrder(p *pass, c check) bool {
	x, y, op, ok := c.comparison()
	if op == token.GTR || op == token.GEQ {
		x, y, op = y, x, mirrors[op]
	}
	a, isA := p.callOf(x, "strings", "Index")
	b, isB := p.callOf(y, "strings", "Index")
	if !ok || op != token.LSS && op != token.LEQ || !isA || !isB || p.source(a.Args[0]) != p.source(b.Args[0]) {
		return false
	}
	p.report(c.node, "contains-in-order", "ContainsInOrder", nil)
	return true
}

// permutation reports an equality of two slices that earlier statements of
// its list sort in place. Permutation states that two slices contain the
// same elements in any order. A check that sorts one slice keeps the order
// of the other in the comparison.
func permutation(p *pass, c check) bool {
	e, ok := p.equality(c)
	stmt, listed := c.statement()
	if !ok || !e.equal || !listed {
		return false
	}
	x, y := p.sorted(stmt, e.x), p.sorted(stmt, e.y)
	if x == nil || y == nil {
		return false
	}
	p.report(c.node, "permutation", "Permutation of "+p.brief(x)+" and "+p.brief(y), nil)
	return true
}

// sorted returns the call of the nearest earlier statement of the list of
// stmt that sorts the variable that e names in place, and nil where none
// does.
func (p *pass) sorted(stmt inspector.Cursor, e ast.Expr) *ast.CallExpr {
	v := p.variable(e)
	for _, s := range before(stmt) {
		if n := p.sorts(s.Node(), v); n != nil {
			return n
		}
	}
	return nil
}

// sorts returns the statement s as a call of a function of sorts on the
// variable v, and nil for any other statement.
func (p *pass) sorts(s ast.Node, v *types.Var) *ast.CallExpr {
	stmt, ok := s.(*ast.ExprStmt)
	if !ok || v == nil {
		return nil
	}
	n, ok := stmt.X.(*ast.CallExpr)
	if !ok || !slices.Contains(sorts, p.callee(n)) || p.variable(n.Args[0]) != v {
		return nil
	}
	return n
}
