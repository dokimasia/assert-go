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
	"golang.org/x/tools/go/types/typeutil"
)

// reading is a call whose result a check compares, and the statement that
// makes the call.
type reading struct {
	call *ast.CallExpr
	at   inspector.Cursor
}

// grouped is the function and the three operands of a grouping of two calls
// of the function, as f(f(a, b), c).
type grouped struct {
	call       *ast.CallExpr
	f, a, b, c ast.Expr
}

// commutative reports an equality of f(a, b) and f(b, a). Commutative states
// it, and suggests Commutative(t, f, a, b, msg) for a call of Equal where f
// is a function value and no operand calls a function: Commutative calls f
// once for each order, and Equal evaluates each operand twice.
func commutative(p *pass, c check) bool {
	e, ok := p.equality(c)
	x, isX := ast.Unparen(e.x).(*ast.CallExpr)
	y, isY := ast.Unparen(e.y).(*ast.CallExpr)
	if !ok || !e.equal || !isX || !isY || len(x.Args) != 2 || len(y.Args) != 2 {
		return false
	}
	f, a, b := p.source(x.Fun), p.source(x.Args[0]), p.source(x.Args[1])
	if f != p.source(y.Fun) || a == b || a != p.source(y.Args[1]) || b != p.source(y.Args[0]) {
		return false
	}
	var fixes []analysis.SuggestedFix
	if c.is("Equal") && p.combinable(x) && pure(x.Fun) && pure(x.Args[0]) && pure(x.Args[1]) {
		fixes = c.rewrite("Commutative", f+", "+a+", "+b)
	}
	p.report(c.node, "commutative", "Commutative", fixes)
	return true
}

// associative reports an equality of f(f(a, b), c) and f(a, f(b, c)).
// Associative states it, and suggests Associative(t, f, a, b, c, msg) for a
// call of Equal whose first value is the left grouping, under the conditions
// of commutative.
func associative(p *pass, c check) bool {
	e, ok := p.equality(c)
	l, isLeft := p.grouping(e.x, true)
	r, isRight := p.grouping(e.y, false)
	first := isLeft && isRight
	if !first {
		l, isLeft = p.grouping(e.y, true)
		r, isRight = p.grouping(e.x, false)
	}
	if !ok || !e.equal || !isLeft || !isRight || p.arguments(l) != p.arguments(r) {
		return false
	}
	var fixes []analysis.SuggestedFix
	if c.is("Equal") && first && p.combinable(l.call) && pure(l.f) && pure(l.a) && pure(l.b) && pure(l.c) {
		fixes = c.rewrite("Associative", p.arguments(l))
	}
	p.report(c.node, "associative", "Associative", fixes)
	return true
}

// arguments returns the arguments of Associative that state the grouping g.
func (p *pass) arguments(g grouped) string {
	return p.source(g.f) + ", " + p.source(g.a) + ", " + p.source(g.b) + ", " + p.source(g.c)
}

// grouping returns e as f(f(a, b), c) where left is true, and as f(a, f(b,
// c)) otherwise, and false where e is no such grouping.
func (p *pass) grouping(e ast.Expr, left bool) (grouped, bool) {
	outer, ok := ast.Unparen(e).(*ast.CallExpr)
	if !ok || len(outer.Args) != 2 {
		return grouped{}, false
	}
	nested := outer.Args[1]
	if left {
		nested = outer.Args[0]
	}
	inner, ok := ast.Unparen(nested).(*ast.CallExpr)
	if !ok || len(inner.Args) != 2 || p.source(inner.Fun) != p.source(outer.Fun) {
		return grouped{}, false
	}
	if left {
		return grouped{outer, outer.Fun, inner.Args[0], inner.Args[1], outer.Args[1]}, true
	}
	return grouped{outer, outer.Fun, outer.Args[0], inner.Args[0], inner.Args[1]}, true
}

// combinable reports whether the function that the call n calls can be
// passed as a value: a function or a method without type parameters, or a
// variable.
func (p *pass) combinable(n *ast.CallExpr) bool {
	switch callee := typeutil.Callee(p.TypesInfo, n).(type) {
	case *types.Func:
		return callee.Signature().TypeParams().Len() == 0
	case *types.Var:
		return true
	}
	return false
}

