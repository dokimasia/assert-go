// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// skipPrefix opens an annotation.
const skipPrefix = "//dokimi:lint-skip"

// skip is an annotation, a line comment that leaves out the reports of the
// rules that it lists on one line: //dokimi:lint-skip, a comma-separated
// list of rules, a colon and a reason. On a line of its own it covers the
// next line, and after code it covers its own line.
type skip struct {
	comment *ast.Comment
	// file and line are the file and the line that the annotation covers.
	file string
	line int
	// rules are the rules that the annotation lists, in its order.
	rules []string
	// left are the rules of which the annotation left out a report.
	left map[string]bool
}

// annotations returns the annotations of the files of the package. It
// reports each comment that opens with skipPrefix and states no rule or no
// reason, which leaves out nothing.
func (p *pass) annotations() []*skip {
	var skips []*skip
	for _, f := range p.Files {
		for _, group := range f.Comments {
			for _, c := range group.List {
				rest, ok := strings.CutPrefix(c.Text, skipPrefix)
				if !ok || rest != "" && rest[0] != ' ' {
					continue
				}
				list, reason, stated := strings.Cut(rest, ":")
				rules := strings.Split(list, ",")
				for i, rule := range rules {
					rules[i] = strings.TrimSpace(rule)
				}
				if !stated || strings.TrimSpace(reason) == "" || slices.Contains(rules, "") {
					p.reportSkip(c, "state the rules and a reason after them, as in "+
						skipPrefix+" for-all: the seed is fixed to replay a workload")
					continue
				}
				position := p.Fset.Position(c.Pos())
				line := position.Line + 1
				if p.afterCode(f, c) {
					line = position.Line
				}
				skips = append(skips, &skip{c, position.Filename, line, rules, make(map[string]bool)})
			}
		}
	}
	return skips
}

// afterCode reports whether code of the file f precedes the comment c on its
// line: a node other than a comment ends there before c.
func (p *pass) afterCode(f *ast.File, c *ast.Comment) bool {
	line := p.Fset.Position(c.Pos()).Line
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		switch n.(type) {
		case nil, *ast.Comment, *ast.CommentGroup:
			return false
		}
		found = found || n.End() <= c.Pos() && p.Fset.Position(n.End()).Line == line
		return !found
	})
	return found
}

// skipped reports whether an annotation leaves out the reports of the rule
// that start at pos, and records that the annotation left one out.
func (p *pass) skipped(pos token.Pos, rule string) bool {
	position := p.Fset.Position(pos)
	for _, s := range p.skips {
		if s.file == position.Filename && s.line == position.Line && slices.Contains(s.rules, rule) {
			s.left[rule] = true
			return true
		}
	}
	return false
}

// reportUnused reports each rule of an annotation that left out no report,
// so that an annotation goes when the check that it excuses goes.
func (p *pass) reportUnused() {
	for _, s := range p.skips {
		for _, rule := range s.rules {
			if !s.left[rule] {
				p.reportSkip(s.comment, "remove "+rule+", because no report of the rule starts on the line that the "+
					"annotation covers")
			}
		}
	}
}

// reportSkip reports the annotation c with the message, which no annotation
// leaves out.
func (p *pass) reportSkip(c *ast.Comment, message string) {
	p.Report(analysis.Diagnostic{Pos: c.Pos(), End: c.End(), Message: "lint-skip: " + message})
}
