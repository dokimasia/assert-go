// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
)

// exact is the largest magnitude of an integer that a float64 represents
// exactly, 2^53.
const exact = 1 << 53

// order reports True or False of <, <=, > or >= between a number and a
// constant bound, which InRange states with the number's range. An order of
// two variables states no range of one number.
//
// It suggests InRange(t, x, low, high, msg) for an integer x and a bound that
// limit can state exactly. The open end of the range is the bound that no
// integer of x's type passes as a float64: -1<<63 and 1<<63 for a signed
// integer, and 0 and 1<<64 for an unsigned one.
func order(p *pass, c check) bool {
	x, y, op, ok := c.comparison()
	if !ok || op == token.EQL || op == token.NEQ || !numeric(p.TypesInfo.TypeOf(x)) || !numeric(p.TypesInfo.TypeOf(y)) {
		return false
	}
	if p.constant(x) {
		x, y, op = y, x, mirrors[op]
	}
	if p.constant(x) || !p.constant(y) {
		return false
	}
	var fixes []analysis.SuggestedFix
	t := p.TypesInfo.TypeOf(x).Underlying().(*types.Basic)
	if text, ok := p.limit(x, bound{y, op}); ok && t.Info()&types.IsInteger != 0 {
		low, high := "-1<<63", "1<<63"
		if t.Info()&types.IsUnsigned != 0 {
			low, high = "0", "1<<64"
		}
		if lower(op) {
			low = text
		} else {
			high = text
		}
		fixes = c.rewrite("InRange", p.source(x)+", "+low+", "+high)
	}
	p.report(c.node, "order", "InRange", fixes)
	return true
}

// numeric reports whether t is an integer or floating-point type.
func numeric(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&(types.IsInteger|types.IsFloat) != 0
}

// closeTo reports a check that the difference of two numbers is within a
// tolerance, math.Abs(a-b) <= tol or < tol. CloseTo states it, and passes a
// difference equal to the tolerance and fails a NaN, as <= does. The rule
// suggests CloseTo(t, a, b, tol, msg) for <=, with a constant operand as the
// value that the test wants.
func closeTo(p *pass, c check) bool {
	x, y, op, ok := c.comparison()
	if op == token.GEQ || op == token.GTR {
		x, y, op = y, x, mirrors[op]
	}
	abs, isAbs := p.callOf(x, "math", "Abs")
	if !ok || !isAbs || op != token.LEQ && op != token.LSS {
		return false
	}
	diff, ok := ast.Unparen(abs.Args[0]).(*ast.BinaryExpr)
	if !ok || diff.Op != token.SUB {
		return false
	}
	var fixes []analysis.SuggestedFix
	if op == token.LEQ {
		got, want := diff.X, diff.Y
		if p.constant(got) {
			got, want = want, got
		}
		fixes = c.rewrite("CloseTo", p.source(got)+", "+p.source(want)+", "+p.source(y))
	}
	p.report(c.node, "close-to", "CloseTo", fixes)
	return true
}

// inRange reports a check that two orders bound one number from both sides:
// True of lo <= x && x <= hi, or False of x < lo || x > hi, in any order of
// each comparison's operands. InRange states it, and the rule suggests
// InRange(t, x, lo, hi, msg) where both bounds are exact: see limit.
func inRange(p *pass, c check) bool {
	chain := token.LAND
	if !c.holds {
		chain = token.LOR
	}
	terms := operands(c.cond, chain)
	if len(terms) != 2 {
		return false
	}
	first, isFirst := terms[0].(*ast.BinaryExpr)
	second, isSecond := terms[1].(*ast.BinaryExpr)
	if !isFirst || !isSecond {
		return false
	}
	for _, x := range []ast.Expr{first.X, first.Y} {
		a, isA := p.orient(first, x, !c.holds)
		b, isB := p.orient(second, x, !c.holds)
		if !isA || !isB || lower(a.op) == lower(b.op) || !numeric(p.TypesInfo.TypeOf(x)) {
			continue
		}
		if !lower(a.op) {
			a, b = b, a
		}
		var fixes []analysis.SuggestedFix
		low, isLow := p.limit(x, a)
		high, isHigh := p.limit(x, b)
		if isLow && isHigh && pure(x) {
			fixes = c.rewrite("InRange", p.source(x)+", "+low+", "+high)
		}
		p.report(c.node, "in-range", "InRange", fixes)
		return true
	}
	return false
}