// roundTrip reports an equality of a value x and g(f(x)), where each call is
// in the check or in an earlier statement of its list, as json.Unmarshal of
// the output of json.Marshal. RoundTrip states it.
func roundTrip(p *pass, c check) bool {
	e, ok := p.equality(c)
	stmt, listed := c.statement()
	if !ok || !e.equal || !listed {
		return false
	}
	f, g, inverts := p.inverts(stmt, e.x, e.y)
	if !inverts {
		f, g, inverts = p.inverts(stmt, e.y, e.x)
	}
	if !inverts {
		return false
	}
	p.report(c.node, "round-trip", "RoundTrip of "+p.brief(f.Fun)+" and "+p.brief(g.Fun), nil)
	return true
}

// inverts returns the conversions f and g where back, for the statement at
// stmt, is the result of g of the result of f of input, and false where it
// is no such result. Each call is in the statement or in an earlier
// statement of its list. RoundTrip makes both calls in one step, so every
// other statement from the call of f up to stmt is inert, and no statement
// but the call of g reads the result of f. A builtin, such as len, converts
// nothing, a constant input makes no round trip, and a constructor of a
// test's fixture is no conversion of the code under test.
func (p *pass) inverts(stmt inspector.Cursor, input, back ast.Expr) (f, g *ast.CallExpr, ok bool) {
	g, at, ok := p.localSource(stmt, back)
	if !ok || p.builtin(g) || p.constant(input) {
		return nil, nil, false
	}
	want := p.source(input)
	for _, arg := range g.Args {
		f, from, ok := p.localSource(at, arg)
		if !ok || p.builtin(f) || p.constructor(f) || !slices.ContainsFunc(f.Args, func(in ast.Expr) bool {
			return p.source(in) == want
		}) {
			continue
		}
		if p.direct(from, at, stmt) && !p.readAfter(from, g, p.variable(p.operand(arg))) {
			return f, g, true
		}
	}
	return nil, nil, false
}

// direct reports whether every statement after the statement at from and
// before the statement at to, other than the statement at skip, is inert.
// from is to or a statement before it in one list.
func (p *pass) direct(from, skip, to inspector.Cursor) bool {
	if from == to {
		return true
	}
	for s, ok := from.NextSibling(); ok && s != to; s, ok = s.NextSibling() {
		if s != skip && !p.inert(s.Node()) {
			return false
		}
	}
	return true
}

// readAfter reports whether a statement after the statement at from reads the
// variable v outside the call n. A nil v is read nowhere.
func (p *pass) readAfter(from inspector.Cursor, n *ast.CallExpr, v *types.Var) bool {
	for s, ok := from.NextSibling(); ok && v != nil; s, ok = s.NextSibling() {
		if p.reads(s.Node(), n, v) {
			return true
		}
	}
	return false
}

// constructor reports whether the call n calls a function of a test file that
// takes no test, as a constructor of a test's fixture does.
func (p *pass) constructor(n *ast.CallExpr) bool {
	fn, ok := typeutil.Callee(p.TypesInfo, n).(*types.Func)
	if !ok || !strings.HasSuffix(p.Fset.Position(fn.Pos()).Filename, "_test.go") {
		return false
	}
	for param := range fn.Signature().Params().Variables() {
		if types.NewMethodSet(param.Type()).Lookup(nil, "Helper") != nil {
			return false
		}
	}
	return true
}

// builtin reports whether the call n calls a builtin function.
func (p *pass) builtin(n *ast.CallExpr) bool {
	_, ok := typeutil.Callee(p.TypesInfo, n).(*types.Builtin)
	return ok
}

