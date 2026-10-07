// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// The surfaces, the packages whose functions are the assertions.
const (
	// assertPath is the import path of the surface that stops a test at its
	// first failure.
	assertPath = "go.dokimi.dev/assert"
	// expectPath is the import path of the surface that records a failure
	// and continues.
	expectPath = "go.dokimi.dev/assert/expect"
)

// failures are the methods of a test that fail it.
var failures = []string{"Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow"}

// negations maps each comparison to the comparison that is true where it is
// false.
var negations = map[token.Token]token.Token{
	token.EQL: token.NEQ, token.NEQ: token.EQL,
	token.LSS: token.GEQ, token.GEQ: token.LSS,
	token.GTR: token.LEQ, token.LEQ: token.GTR,
}

// mirrors maps each comparison to the comparison that is equivalent with its
// operands swapped.
var mirrors = map[token.Token]token.Token{
	token.EQL: token.EQL, token.NEQ: token.NEQ,
	token.LSS: token.GTR, token.GTR: token.LSS,
	token.LEQ: token.GEQ, token.GEQ: token.LEQ,
}

// equalFuncs are the functions whose call reports whether two values are
// equal, as an equality reads them.
var equalFuncs = [][2]string{{"reflect", "DeepEqual"}, {"bytes", "Equal"}, {"slices", "Equal"}, {"maps", "Equal"}}

// check is a check that a test states: a call of an assertion, or an if
// check, an if statement without else whose body is one failure of a test.
type check struct {
	// node is the call, or the if statement.
	node ast.Node
	// cursor is the cursor of node.
	cursor inspector.Cursor
	// call is the call of the assertion, and nil for an if check.
	call *ast.CallExpr
	// fn is the assertion, a package-level function of assert or expect,
	// and nil for an if check.
	fn *types.Func
	// name is the identifier that names fn after the name of its package, as
	// in assert.Equal. It is nil for an if check, and for a call that names
	// fn through a dot import or with type arguments. A fix renames the
	// assertion through it.
	name *ast.Ident
	// qualifier is the name of fn's package as the call writes it, and empty
	// where name is nil.
	qualifier string
	// stmt reports whether the call is a statement of its own.
	stmt bool
	// tb is the test that the check fails.
	tb ast.Expr
	// cond is the condition of a call of True or False or of an if check,
	// without its parentheses and negations, and nil for any other check.
	cond ast.Expr
	// holds reports whether the check states that cond is true: true for True
	// and false for False and an if check, inverted by each negation that
	// cond lost.
	holds bool
}

// equality is a check that two values are equal, or that they differ.
type equality struct {
	x, y  ast.Expr
	equal bool
}

// check returns the node at cursor as a check, and false for a node that is
// none.
func (p *pass) check(cursor inspector.Cursor) (check, bool) {
	switch n := cursor.Node().(type) {
	case *ast.CallExpr:
		c, ok := p.assertion(n)
		c.cursor = cursor
		_, c.stmt = cursor.Parent().Node().(*ast.ExprStmt)
		return c, ok
	default:
		return p.ifCheck(cursor, n.(*ast.IfStmt))
	}
}

// assertion returns n as a call of an assertion, a package-level function of
// assert or expect whose first parameter is a test, and false for a call of
// anything else.
func (p *pass) assertion(n *ast.CallExpr) (check, bool) {
	fn := p.function(n)
	if fn == nil || fn.Pkg().Path() != assertPath && fn.Pkg().Path() != expectPath || !takesTest(fn) {
		return check{}, false
	}
	c := check{node: n, call: n, fn: fn, tb: n.Args[0]}
	if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
		c.name, c.qualifier = sel.Sel, sel.X.(*ast.Ident).Name
	}
	if c.is("True", "False") {
		c.cond, c.holds = normalize(n.Args[1], fn.Name() == "True")
	}
	return c, true
}

// assertionStatement returns the call of an assertion that the statement s
// consists of, and false for any other statement.
func (p *pass) assertionStatement(s ast.Node) (check, bool) {
	stmt, ok := s.(*ast.ExprStmt)
	if !ok {
		return check{}, false
	}
	n, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return check{}, false
	}
	c, ok := p.assertion(n)
	c.stmt = true
	return c, ok
}

