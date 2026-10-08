// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// link is a statement that a chain states: a call of an assertion of a
// surface whose chain has a method of the assertion's name, with a test and a
// variable.
type link struct {
	surface *types.Package
	tb, v   *types.Var
}

// chain reports two or more consecutive statements that each call an
// assertion of one surface with one test and one variable, where the
// surface's chain has a method of the assertion's name. A chain on the
// variable states them in one statement.
//
// It suggests the chain, That(t, v).A(...).B(...), where each call names its
// assertion through its package, no other fix rewrites one of the calls, and
// no comment is between them. The chain evaluates v once, and a statement of
// it reads the variable as each call did.
func chain(p *pass, cursor inspector.Cursor) {
	first, ok := p.link(cursor.Node())
	if !ok {
		return
	}
	if prev, ok := cursor.PrevSibling(); ok {
		if l, ok := p.link(prev.Node()); ok && l == first {
			return
		}
	}
	run := []inspector.Cursor{cursor}
	for next, ok := cursor.NextSibling(); ok; next, ok = next.NextSibling() {
		if l, ok := p.link(next.Node()); !ok || l != first {
			break
		}
		run = append(run, next)
	}
	if len(run) < 2 {
		return
	}
	message := "state the assertions about " + first.v.Name() + " with " + first.surface.Name() + ".That"
	p.reportf(cursor.Node(), "chain", message, p.chainFix(run))
}

// chainFix returns the fix that writes the run of statements as one chain,
// and no fix where a call of the run names its assertion otherwise than
// through its package, where another fix rewrites a call, or where a comment
// is between the statements.
func (p *pass) chainFix(run []inspector.Cursor) []analysis.SuggestedFix {
	start, end := run[0].Node().Pos(), run[len(run)-1].Node().End()
	for _, f := range p.Files {
		for _, group := range f.Comments {
			if start < group.Pos() && group.End() < end {
				return nil
			}
		}
	}
	var text strings.Builder
	for i, s := range run {
		c, _ := p.assertionStatement(s.Node())
		if c.name == nil || p.fixed[c.call] {
			return nil
		}
		if i == 0 {
			text.WriteString(c.qualifier + ".That(" + p.source(c.tb) + ", " + p.source(c.call.Args[1]) + ")")
		}
		args := make([]string, len(c.call.Args)-2)
		for j, arg := range c.call.Args[2:] {
			args[j] = p.source(arg)
		}
		text.WriteString(".\n" + c.fn.Name() + "(" + strings.Join(args, ", ") + ")")
	}
	return []analysis.SuggestedFix{{
		Message:   "Call the assertions through That",
		TextEdits: []analysis.TextEdit{{Pos: start, End: end, NewText: []byte(text.String())}},
	}}
}

// link returns the statement s as a link, and false for any other statement.
func (p *pass) link(s ast.Node) (link, bool) {
	c, ok := p.assertionStatement(s)
	if !ok || !chainable(c.fn) {
		return link{}, false
	}
	l := link{surface: c.fn.Pkg(), tb: p.variable(c.tb), v: p.variable(c.call.Args[1])}
	return l, l.tb != nil && l.v != nil
}

// chainable reports whether the chain of the surface of the assertion fn has
// a method of fn's name.
func chainable(fn *types.Func) bool {
	t, ok := fn.Pkg().Scope().Lookup("Assertion").(*types.TypeName)
	return ok && types.NewMethodSet(types.NewPointer(t.Type())).Lookup(fn.Pkg(), fn.Name()) != nil
}