// repetition reports an equality of the results of two calls of one function
// with one input. Two consecutive calls state Deterministic, or StableOrder
// for a function without parameters that returns a slice other than bytes.
// Calls with statements between them state Pure, or NotPure where the
// results differ, and Idempotent where one statement repeats before each
// call. Two calls are consecutive where only still statements of the first
// call are between them. A result that a later statement assigns is no
// result of its call. Where a later statement passes the first result to a
// call that can write it, Pure and NotPure observe a copy of the result
// around the statements up to the check. Two errors are no results that the
// assertions compare: Deterministic fails on an error, and Pure reads state
// before and after a call.
func repetition(p *pass, c check) bool {
	e, ok := p.equality(c)
	stmt, listed := c.statement()
	if !ok || !listed || p.isErrorValue(e.x) || p.isErrorValue(e.y) {
		return false
	}
	first, isFirst := p.reading(stmt, e.x)
	second, isSecond := p.reading(stmt, e.y)
	if !isFirst || !isSecond || p.source(first.call) != p.source(second.call) ||
		p.changed(first.at, stmt, e.x) || p.changed(second.at, stmt, e.y) {
		return false
	}
	firstValue, secondValue := e.x, e.y
	if second.at.Index() < first.at.Index() {
		first, second = second, first
		firstValue, secondValue = secondValue, firstValue
	}
	if p.passed(second.at, stmt, secondValue) {
		return false
	}
	if p.passed(first.at, stmt, firstValue) {
		p.reportCopy(c, e.equal, firstValue, first.at, second.at, stmt)
		return true
	}
	quiet := func(s ast.Node) bool { return p.still(s, first.call) }
	rule, name := "pure", "Pure"
	next, _ := following(first.at, stmt, quiet)
	switch {
	case first.at == second.at || next == second.at:
		rule, name = "deterministic", "Deterministic"
		if p.listing(first.call) {
			rule, name = "stable-order", "StableOrder"
		}
	case p.repeats(first.at, second.at, quiet):
		rule, name = "idempotent", "Idempotent"
	case !e.equal:
		rule, name = "not-pure", "NotPure"
	}
	if !e.equal && rule != "not-pure" {
		return false
	}
	matched := name + " of " + p.brief(first.call)
	switch rule {
	case "pure", "not-pure":
		matched += ", around " + p.steps(first.at, second.at, second.at, quiet)
	case "idempotent":
		repeated, _ := preceding(first.at, quiet)
		matched = name + " of " + p.brief(repeated.Node()) + ", read by " + p.brief(first.call)
	}
	p.report(c.node, rule, matched, nil)
	return true
}

// reportCopy reports an equality of value, a variable that a statement
// between the reading at from and the check at stmt passes to a call that
// can write it, and a second reading at second of value's call. Pure states
// the check with an observation that returns a copy of value, around the
// statements up to the check, and NotPure where the readings differ.
func (p *pass) reportCopy(c check, equal bool, value ast.Expr, from, second, stmt inspector.Cursor) {
	rule, name := "not-pure", "NotPure"
	if equal {
		rule, name = "pure", "Pure"
	}
	quiet := func(s ast.Node) bool { return p.still(s, value) }
	p.report(c.node, rule, name+" of a copy of "+p.source(value)+", around "+p.steps(from, stmt, second, quiet), nil)
}

// steps returns the text of the statements after the statement at from and
// before the statement at to, other than the statement at skip, that quiet
// does not accept, joined by semicolons. from is before to in one list.
func (p *pass) steps(from, to, skip inspector.Cursor, quiet func(ast.Node) bool) string {
	var texts []string
	for s, _ := from.NextSibling(); s != to; s, _ = s.NextSibling() {
		if s != skip && !quiet(s.Node()) {
			texts = append(texts, p.brief(s.Node()))
		}
	}
	return strings.Join(texts, "; ")
}

// still reports whether the statement s leaves what observed reads as it
// is, as the rules over two readings count it: a declaration whose
// variables take no values, or an assertion whose values make no call that
// can write a variable that observed reads. Such a call is a method of the
// variable, or receives the variable's address or a value of it that shares
// memory, as passes states. A call of observed's own function changes
// nothing, and a test, which every assertion and helper takes, is no such
// variable.
func (p *pass) still(s ast.Node, observed ast.Expr) bool {
	c, isAssertion := p.assertionStatement(s)
	if !isAssertion {
		return p.inert(s)
	}
	read := p.observed(observed)
	own := ""
	if n, isCall := ast.Unparen(observed).(*ast.CallExpr); isCall {
		own = p.source(n.Fun)
	}
	return !slices.ContainsFunc(c.values(), func(value ast.Expr) bool {
		touched := false
		ast.Inspect(value, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !touched && isCall && p.source(call.Fun) != own && !p.inspects(call) {
				sel, isMethod := call.Fun.(*ast.SelectorExpr)
				touched = isMethod && p.readsAny(sel.X, read) ||
					slices.ContainsFunc(call.Args, func(arg ast.Expr) bool {
						return p.readsAny(arg, read) && shares(p.TypesInfo.TypeOf(arg))
					})
			}
			return !touched
		})
		return touched
	})
}