// bound is a bound of a number: x op value.
type bound struct {
	value ast.Expr
	op    token.Token
}

// orient returns the order b as a bound of x, x op value, with op negated
// where negate is true, and false where b is no order of x.
func (p *pass) orient(b *ast.BinaryExpr, x ast.Expr, negate bool) (bound, bool) {
	op := b.Op
	if negate {
		op = negations[op]
	}
	order := op == token.LSS || op == token.LEQ || op == token.GTR || op == token.GEQ
	switch text := p.source(x); {
	case !order:
		return bound{}, false
	case p.source(b.X) == text:
		return bound{b.Y, op}, true
	case p.source(b.Y) == text:
		return bound{b.X, mirrors[op]}, true
	}
	return bound{}, false
}

// lower reports whether a bound of the comparison op bounds a number from
// below.
func lower(op token.Token) bool {
	return op == token.GTR || op == token.GEQ
}

// limit returns the text of the closed bound of InRange that equals the
// bound b of x, and false where InRange cannot state b exactly. InRange reads
// a number as a float64.
//
// A float64 x takes a bound of <= or >= whose value is a constant or a
// float64, unchanged. An integer x takes a constant whose bound, moved by one
// for < and >, is of a magnitude up to 2^53, which a float64 represents
// exactly. A literal bound becomes the number of the closed bound. Any other
// bound keeps its source text, with +1 or -1 after it for > and <, so that the
// rewritten check changes with a named constant. A typed bound gets a
// conversion to float64.
func (p *pass) limit(x ast.Expr, b bound) (string, bool) {
	t := p.TypesInfo.TypeOf(x).Underlying().(*types.Basic)
	strict := b.op == token.LSS || b.op == token.GTR
	if t.Kind() == types.Float64 {
		valueType := p.TypesInfo.TypeOf(b.value).Underlying().(*types.Basic)
		return p.source(b.value), !strict && pure(b.value) && (p.constant(b.value) || valueType.Kind() == types.Float64)
	}
	value := p.TypesInfo.Types[b.value].Value
	if t.Info()&types.IsInteger == 0 || value == nil {
		return "", false
	}
	v, ok := constant.Int64Val(constant.ToInt(value))
	text := p.source(b.value)
	switch b.op {
	case token.GTR:
		v, text = v+1, text+"+1"
	case token.LSS:
		v, text = v-1, text+"-1"
	default:
		// An inclusive bound, >= or <=, is the value itself.
	}
	switch {
	case literal(b.value):
		text = strconv.FormatInt(v, 10)
	case !p.untyped(b.value):
		text = "float64(" + text + ")"
	}
	return text, ok && -exact <= v && v <= exact
}

// literal reports whether e is a number as written, with at most one unary
// operator, such as 10 or -1.
func literal(e ast.Expr) bool {
	if u, ok := ast.Unparen(e).(*ast.UnaryExpr); ok {
		e = u.X
	}
	_, ok := ast.Unparen(e).(*ast.BasicLit)
	return ok
}

// untyped reports whether the constant expression e is untyped as it is
// written: literals and untyped constants joined by operators, which a
// float64 parameter takes without a conversion. A call, such as len of a
// constant string, has a type.
func (p *pass) untyped(e ast.Expr) bool {
	untyped := true
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			untyped = false
		case *ast.Ident:
			if c, isConst := p.TypesInfo.Uses[n].(*types.Const); isConst {
				if b, isBasic := c.Type().(*types.Basic); !isBasic || b.Info()&types.IsUntyped == 0 {
					untyped = false
				}
			}
		}
		return untyped
	})
	return untyped
}
