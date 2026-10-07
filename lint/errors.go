// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// errorsIs reports True or False of errors.Is, and suggests ErrorIs or
// ErrorIsNot, which match an error as errors.Is matches it.
func errorsIs(p *pass, c check) bool {
	n, ok := p.callOf(c.cond, "errors", "Is")
	if !ok {
		return false
	}
	name := "ErrorIs"
	if !c.holds {
		name = "ErrorIsNot"
	}
	p.report(c.node, "errors-is", name, c.rewrite(name, p.source(n.Args[0])+", "+p.source(n.Args[1])))
	return true
}

// errorsAs reports True of errors.As, and True of a variable that a call of
// errors.As assigns. ErrorAs states the check and returns the error that it
// finds.
//
// It suggests target = assert.ErrorAs[T](t, err, msg) for a statement of the
// surface that stops the test, where the file can write the target's type T,
// and where a statement reads the target after the fix assigns it. ErrorAs
// returns the zero T on a failure, where errors.As leaves the target as it
// was, and a failure of that surface ends the test before any statement
// reads the target.
func errorsAs(p *pass, c check) bool {
	n, isCall := p.callOf(c.cond, "errors", "As")
	matched := n
	if !isCall {
		matched = p.assignedBy(p.variable(c.cond), "errors.As")
	}
	if !c.holds || matched == nil {
		return false
	}
	var fixes []analysis.SuggestedFix
	if isCall && c.stmt && c.name != nil && c.surface(assertPath) {
		if amp, ok := ast.Unparen(n.Args[1]).(*ast.UnaryExpr); ok && amp.Op == token.AND && p.readElsewhere(amp.X) {
			if text, ok := p.spell(n.Pos(), p.TypesInfo.TypeOf(amp.X)); ok {
				call := p.source(amp.X) + " = " + c.qualifier + ".ErrorAs[" + text + "](" +
					p.source(c.tb) + ", " + p.source(n.Args[0]) + ", " + p.source(c.call.Args[2]) + ")"
				fixes = replace("Call ErrorAs", c.call, call)
			}
		}
	}
	p.report(c.node, "errors-as", "ErrorAs of "+p.brief(matched), fixes)
	return true
}

// readElsewhere reports whether a statement reads the value of e besides e
// itself: e is no local variable, which a fix may assign without a read, or
// another identifier reads the variable. A local variable that a fix assigns
// and nothing reads fails to compile.
func (p *pass) readElsewhere(e ast.Expr) bool {
	v := p.variable(e)
	if v == nil || v.Parent() == v.Pkg().Scope() {
		return true
	}
	for id, obj := range p.TypesInfo.Uses {
		if obj == v && id != ast.Unparen(e) {
			return true
		}
	}
	return false
}

// assignedBy returns the call of the function name, by full name, that
// assigns the variable v, and nil where none does.
func (p *pass) assignedBy(v *types.Var, name string) *ast.CallExpr {
	for _, n := range p.origins[v] {
		if p.callee(n) == name {
			return n
		}
	}
	return nil
}

// sentinel reports an equality of two errors, one of them a package-level
// variable, which ErrorIs or ErrorIsNot states through errors.Is. It suggests
// no fix: == compares the errors themselves, and errors.Is also matches a
// sentinel that another error wraps. An equality of two package-level
// variables checks that they are one error, which no match states.
func sentinel(p *pass, c check) bool {
	e, ok := p.equality(c)
	if !ok || !p.isErrorValue(e.x) || !p.isErrorValue(e.y) || p.global(e.x) == p.global(e.y) {
		return false
	}
	name := "ErrorIs"
	if !e.equal {
		name = "ErrorIsNot"
	}
	p.report(c.node, "sentinel", name, nil)
	return true
}

// match returns the error and the target of a check that states that an
// error matches a target: True of errors.Is, ErrorIs, and an equality of two
// errors. The target is the operand that is a package-level variable.
func (p *pass) match(c check) (err, target ast.Expr, ok bool) {
	if n, isIs := p.callOf(c.cond, "errors", "Is"); isIs && c.holds {
		return n.Args[0], n.Args[1], true
	}
	if c.is("ErrorIs") {
		return c.call.Args[1], c.call.Args[2], true
	}
	e, ok := p.equality(c)
	if !ok || !e.equal || !p.isErrorValue(e.x) || !p.isErrorValue(e.y) {
		return nil, nil, false
	}
	if p.global(e.x) {
		return e.y, e.x, true
	}
	return e.x, e.y, true
}

// isErrorValue reports whether e is a value of a type that implements error.
func (p *pass) isErrorValue(e ast.Expr) bool {
	t := p.TypesInfo.TypeOf(e)
	return t != nil && types.Implements(t, errorType)
}

// global reports whether e names a package-level variable.
func (p *pass) global(e ast.Expr) bool {
	v := p.object(e)
	_, isVar := v.(*types.Var)
	return isVar && v.Pkg() != nil && v.Parent() == v.Pkg().Scope()
}

// object returns the object that e names, through an identifier or a
// qualified identifier, and nil where e names none.
func (p *pass) object(e ast.Expr) types.Object {
	switch e := ast.Unparen(e).(type) {
	case *ast.Ident:
		return p.TypesInfo.ObjectOf(e)
	case *ast.SelectorExpr:
		return p.TypesInfo.ObjectOf(e.Sel)
	}
	return nil
}

// refers reports whether e names the package-level object name of the
// package path, such as context.Canceled.
func (p *pass) refers(e ast.Expr, path, name string) bool {
	obj := p.object(e)
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == path && obj.Name() == name
}