// isAddress reports whether e takes the address of an operand, as &v does.
func isAddress(e ast.Expr) bool {
	u, ok := ast.Unparen(e).(*ast.UnaryExpr)
	return ok && u.Op == token.AND
}

// inspects reports whether the call n reads its arguments and writes none:
// a call of an assertion, or of a method of a test, such as t.Logf.
func (p *pass) inspects(n *ast.CallExpr) bool {
	if _, isAssertion := p.assertion(n); isAssertion {
		return true
	}
	sel, _ := n.Fun.(*ast.SelectorExpr)
	selection, isSelection := p.TypesInfo.Selections[sel]
	return isSelection && types.NewMethodSet(selection.Recv()).Lookup(nil, "Helper") != nil
}

// observed returns the variables that the expression e reads, other than a
// test: a value whose methods include Helper.
func (p *pass) observed(e ast.Expr) map[*types.Var]bool {
	read := make(map[*types.Var]bool)
	ast.Inspect(e, func(node ast.Node) bool {
		if id, isIdent := node.(*ast.Ident); isIdent {
			if v, isVar := p.TypesInfo.Uses[id].(*types.Var); isVar &&
				types.NewMethodSet(v.Type()).Lookup(nil, "Helper") == nil {
				read[v] = true
			}
		}
		return true
	})
	return read
}

// readsAny reports whether the node n reads one of the variables vars.
func (p *pass) readsAny(n ast.Node, vars map[*types.Var]bool) bool {
	found := false
	ast.Inspect(n, func(node ast.Node) bool {
		id, _ := node.(*ast.Ident)
		v, _ := p.TypesInfo.Uses[id].(*types.Var)
		found = found || vars[v]
		return !found
	})
	return found
}

// passed reports whether a statement after the statement at from and before
// the statement at to passes the variable that e names to a call that can
// write it, and false where e names no variable. from is before to in one
// list where e names a variable, because the reading of a variable is the
// statement that assigns it.
func (p *pass) passed(from, to inspector.Cursor, e ast.Expr) bool {
	v := p.variable(e)
	if v == nil {
		return false
	}
	for s, _ := from.NextSibling(); s != to; s, _ = s.NextSibling() {
		if p.passes(s.Node(), v) {
			return true
		}
	}
	return false
}

// passes reports whether the node n makes a call that can write the variable
// v: a call that receives the address of v or of a part of it, a part of v
// whose value shares memory, as a slice, a map or a pointer does, or v as
// the receiver of a method that takes a pointer or of a value that shares
// memory. A builtin writes through clear and copy alone, which changes
// counts as assignments, a conversion writes nothing, and neither does a
// call that inspects its arguments.
func (p *pass) passes(n ast.Node, v *types.Var) bool {
	found := false
	ast.Inspect(n, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !found && isCall && p.computes(call) && !p.inspects(call) {
			found = p.writesReceiver(call, v) || slices.ContainsFunc(call.Args, func(arg ast.Expr) bool {
				if isAddress(arg) {
					return p.root(ast.Unparen(arg).(*ast.UnaryExpr).X) == v
				}
				return p.root(arg) == v && shares(p.TypesInfo.TypeOf(arg))
			})
		}
		return !found
	})
	return found
}

// writesReceiver reports whether the call n is a call of a method or of a
// function field whose receiver is v or a part of v, where a method takes a
// pointer or the receiver shares memory, so that the call can write v.
func (p *pass) writesReceiver(n *ast.CallExpr, v *types.Var) bool {
	sel, isSelector := n.Fun.(*ast.SelectorExpr)
	if !isSelector || p.root(sel.X) != v {
		return false
	}
	byPointer := false
	if method, isMethod := p.TypesInfo.Selections[sel].Obj().(*types.Func); isMethod {
		_, byPointer = method.Signature().Recv().Type().(*types.Pointer)
	}
	return byPointer || shares(p.TypesInfo.TypeOf(sel.X))
}

