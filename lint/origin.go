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
)

// sourceDepth is the most conversions and variables that sources follows
// from an expression to the call that produces its value.
const sourceDepth = 4

// origins returns the origins of the variables of a package: for each
// variable, the calls that assign it or receive its address, as
// os.ReadFile in want, err := os.ReadFile(path) and json.Unmarshal in
// json.Unmarshal(data, &got). It also returns, for each variable that is the
// ok of a lookup of a map, as ok in _, ok := m[k], that lookup.
func origins(p *pass, in *inspector.Inspector) (map[types.Object][]*ast.CallExpr, map[types.Object]*ast.IndexExpr) {
	calls := make(map[types.Object][]*ast.CallExpr)
	lookups := make(map[types.Object]*ast.IndexExpr)
	assign := func(lhs, rhs []ast.Expr) {
		for i, l := range lhs {
			v := p.variable(l)
			if v == nil || len(rhs) == 0 {
				continue
			}
			switch r := ast.Unparen(assignedValue(rhs, len(lhs), i)).(type) {
			case *ast.CallExpr:
				calls[v] = append(calls[v], r)
			case *ast.IndexExpr:
				if _, isMap := p.TypesInfo.TypeOf(r.X).Underlying().(*types.Map); isMap && len(rhs) < len(lhs) {
					lookups[v] = r
				}
			}
		}
	}
	for cursor := range in.Root().Preorder((*ast.AssignStmt)(nil), (*ast.ValueSpec)(nil), (*ast.UnaryExpr)(nil)) {
		switch n := cursor.Node().(type) {
		case *ast.AssignStmt:
			assign(n.Lhs, n.Rhs)
		case *ast.ValueSpec:
			names := make([]ast.Expr, len(n.Names))
			for i, name := range n.Names {
				names[i] = name
			}
			assign(names, n.Values)
		case *ast.UnaryExpr:
			if v := p.variable(n.X); v != nil && n.Op == token.AND && cursor.ParentEdgeKind() == edge.CallExpr_Args {
				calls[v] = append(calls[v], cursor.Parent().Node().(*ast.CallExpr))
			}
		}
	}
	return calls, lookups
}

// sources returns the calls that produce the value of e: e where it is a
// call, and the origins of a variable that e names. A conversion passes on
// to its operand. sources follows at most sourceDepth steps.
func (p *pass) sources(e ast.Expr) []*ast.CallExpr {
	return p.sourcesWithin(e, sourceDepth)
}

// sourcesWithin returns the sources of e within depth steps.
func (p *pass) sourcesWithin(e ast.Expr, depth int) []*ast.CallExpr {
	if depth == 0 {
		return nil
	}
	e = ast.Unparen(e)
	if n, ok := e.(*ast.CallExpr); ok {
		if p.TypesInfo.Types[n.Fun].IsType() {
			return p.sourcesWithin(n.Args[0], depth-1)
		}
		return []*ast.CallExpr{n}
	}
	var calls []*ast.CallExpr
	for _, n := range p.origins[p.variable(e)] {
		calls = append(calls, p.sourcesWithin(n, depth-1)...)
	}
	return calls
}

// localSource returns the call that produces the value of e for the
// statement at stmt, and the statement that makes the call: e where it is a
// call, and otherwise the call of the nearest earlier statement of stmt's
// list that gives e's variable its value. A conversion passes on to its
// operand. It returns false where no statement of the list produces the
// value, as for a value that a loop assigns, and where a function literal of
// the list assigns the value, which the literal gives when it runs. Each step
// peels a conversion or moves to an earlier statement, so the search ends.
func (p *pass) localSource(stmt inspector.Cursor, e ast.Expr) (*ast.CallExpr, inspector.Cursor, bool) {
	for {
		e = p.operand(e)
		if n, ok := e.(*ast.CallExpr); ok {
			return n, stmt, true
		}
		v := p.variable(e)
		if v == nil {
			return nil, stmt, false
		}
		s, value, ok := p.producer(stmt, v)
		if !ok {
			return nil, stmt, false
		}
		stmt, e = s, value
	}
}

