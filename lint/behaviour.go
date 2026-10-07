// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// derivations are the functions of the package context that derive a context
// and its cancel function.
var derivations = []string{
	"context.WithCancel", "context.WithCancelCause", "context.WithDeadline", "context.WithDeadlineCause",
	"context.WithTimeout", "context.WithTimeoutCause",
}

// honoursCancellation reports a match of an error with context.Canceled,
// where the call that returns the error receives a context that was cancelled
// before the call. HonoursCancellation passes a cancelled context to its
// subject and states the match. A context that ends while the call runs is
// none that HonoursCancellation passes.
func honoursCancellation(p *pass, c check) bool {
	return p.contextMatch(c, "Canceled", "honours-cancellation", "HonoursCancellation")
}

// honoursDeadline reports a match of an error with context.DeadlineExceeded,
// where the call that returns the error receives a context whose deadline had
// passed when the context was made. HonoursDeadline passes an expired context
// to its subject and states the match.
func honoursDeadline(p *pass, c check) bool {
	return p.contextMatch(c, "DeadlineExceeded", "honours-deadline", "HonoursDeadline")
}

// contextMatch reports, under the rule, a match of an error with the error
// sentinel of the package context, where the call that returns the error, in
// the check or in an earlier statement of its list, receives a context that
// ended with the sentinel before the call. The assertion states the check.
//
// A function literal of an earlier statement that assigns the error, as the
// body that an allocation test measures does, returns the error where each
// of its assignments of the error is a call that receives a context that
// ended before the literal. The literal reads the context's variable when it
// runs, so no statement from the literal up to the check may assign the
// variable.
func (p *pass) contextMatch(c check, sentinel, rule, assertion string) bool {
	err, target, ok := p.match(c)
	stmt, listed := c.statement()
	if !ok || !listed || !p.refers(target, "context", sentinel) {
		return false
	}
	n, at, ok := p.localSource(stmt, err)
	if !ok {
		n, ok = p.literalEnded(stmt, err, sentinel)
	} else {
		ok = p.receivesEnded(at, n, sentinel)
	}
	if !ok {
		return false
	}
	p.report(c.node, rule, assertion+" of "+p.brief(n), nil)
	return true
}

// receivesEnded reports whether the call n, in the statement at, receives a
// context that ended with the sentinel before the statement.
func (p *pass) receivesEnded(at inspector.Cursor, n *ast.CallExpr, sentinel string) bool {
	return slices.ContainsFunc(n.Args, func(arg ast.Expr) bool { return p.ended(at, arg, sentinel, sourceDepth) })
}

// literalEnded returns the first call of a function literal of an earlier
// statement of the list of stmt that gives the variable that e names its
// value, where each of the literal's assignments of the variable is a call
// that receives a context that ended with the sentinel before the literal's
// statement, and no statement from that statement up to stmt assigns the
// context's variable. It returns false where no such literal gives the value.
func (p *pass) literalEnded(stmt inspector.Cursor, e ast.Expr, sentinel string) (*ast.CallExpr, bool) {
	v := p.variable(e)
	if v == nil {
		return nil, false
	}
	s, value, _ := p.producer(stmt, v)
	lit, isLiteral := value.(*ast.FuncLit)
	if !isLiteral {
		return nil, false
	}
	kept := func(arg ast.Expr) bool {
		ctx := p.variable(arg)
		return (ctx == nil || !p.changes(s.Node(), ctx)) && !p.changed(s, stmt, arg)
	}
	var first *ast.CallExpr
	for _, assigned := range p.assigned(lit.Body, v) {
		n, isCall := ast.Unparen(assigned).(*ast.CallExpr)
		if !isCall || !slices.ContainsFunc(n.Args, func(arg ast.Expr) bool {
			return p.ended(s, arg, sentinel, sourceDepth) && kept(arg)
		}) {
			return nil, false
		}
		if first == nil {
			first = n
		}
	}
	return first, true
}

// ended reports whether e, an argument of a call in the statement at stmt, is
// a context that ended with the sentinel before the statement. A context ends
// with Canceled where an earlier statement of the list calls its cancel
// function, and with DeadlineExceeded where its deadline had passed when it
// was made. A function of the package that returns such a context, as a
// helper cancelled(t) does, ends its result too. depth bounds the helpers
// that ended follows.
func (p *pass) ended(stmt inspector.Cursor, e ast.Expr, sentinel string, depth int) bool {
	if depth == 0 || !isContext(p.TypesInfo.TypeOf(e)) {
		return false
	}
	if n, ok := ast.Unparen(e).(*ast.CallExpr); ok {
		return p.returnsEnded(stmt, n, sentinel, depth-1)
	}
	v := p.variable(e)
	if v == nil {
		return false
	}
	s, value, ok := p.producer(stmt, v)
	n, isCall := ast.Unparen(value).(*ast.CallExpr)
	if !ok || !isCall {
		return false
	}
	if !slices.Contains(derivations, p.callee(n)) {
		return p.returnsEnded(stmt, n, sentinel, depth-1)
	}
	if sentinel == "DeadlineExceeded" {
		return p.expired(n)
	}
	cancel := p.variable(s.Node().(*ast.AssignStmt).Lhs[1])
	for between, ok := s.NextSibling(); ok && between != stmt; between, ok = between.NextSibling() {
		if cancel != nil && p.calls(between.Node(), cancel) {
			return true
		}
	}
	return false
}

