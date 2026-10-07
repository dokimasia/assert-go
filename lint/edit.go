// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// source returns the text of the node n as gofmt writes it, without the
// comments inside it.
func (p *pass) source(n ast.Node) string {
	var b strings.Builder
	// The printer fails only for a writer that fails or for a node that is no
	// Go syntax, and a parsed node written to a strings.Builder is neither.
	_ = format.Node(&b, p.Fset, n)
	return b.String()
}

// rewrite returns the fix that calls the assertion name of the call's
// surface, with args in place of the arguments between the call's test and
// its message, and with a call of each option of opts after the message. It
// returns no fix for an if check, and for a call that names its assertion
// through a dot import or with type arguments.
func (c check) rewrite(name, args string, opts ...string) []analysis.SuggestedFix {
	if c.name == nil {
		return nil
	}
	msg := c.call.Args[len(c.call.Args)-1]
	edits := []analysis.TextEdit{
		{Pos: c.name.Pos(), End: c.name.End(), NewText: []byte(name)},
		{Pos: c.call.Args[1].Pos(), End: c.call.Args[len(c.call.Args)-2].End(), NewText: []byte(args)},
	}
	for _, opt := range opts {
		text := ", " + c.qualifier + "." + opt + "()"
		edits = append(edits, analysis.TextEdit{Pos: msg.End(), End: msg.End(), NewText: []byte(text)})
	}
	return []analysis.SuggestedFix{{Message: "Call " + name, TextEdits: edits}}
}

// replace returns the fix that writes text in place of the node n.
func replace(message string, n ast.Node, text string) []analysis.SuggestedFix {
	return []analysis.SuggestedFix{{
		Message:   message,
		TextEdits: []analysis.TextEdit{{Pos: n.Pos(), End: n.End(), NewText: []byte(text)}},
	}}
}

// spell returns the text of the type t in the file at pos, and false for a
// type that the file cannot write. The file writes a basic type, a defined
// type of its own package, an exported defined type of a package that it
// imports, and a pointer to either. A defined type with type arguments is
// none of them.
func (p *pass) spell(pos token.Pos, t types.Type) (string, bool) {
	switch t := types.Unalias(t).(type) {
	case *types.Basic:
		return t.Name(), t.Info()&types.IsUntyped == 0 && t.Kind() != types.UnsafePointer
	case *types.Pointer:
		text, ok := p.spell(pos, t.Elem())
		return "*" + text, ok
	case *types.Named:
		obj := t.Obj()
		if obj.Pkg() == nil || obj.Pkg() == p.Pkg {
			return obj.Name(), t.TypeArgs().Len() == 0
		}
		qualifier, ok := p.importName(pos, obj.Pkg())
		return qualifier + obj.Name(), ok && obj.Exported() && t.TypeArgs().Len() == 0
	}
	return "", false
}

// importName returns the qualifier through which the file at pos names the
// package pkg, such as "fs." or "" for a dot import, and false where the file
// does not import pkg.
func (p *pass) importName(pos token.Pos, pkg *types.Package) (string, bool) {
	for _, f := range p.Files {
		if f.FileStart > pos || pos > f.FileEnd {
			continue
		}
		for _, spec := range f.Imports {
			if name := p.TypesInfo.PkgNameOf(spec); name != nil && name.Imported() == pkg {
				if name.Name() == "." {
					return "", true
				}
				return name.Name() + ".", true
			}
		}
	}
	return "", false
}