// shares reports whether a value of type t refers to memory that a copy of
// the value shares: a slice, a map, a pointer, a channel or an interface, or
// an array or a struct that contains one.
func shares(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Slice, *types.Map, *types.Pointer, *types.Chan, *types.Interface:
		return true
	case *types.Array:
		return shares(u.Elem())
	case *types.Struct:
		for field := range u.Fields() {
			if shares(field.Type()) {
				return true
			}
		}
	}
	return false
}

// reading returns the call whose result e is: e where it is a call, and the
// call of the nearest statement before stmt that assigns e's variable. A
// conversion and a call of a builtin, such as make, read nothing.
func (p *pass) reading(stmt inspector.Cursor, e ast.Expr) (reading, bool) {
	if n, ok := ast.Unparen(e).(*ast.CallExpr); ok {
		return reading{n, stmt}, p.computes(n)
	}
	s, ok := p.assignment(stmt, p.variable(e))
	if !ok || len(s.Node().(*ast.AssignStmt).Rhs) != 1 {
		return reading{}, false
	}
	n, ok := ast.Unparen(s.Node().(*ast.AssignStmt).Rhs[0]).(*ast.CallExpr)
	return reading{n, s}, ok && p.computes(n)
}

// computes reports whether the call n calls a function: no conversion and no
// builtin.
func (p *pass) computes(n *ast.CallExpr) bool {
	return !p.TypesInfo.Types[n.Fun].IsType() && !p.builtin(n)
}

// changed reports whether a statement after the statement at from and before
// the statement at to changes the variable that e names, or an element or a
// field of it. from is before to in one list.
func (p *pass) changed(from, to inspector.Cursor, e ast.Expr) bool {
	v := p.variable(e)
	if v == nil {
		return false
	}
	for s, ok := from.NextSibling(); ok && s != to; s, ok = s.NextSibling() {
		if p.changes(s.Node(), v) {
			return true
		}
	}
	return false
}

// changes reports whether the statement s assigns the variable v or an
// element or a field of it, increments or decrements one, or clears or
// copies into v.
func (p *pass) changes(s ast.Node, v *types.Var) bool {
	found := false
	ast.Inspect(s, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			found = found || slices.ContainsFunc(n.Lhs, func(lhs ast.Expr) bool { return p.root(lhs) == v })
		case *ast.IncDecStmt:
			found = found || p.root(n.X) == v
		case *ast.CallExpr:
			b, ok := typeutil.Callee(p.TypesInfo, n).(*types.Builtin)
			found = found || ok && (b.Name() == "clear" || b.Name() == "copy") && p.root(n.Args[0]) == v
		}
		return !found
	})
	return found
}

// root returns the variable at the root of e, through indexes and fields, as
// v of v[i].f, and nil where e has no variable at its root.
func (p *pass) root(e ast.Expr) *types.Var {
	for {
		switch x := ast.Unparen(e).(type) {
		case *ast.IndexExpr:
			e = x.X
		case *ast.SelectorExpr:
			e = x.X
		default:
			return p.variable(x)
		}
	}
}

// following returns the statement after the statement at cursor in its
// list, past the statements that quiet accepts, up to the statement at stop,
// and false at the end of the list.
func following(cursor, stop inspector.Cursor, quiet func(ast.Node) bool) (inspector.Cursor, bool) {
	next, ok := cursor.NextSibling()
	for ok && next != stop && quiet(next.Node()) {
		next, ok = next.NextSibling()
	}
	return next, ok
}

// preceding returns the statement before the statement at cursor in its
// list, past the statements that quiet accepts, and false at the start of
// the list.
func preceding(cursor inspector.Cursor, quiet func(ast.Node) bool) (inspector.Cursor, bool) {
	prev, ok := cursor.PrevSibling()
	for ok && quiet(prev.Node()) {
		prev, ok = prev.PrevSibling()
	}
	return prev, ok
}

// assignment returns the nearest statement before stmt that assigns the
// variable v, and false where none does.
func (p *pass) assignment(stmt inspector.Cursor, v *types.Var) (inspector.Cursor, bool) {
	for _, s := range before(stmt) {
		if assign, ok := s.Node().(*ast.AssignStmt); ok && p.assigns(assign, v) {
			return s, true
		}
	}
	return stmt, false
}

