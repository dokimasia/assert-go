// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// propPath is the import path of the package of properties.
const propPath = "go.dokimi.dev/assert/prop"

// randomPaths are the packages whose functions and methods draw random
// values.
var randomPaths = []string{"math/rand", "math/rand/v2"}

// quickFuncs are the functions of testing/quick that check a property.
var quickFuncs = []string{"testing/quick.Check", "testing/quick.CheckEqual"}

// forAll reports a loop that draws a random value through math/rand or
// math/rand/v2 and checks. prop.ForAll states the property: it draws from a
// seed that a failure reports, shrinks a failing case, and replays it.
func forAll(p *pass, cursor inspector.Cursor) {
	random, checks := false, false
	for inner := range cursor.Preorder((*ast.CallExpr)(nil), (*ast.IfStmt)(nil)) {
		_, isCheck := p.check(inner)
		checks = checks || isCheck
		if n, ok := inner.Node().(*ast.CallExpr); ok {
			fn, isFunc := typeutil.Callee(p.TypesInfo, n).(*types.Func)
			random = random || isFunc && fn.Pkg() != nil && slices.Contains(randomPaths, fn.Pkg().Path())
		}
	}
	if random && checks {
		p.report(cursor.Node(), "for-all", "prop.ForAll", nil)
	}
}

// quick reports a call of testing/quick.Check or CheckEqual, which
// prop.ForAll states with generators of the definition, shrinking and
// replay.
func quick(p *pass, cursor inspector.Cursor) {
	n := cursor.Node().(*ast.CallExpr)
	if slices.Contains(quickFuncs, p.callee(n)) {
		p.report(n, "for-all", "prop.ForAll", nil)
	}
}

// propertyForm reports a call of prop.ForAll whose body draws one value and
// then calls one assertion that has a property form, where the form takes
// each argument that reads the drawn value. The form states the property in
// one call, and draws its input from the registry or from the generator of
// prop.Using.
//
// A form passes its input to each function that it takes in place of a
// value or of a function with fewer parameters, as got func(T) U in place of
// got any, and generates each argument that it leaves out, as the input of
// RoundTrip. Every other argument, such as a needle or the observe function
// of prop.Pure, is the same for every input.
func propertyForm(p *pass, cursor inspector.Cursor) {
	body, ok := p.forAllBody(cursor.Node().(*ast.CallExpr))
	if !ok || len(body.List) != 2 || !p.draws(body.List[0]) {
		return
	}
	c, ok := p.assertionStatement(body.List[1])
	if !ok {
		return
	}
	form, ok := p.form(c.fn.Name())
	if !ok {
		return
	}
	drawn := p.variable(body.List[0].(*ast.AssignStmt).Lhs[0])
	params := c.fn.Signature().Params()
	for i := 1; i < params.Len() && params.At(i).Name() != "msg"; i++ {
		taken := counterpart(form.Signature().Params(), params, i)
		if drawn != nil && p.reads(c.call.Args[i], nil, drawn) && taken != nil &&
			arity(taken.Type()) <= arity(params.At(i).Type()) {

			return
		}
	}
	p.report(cursor.Node(), "property-form", "prop."+form.Name(), nil)
}

// counterpart returns the parameter of form that takes the place of the i-th
// parameter of an assertion with the parameters params: the parameter of the
// same name, or the i-th one where params has no parameter of its name. It
// returns nil for a parameter that the form leaves out.
func counterpart(form, params *types.Tuple, i int) *types.Var {
	for v := range form.Variables() {
		if v.Name() == params.At(i).Name() {
			return v
		}
	}
	if i >= form.Len() {
		return nil
	}
	for v := range params.Variables() {
		if v.Name() == form.At(i).Name() {
			return nil
		}
	}
	return form.At(i)
}

// arity returns the number of parameters of the function type t, and -1 for
// a type of no function.
func arity(t types.Type) int {
	signature, ok := t.Underlying().(*types.Signature)
	if !ok {
		return -1
	}
	return signature.Params().Len()
}

// machine reports a call of prop.ForAll whose body loops over actions that
// it draws, through a switch on a drawn value. stateful.Steps states the
// actions as a machine, whose steps shrink and whose history it checks
// against a sequential specification.
func machine(p *pass, cursor inspector.Cursor) {
	if _, ok := p.forAllBody(cursor.Node().(*ast.CallExpr)); !ok {
		return
	}
	for inner := range cursor.Preorder((*ast.SwitchStmt)(nil)) {
		inLoop := false
		for loop := range inner.Enclosing((*ast.ForStmt)(nil), (*ast.RangeStmt)(nil)) {
			inLoop = inLoop || cursor.Contains(loop)
		}
		if tag := inner.Node().(*ast.SwitchStmt).Tag; tag != nil && inLoop && p.drawn(tag) {
			p.report(cursor.Node(), "machine", "stateful.Steps", nil)
			return
		}
	}
}

// forAllBody returns the body of the function literal that a call of
// prop.ForAll runs, and false for any other call.
func (p *pass) forAllBody(n *ast.CallExpr) (*ast.BlockStmt, bool) {
	fn := p.function(n)
	if fn == nil || fn.Pkg().Path() != propPath || fn.Name() != "ForAll" {
		return nil, false
	}
	lit, ok := n.Args[2].(*ast.FuncLit)
	if !ok {
		return nil, false
	}
	return lit.Body, true
}

// draws reports whether the statement s assigns the value of a draw of a
// case.
func (p *pass) draws(s ast.Stmt) bool {
	assign, ok := s.(*ast.AssignStmt)
	return ok && len(assign.Rhs) == 1 && p.drawn(assign.Rhs[0])
}

// drawn reports whether a source of e is a draw of a case.
func (p *pass) drawn(e ast.Expr) bool {
	for _, n := range p.sources(e) {
		if p.callee(n) == "(*"+propPath+".Case).Draw" {
			return true
		}
	}
	return false
}

// form returns the property form of the assertion name: the function of prop
// of that name, or of the name with the prefix Is, whose first parameter is a
// test. It returns false for an assertion without a form.
func (p *pass) form(name string) (*types.Func, bool) {
	for _, pkg := range p.Pkg.Imports() {
		if pkg.Path() != propPath {
			continue
		}
		for _, candidate := range []string{name, "Is" + name} {
			if fn, ok := pkg.Scope().Lookup(candidate).(*types.Func); ok && takesTest(fn) {
				return fn, true
			}
		}
	}
	return nil, false
}