// returnsEnded reports whether the call n calls a function of the package
// whose last statement returns a context that ended with the sentinel. stmt
// is any statement of the package.
func (p *pass) returnsEnded(stmt inspector.Cursor, n *ast.CallExpr, sentinel string, depth int) bool {
	fn, ok := typeutil.Callee(p.TypesInfo, n).(*types.Func)
	for _, f := range p.Files {
		for _, d := range f.Decls {
			decl, isFunc := d.(*ast.FuncDecl)
			if !ok || !isFunc || decl.Body == nil || len(decl.Body.List) == 0 || p.TypesInfo.Defs[decl.Name] != fn {
				continue
			}
			ret, isReturn := decl.Body.List[len(decl.Body.List)-1].(*ast.ReturnStmt)
			if !isReturn || len(ret.Results) != 1 {
				return false
			}
			at, _ := stmt.Inspector().Root().FindNode(ret)
			return p.ended(at, ret.Results[0], sentinel, depth)
		}
	}
	return false
}

// expired reports whether the call n of the package context makes a context
// whose deadline has passed: a timeout that is a constant of at most zero, or
// a deadline of time.Now() plus such a constant, or of a time.Unix of seconds
// at or before the epoch.
func (p *pass) expired(n *ast.CallExpr) bool {
	switch p.callee(n) {
	case "context.WithTimeout", "context.WithTimeoutCause":
		return p.nonPositive(n.Args[1])
	case "context.WithDeadline", "context.WithDeadlineCause":
		deadline, ok := ast.Unparen(n.Args[1]).(*ast.CallExpr)
		if !ok {
			return false
		}
		switch p.callee(deadline) {
		case "time.Unix":
			return p.nonPositive(deadline.Args[0])
		case "(time.Time).Add":
			now, isCall := ast.Unparen(deadline.Fun.(*ast.SelectorExpr).X).(*ast.CallExpr)
			return isCall && p.callee(now) == "time.Now" && p.nonPositive(deadline.Args[0])
		}
	}
	return false
}

// nonPositive reports whether e is a constant of at most zero.
func (p *pass) nonPositive(e ast.Expr) bool {
	value := p.TypesInfo.Types[e].Value
	return value != nil && constant.Sign(value) <= 0
}

// calls reports whether the statement s is a call of the function that the
// variable v holds.
func (p *pass) calls(s ast.Node, v *types.Var) bool {
	stmt, ok := s.(*ast.ExprStmt)
	if !ok {
		return false
	}
	n, ok := stmt.X.(*ast.CallExpr)
	return ok && p.variable(n.Fun) == v
}

// completesWithin reports a select whose case of time.After fails the test.
// CompletesWithin runs its subject and fails when the time passes first.
func completesWithin(p *pass, cursor inspector.Cursor) {
	for _, s := range cursor.Node().(*ast.SelectStmt).Body.List {
		clause := s.(*ast.CommClause)
		if _, fails := p.failure(clause.Body); fails && clause.Comm != nil && p.timesOut(clause.Comm) {
			p.report(cursor.Node(), "completes-within", "CompletesWithin", nil)
			return
		}
	}
}

// timesOut reports whether the communication of a case receives from
// time.After.
func (p *pass) timesOut(comm ast.Stmt) bool {
	found := false
	ast.Inspect(comm, func(n ast.Node) bool {
		receive, ok := n.(*ast.UnaryExpr)
		if ok && receive.Op == token.ARROW {
			_, found = p.callOf(receive.X, "time", "After")
		}
		return !found
	})
	return found
}

// nilContextSafe reports NotPanics of a function literal that passes nil for
// a parameter whose declared type is context.Context. NilContextSafe passes
// nil to its subject and states that the subject does not panic.
func nilContextSafe(p *pass, c check) bool {
	if !c.is("NotPanics") {
		return false
	}
	lit, ok := ast.Unparen(c.call.Args[1]).(*ast.FuncLit)
	if !ok || !p.passesNilContext(lit.Body) {
		return false
	}
	p.report(c.node, "nil-context-safe", "NilContextSafe", nil)
	return true
}

// passesNilContext reports whether a call in n passes nil for a parameter
// whose declared type is context.Context. A parameter of a type parameter is
// none, whatever its instance.
func (p *pass) passesNilContext(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok {
			params := p.declared(call)
			for i, arg := range call.Args {
				found = found || i < params.Len() && p.TypesInfo.Types[arg].IsNil() && isContext(params.At(i).Type())
			}
		}
		return !found
	})
	return found
}

// declared returns the parameters that the function of the call n declares:
// those of a generic function before its instantiation, and none for a
// conversion.
func (p *pass) declared(n *ast.CallExpr) *types.Tuple {
	if fn, ok := typeutil.Callee(p.TypesInfo, n).(*types.Func); ok {
		return fn.Origin().Signature().Params()
	}
	if signature, ok := p.TypesInfo.TypeOf(n.Fun).Underlying().(*types.Signature); ok {
		return signature.Params()
	}
	return nil
}

// isContext reports whether t is context.Context.
func isContext(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "context" && named.Obj().Name() == "Context"
}