// inert reports whether the statement s is a check or a declaration: a call
// of an assertion, whatever its arguments call, or a declaration whose
// variables take no values. The rules that relate two calls count no such
// statement as a step between them.
func (p *pass) inert(s ast.Node) bool {
	if decl, ok := s.(*ast.DeclStmt); ok {
		return !slices.ContainsFunc(decl.Decl.(*ast.GenDecl).Specs, func(spec ast.Spec) bool {
			value, isValue := spec.(*ast.ValueSpec)
			return isValue && len(value.Values) > 0
		})
	}
	_, ok := p.assertionStatement(s)
	return ok
}

// takesTest reports whether the first parameter of fn is the test of the
// assertions, assert.TB.
func takesTest(fn *types.Func) bool {
	params := fn.Signature().Params()
	if params.Len() == 0 {
		return false
	}
	named, ok := types.Unalias(params.At(0).Type()).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == assertPath && named.Obj().Name() == "TB"
}

// ifCheck returns the if statement n as a check, and false for an if
// statement that is none: one with an else branch, or whose body is no
// single failure of a test. A test is a value whose method set has Helper
// and the failure.
func (p *pass) ifCheck(cursor inspector.Cursor, n *ast.IfStmt) (check, bool) {
	tb, ok := p.failure(n.Body.List)
	if !ok || n.Else != nil {
		return check{}, false
	}
	c := check{node: n, cursor: cursor, tb: tb}
	c.cond, c.holds = normalize(n.Cond, false)
	return c, true
}

// failure returns the test of the statements list when they are one call
// of a method of a test that fails it, and false otherwise.
func (p *pass) failure(list []ast.Stmt) (ast.Expr, bool) {
	if len(list) != 1 {
		return nil, false
	}
	stmt, ok := list[0].(*ast.ExprStmt)
	if !ok {
		return nil, false
	}
	n, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := n.Fun.(*ast.SelectorExpr)
	if !ok || !slices.Contains(failures, sel.Sel.Name) {
		return nil, false
	}
	methods := types.NewMethodSet(p.TypesInfo.TypeOf(sel.X))
	return sel.X, methods.Lookup(nil, "Helper") != nil && methods.Lookup(nil, sel.Sel.Name) != nil
}

// normalize returns cond without its parentheses and negations, and whether
// a check of cond states that it is true, given that the check of the
// original states so when holds is true.
func normalize(cond ast.Expr, holds bool) (ast.Expr, bool) {
	for {
		cond = ast.Unparen(cond)
		not, ok := cond.(*ast.UnaryExpr)
		if !ok || not.Op != token.NOT {
			return cond, holds
		}
		cond, holds = not.X, !holds
	}
}

// is reports whether c calls one of the assertions names.
func (c check) is(names ...string) bool {
	return c.fn != nil && slices.Contains(names, c.fn.Name())
}

// surface reports whether c calls an assertion of the surface path.
func (c check) surface(path string) bool {
	return c.fn != nil && c.fn.Pkg().Path() == path
}

// comparison returns the comparison x op y that the condition of c states,
// with op negated where c states that the condition is false. ok is
// false where the condition is no comparison.
func (c check) comparison() (x, y ast.Expr, op token.Token, ok bool) {
	b, isBinary := c.cond.(*ast.BinaryExpr)
	if !isBinary {
		return nil, nil, 0, false
	}
	op, ok = negations[b.Op]
	if c.holds {
		op = b.Op
	}
	return b.X, b.Y, op, ok
}

// values returns the values that c checks: the condition of a condition,
// and the arguments of an assertion between its test and its message.
func (c check) values() []ast.Expr {
	if c.call == nil || c.cond != nil {
		return []ast.Expr{c.cond}
	}
	params := c.fn.Signature().Params()
	end := len(c.call.Args)
	for i := range params.Len() {
		if params.At(i).Name() == "msg" {
			end = i
		}
	}
	return c.call.Args[1:end]
}