// assigns reports whether the assignment assigns the variable v.
func (p *pass) assigns(assign *ast.AssignStmt, v *types.Var) bool {
	for _, lhs := range assign.Lhs {
		if v != nil && p.variable(lhs) == v {
			return true
		}
	}
	return false
}

// listing reports whether the call n calls a function without parameters
// whose first result is a slice, as an iteration of a subject's elements. A
// slice of bytes is a value, such as a digest, and no listing.
func (p *pass) listing(n *ast.CallExpr) bool {
	signature, ok := p.TypesInfo.TypeOf(n.Fun).(*types.Signature)
	if !ok || signature.Params().Len() != 0 || signature.Results().Len() == 0 {
		return false
	}
	result := signature.Results().At(0).Type()
	_, isSlice := result.Underlying().(*types.Slice)
	return isSlice && !isBytes(result)
}

// repeats reports whether the statements right before the two readings at
// first and second are one statement, which runs before each reading. The
// statements that quiet accepts between a statement and its reading do not
// count.
func (p *pass) repeats(first, second inspector.Cursor, quiet func(ast.Node) bool) bool {
	a, okA := preceding(first, quiet)
	b, okB := preceding(second, quiet)
	return okA && okB && b != first && listed(a) && listed(b) && p.source(a.Node()) == p.source(b.Node())
}

// afterClose reports a match of an error with a sentinel, where the error
// comes from a method of a value whose Close an earlier statement of the
// list called, and only inert statements are between the two.
// FailsAfterClose states it: it calls the close function, then the call, and
// matches the error.
func afterClose(p *pass, c check) bool {
	err, _, ok := p.match(c)
	stmt, listed := c.statement()
	if !ok || !listed {
		return false
	}
	n, at, ok := p.localSource(stmt, err)
	if !ok {
		return false
	}
	method, ok := n.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	for _, s := range before(at) {
		if closed := p.closes(s.Node(), method.X); closed != nil {
			p.report(c.node, "after-close", "FailsAfterClose of "+p.brief(closed)+" and "+p.brief(n), nil)
			return true
		}
		if !p.inert(s.Node()) {
			return false
		}
	}
	return false
}

// closes returns the call of Close without arguments on the receiver in the
// statement s, outside a deferred call and a function literal, and nil where
// s makes no such call.
func (p *pass) closes(s ast.Node, receiver ast.Expr) *ast.CallExpr {
	want := p.source(receiver)
	var found *ast.CallExpr
	ast.Inspect(s, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.DeferStmt, *ast.FuncLit:
			return false
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Close" && len(n.Args) == 0 &&
				p.source(sel.X) == want {
				found = n
			}
		}
		return found == nil
	})
	return found
}

// total reports a range over a slice whose body is one NoError of
// f(element), where f does not read the element. Total states the check with
// f itself. A loop whose call needs a function literal to become f states
// its check as plainly as Total does, and the rule leaves it out.
//
// It suggests Total(t, f, xs, msg) for a statement of the surface that stops
// the test, where evaluating f calls no function and the message does not
// read the element: the loop stops at the first error, as Total does, and
// evaluates f once per element. The surface that records a failure continues
// the loop, and Total reports the first error alone. The record of Total
// states the index and the error of the element that fails, and the fix
// would use the element outside the loop that declares it.
func total(p *pass, cursor inspector.Cursor) {
	n := cursor.Node().(*ast.RangeStmt)
	element := p.variable(n.Value)
	_, slice := p.TypesInfo.TypeOf(n.X).Underlying().(*types.Slice)
	if element == nil || !slice || len(n.Body.List) != 1 {
		return
	}
	c, ok := p.assertionStatement(n.Body.List[0])
	if !ok || !c.is("NoError") {
		return
	}
	subject, ok := c.call.Args[1].(*ast.CallExpr)
	if !ok || len(subject.Args) != 1 || p.variable(subject.Args[0]) != element || p.reads(subject.Fun, nil, element) {
		return
	}
	var fixes []analysis.SuggestedFix
	if c.name != nil && c.surface(assertPath) && pure(subject.Fun) && !p.reads(c.call.Args[2], nil, element) {
		fixes = replace("Call Total", n, c.qualifier+".Total("+p.source(c.tb)+", "+p.source(subject.Fun)+", "+
			p.source(n.X)+", "+p.source(c.call.Args[2])+")")
	}
	p.report(n, "total", "Total", fixes)
}