// operand returns e without its parentheses and conversions, as data of
// string(data).
func (p *pass) operand(e ast.Expr) ast.Expr {
	for {
		e = ast.Unparen(e)
		n, ok := e.(*ast.CallExpr)
		if !ok || !p.TypesInfo.Types[n.Fun].IsType() {
			return e
		}
		e = n.Args[0]
	}
}

// producer returns the nearest earlier statement of the list of stmt that
// gives the variable v its value, and the expression whose value it gives:
// the right-hand side of an assignment, the call that receives v's address,
// or a function literal whose body assigns v.
func (p *pass) producer(stmt inspector.Cursor, v *types.Var) (inspector.Cursor, ast.Expr, bool) {
	for _, s := range before(stmt) {
		if assign, ok := s.Node().(*ast.AssignStmt); ok {
			for i, lhs := range assign.Lhs {
				if p.variable(lhs) == v {
					return s, assignedValue(assign.Rhs, len(assign.Lhs), i), true
				}
			}
		}
		if n := p.addressed(s.Node(), v); n != nil {
			return s, n, true
		}
		if lit := p.literal(s.Node(), v); lit != nil {
			return s, lit, true
		}
	}
	return stmt, nil, false
}

// assignedValue returns the expression that gives the i-th of n assigned
// operands its value: the i-th of values, or the one call of a multi-value
// assignment.
func assignedValue(values []ast.Expr, n, i int) ast.Expr {
	if len(values) == n {
		return values[i]
	}
	return values[0]
}

// literal returns the function literal in the statement s whose body assigns
// the variable v, and nil where none does.
func (p *pass) literal(s ast.Node, v *types.Var) *ast.FuncLit {
	var found *ast.FuncLit
	ast.Inspect(s, func(n ast.Node) bool {
		if lit, ok := n.(*ast.FuncLit); ok && len(p.assigned(lit.Body, v)) > 0 {
			found = lit
		}
		return found == nil
	})
	return found
}

// assigned returns the expressions whose values the assignments in n give
// the variable v.
func (p *pass) assigned(n ast.Node, v *types.Var) []ast.Expr {
	var values []ast.Expr
	ast.Inspect(n, func(node ast.Node) bool {
		if assign, ok := node.(*ast.AssignStmt); ok {
			for i, lhs := range assign.Lhs {
				if p.variable(lhs) == v {
					values = append(values, assignedValue(assign.Rhs, len(assign.Lhs), i))
				}
			}
		}
		return true
	})
	return values
}

// addressed returns the call in the statement s that receives the address
// of the variable v, outside a function literal, and nil where none does.
func (p *pass) addressed(s ast.Node, v *types.Var) *ast.CallExpr {
	var found *ast.CallExpr
	ast.Inspect(s, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.CallExpr:
			for _, arg := range n.Args {
				amp, ok := ast.Unparen(arg).(*ast.UnaryExpr)
				if ok && amp.Op == token.AND && p.variable(amp.X) == v {
					found = n
				}
			}
		}
		return found == nil
	})
	return found
}

// sourceOf returns the source of e that calls one of the functions names, by
// full name, and nil where none does. Unlike produced, it does not read a
// call inside e, so a value that e computes from such a call, such as one bit
// of a mode or the base name of a link's target, has no such source.
func (p *pass) sourceOf(e ast.Expr, names ...string) *ast.CallExpr {
	for _, n := range p.sources(e) {
		if slices.Contains(names, p.callee(n)) {
			return n
		}
	}
	return nil
}

// produced returns the call of one of the functions names, by full name,
// that produces a value that e reads: a call inside e, or a source of a
// variable that e reads. It returns nil where no such call produces a value
// that e reads.
func (p *pass) produced(e ast.Expr, names ...string) *ast.CallExpr {
	var found *ast.CallExpr
	ast.Inspect(e, func(n ast.Node) bool {
		var candidates []*ast.CallExpr
		switch n := n.(type) {
		case *ast.CallExpr:
			candidates = []*ast.CallExpr{n}
		case *ast.Ident:
			candidates = p.sources(n)
		}
		for _, call := range candidates {
			if found == nil && slices.Contains(names, p.callee(call)) {
				found = call
			}
		}
		return found == nil
	})
	return found
}
