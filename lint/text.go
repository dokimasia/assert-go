// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/types"
)

// affixes are the functions that report whether a text starts or ends with
// another, each with the rule that reports a check of it and the assertion
// that states the check.
var affixes = []struct {
	path, name, rule string
}{
	{"strings", "HasPrefix", "has-prefix"},
	{"bytes", "HasPrefix", "has-prefix"},
	{"strings", "HasSuffix", "has-suffix"},
	{"bytes", "HasSuffix", "has-suffix"},
}

// matchFuncs are the functions that report whether a regular expression
// matches a text.
var matchFuncs = []string{"regexp.MatchString", "(*regexp.Regexp).MatchString", "(*regexp.Regexp).Match"}

// prefix reports True of strings.HasPrefix, bytes.HasPrefix and their suffix
// forms, and suggests HasPrefix or HasSuffix, which read a string or a
// []byte as text. The fix writes an affix of bytes as the string that it
// converts, or as its conversion to a string.
func prefix(p *pass, c check) bool {
	for _, f := range affixes {
		n, ok := p.callOf(c.cond, f.path, f.name)
		if !ok || !c.holds {
			continue
		}
		affix := p.source(n.Args[1])
		if f.path == "bytes" {
			affix = p.asString(n.Args[1])
		}
		p.report(c.node, f.rule, f.name, c.rewrite(f.name, p.source(n.Args[0])+", "+affix))
		return true
	}
	return false
}

// asString returns the text of an expression of bytes as a string: the
// operand of a conversion of a string to bytes, and the conversion of any
// other expression to a string.
func (p *pass) asString(e ast.Expr) string {
	n, ok := ast.Unparen(e).(*ast.CallExpr)
	if ok && p.TypesInfo.Types[n.Fun].IsType() && isString(p.TypesInfo.TypeOf(n.Args[0])) {
		return p.source(n.Args[0])
	}
	return "string(" + p.source(e) + ")"
}

// isString reports whether t is a string type.
func isString(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsString != 0
}

// isBytes reports whether t is a slice of bytes, whose element type is byte
// itself. A slice of a type defined over uint8, such as an enumeration, is a
// sequence, as the assertions read it.
func isBytes(t types.Type) bool {
	slice, ok := t.Underlying().(*types.Slice)
	return ok && types.Identical(slice.Elem(), types.Typ[types.Byte])
}

// matches reports True of a match of a regular expression, through
// regexp.MatchString or a method of a *regexp.Regexp, where the condition is
// the match or a variable that it assigns. Matches states it. It suggests no
// fix: Matches reads a pattern of the portable subset, in which $ matches at
// the end of the text alone and . matches no line terminator.
func matches(p *pass, c check) bool {
	n := p.sourceOf(c.cond, matchFuncs...)
	if n == nil || !c.holds {
		return false
	}
	p.report(c.node, "matches", "Matches of "+p.brief(n), nil)
	return true
}