// reads reports whether the node n reads the variable v outside the node
// skip, which is nil where n has no part to leave out.
func (p *pass) reads(n, skip ast.Node, v *types.Var) bool {
	found := false
	ast.Inspect(n, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		found = found || ok && p.TypesInfo.Uses[id] == v
		return !found && node != skip
	})
	return found
}

// poisoned reports a counted loop whose body is one HasError of a call: a
// for statement with a condition, or a range over an integer. Poisoned
// states the check: after a failure is induced, each reading fails.
func poisoned(p *pass, cursor inspector.Cursor) {
	body, counted := p.countedLoop(cursor.Node())
	if !counted || len(body.List) != 1 {
		return
	}
	c, ok := p.assertionStatement(body.List[0])
	if !ok || !c.is("HasError") {
		return
	}
	if _, isCall := c.call.Args[1].(*ast.CallExpr); isCall {
		p.report(cursor.Node(), "poisoned", "Poisoned", nil)
	}
}

// countedLoop returns the body of a counted loop: a for statement with a
// condition, or a range over an integer. It returns false for any other loop.
func (p *pass) countedLoop(n ast.Node) (*ast.BlockStmt, bool) {
	if loop, ok := n.(*ast.ForStmt); ok {
		return loop.Body, loop.Cond != nil
	}
	loop := n.(*ast.RangeStmt)
	b, ok := p.TypesInfo.TypeOf(loop.X).Underlying().(*types.Basic)
	return loop.Body, ok && b.Info()&types.IsInteger != 0
}

// noDuplicates reports a loop whose body checks seen[x], or the ok of its
// lookup, and assigns seen[x], for a map seen. NoDuplicates states the check.
func noDuplicates(p *pass, cursor inspector.Cursor) {
	assigned := make(map[string]bool)
	for inner := range cursor.Preorder((*ast.AssignStmt)(nil)) {
		for _, lhs := range inner.Node().(*ast.AssignStmt).Lhs {
			if index, ok := lhs.(*ast.IndexExpr); ok && isMap(p.TypesInfo.TypeOf(index.X)) {
				assigned[p.source(index)] = true
			}
		}
	}
	for inner := range cursor.Preorder((*ast.CallExpr)(nil), (*ast.IfStmt)(nil)) {
		c, ok := p.check(inner)
		if !ok || c.cond == nil {
			continue
		}
		key := c.cond
		if lookup := p.lookups[p.variable(c.cond)]; lookup != nil {
			key = lookup
		}
		if assigned[p.source(key)] {
			p.report(cursor.Node(), "no-duplicates", "NoDuplicates", nil)
			return
		}
	}
}

// isMap reports whether t is a map type.
func isMap(t types.Type) bool {
	_, ok := t.Underlying().(*types.Map)
	return ok
}

// monotonic reports a loop that checks an order of two variables and keeps
// the one in the other, as prev = cur. Monotonic states the check.
func monotonic(p *pass, cursor inspector.Cursor) {
	kept := make(map[*types.Var]*types.Var)
	for inner := range cursor.Preorder((*ast.AssignStmt)(nil)) {
		assign := inner.Node().(*ast.AssignStmt)
		if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 && assign.Tok == token.ASSIGN {
			kept[p.variable(assign.Lhs[0])] = p.variable(assign.Rhs[0])
		}
	}
	for inner := range cursor.Preorder((*ast.CallExpr)(nil), (*ast.IfStmt)(nil)) {
		c, _ := p.check(inner)
		x, y, op, ok := c.comparison()
		vx, vy := p.variable(x), p.variable(y)
		if ok && op != token.EQL && op != token.NEQ && vx != nil && vy != nil && (kept[vx] == vy || kept[vy] == vx) {
			p.report(cursor.Node(), "monotonic", "Monotonic", nil)
			return
		}
	}
}