// equality returns the equality that c states: Equal or NotEqual without
// options, or a condition of ==, !=, or a function of equalFuncs.
func (p *pass) equality(c check) (equality, bool) {
	if c.is("Equal", "NotEqual") && len(c.call.Args) == 4 {
		return equality{c.call.Args[1], c.call.Args[2], c.fn.Name() == "Equal"}, true
	}
	if x, y, op, ok := c.comparison(); ok && (op == token.EQL || op == token.NEQ) {
		return equality{x, y, op == token.EQL}, true
	}
	for _, f := range equalFuncs {
		if n, ok := p.callOf(c.cond, f[0], f[1]); ok {
			return equality{n.Args[0], n.Args[1], c.holds}, true
		}
	}
	return equality{}, false
}

// statement returns the cursor of the statement that c is in a list of
// statements, and false for a check that is no statement of such a list.
func (c check) statement() (inspector.Cursor, bool) {
	cursor := c.cursor
	if c.call != nil {
		cursor = cursor.Parent()
	}
	return cursor, (c.call == nil || c.stmt) && listed(cursor)
}

// listed reports whether the statement at cursor is an element of a list of
// statements, which run in order.
func listed(cursor inspector.Cursor) bool {
	switch cursor.ParentEdgeKind() {
	case edge.BlockStmt_List, edge.CaseClause_Body, edge.CommClause_Body:
		return true
	}
	return false
}

// before returns the statements before the statement at cursor in its list,
// the nearest first.
func before(cursor inspector.Cursor) []inspector.Cursor {
	var earlier []inspector.Cursor
	kind := cursor.ParentEdgeKind()
	for prev, ok := cursor.PrevSibling(); ok && prev.ParentEdgeKind() == kind; prev, ok = prev.PrevSibling() {
		earlier = append(earlier, prev)
	}
	return earlier
}

// function returns the package-level function that the call n calls, and nil
// for a call of anything else: a method, a function value, a builtin or a
// conversion.
func (p *pass) function(n *ast.CallExpr) *types.Func {
	fn := typeutil.StaticCallee(p.TypesInfo, n)
	if fn == nil || fn.Signature().Recv() != nil {
		return nil
	}
	return fn
}

// callee returns the full name of the function or method that the call n
// calls, as testing.AllocsPerRun or (io/fs.FileMode).Perm, and empty for a
// call of anything else: a function value, a builtin or a conversion.
func (p *pass) callee(n *ast.CallExpr) string {
	fn, ok := typeutil.Callee(p.TypesInfo, n).(*types.Func)
	if !ok {
		return ""
	}
	return fn.FullName()
}

// callOf returns e as a call of the package-level function name of the
// package path, such as errors.Is, and false for any other expression.
func (p *pass) callOf(e ast.Expr, path, name string) (*ast.CallExpr, bool) {
	n, ok := ast.Unparen(e).(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	fn := p.function(n)
	return n, fn != nil && fn.Pkg().Path() == path && fn.Name() == name
}

// operands returns the operands of a chain of the operator op, such as the
// three of a && b && c, in order and without their parentheses.
func operands(e ast.Expr, op token.Token) []ast.Expr {
	e = ast.Unparen(e)
	if b, ok := e.(*ast.BinaryExpr); ok && b.Op == op {
		return append(operands(b.X, op), operands(b.Y, op)...)
	}
	return []ast.Expr{e}
}

// variable returns the variable that e names, and nil where e names none.
func (p *pass) variable(e ast.Expr) *types.Var {
	id, ok := ast.Unparen(e).(*ast.Ident)
	if !ok {
		return nil
	}
	v, _ := p.TypesInfo.ObjectOf(id).(*types.Var)
	return v
}

// constant reports whether e is a constant.
func (p *pass) constant(e ast.Expr) bool {
	return p.TypesInfo.Types[e].Value != nil
}

// pure reports whether evaluating e calls no function and receives from no
// channel, so that evaluating it twice has the effect of evaluating it once.
func pure(e ast.Expr) bool {
	ok := true
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			ok = false
		case *ast.UnaryExpr:
			ok = ok && n.Op != token.ARROW
		}
		return ok
	})
	return ok
}
