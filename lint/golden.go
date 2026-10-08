// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/constant"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/inspector"
)

// The directories of a package's test files, each with a trailing slash.
const (
	// testdataDir is the directory of the files that a test reads, such as
	// its golden files.
	testdataDir = "testdata/"
	// goldenDir is the directory of the golden files that golden.Match
	// resolves a name against.
	goldenDir = "testdata/golden/"
)

// joins are the functions that join the elements of a path.
var joins = []string{"path/filepath.Join", "path.Join"}

// flags maps each function of the package flag that defines a boolean flag
// to the index of the flag's name among its arguments.
var flags = map[string]int{"flag.Bool": 0, "flag.BoolVar": 1}

// fileContent reports an equality of the content of a file that os.ReadFile
// reads, in the check or in an earlier statement of its list. A file under
// testdata is a golden file, which golden.Match states under testdata/golden
// and golden.MatchAt anywhere else. files.HasContent states the content of
// any other file, such as one that the code under test writes. A value other
// than text, such as the error of os.ReadFile, is no content.
func fileContent(p *pass, c check) bool {
	e, ok := p.equality(c)
	stmt, listed := c.statement()
	if !ok || !listed {
		return false
	}
	var reads []string
	rule, name := "has-content", "files.HasContent"
	for _, value := range []ast.Expr{e.x, e.y} {
		n, _, read := p.localSource(stmt, value)
		if !read || p.callee(n) != "os.ReadFile" || !p.content(value) {
			continue
		}
		reads = append(reads, p.brief(n))
		switch {
		case p.under(n.Args[0], goldenDir):
			rule, name = "golden-match", "golden.Match"
		case p.under(n.Args[0], testdataDir):
			rule, name = "golden-match", "golden.MatchAt"
		}
	}
	if len(reads) == 0 {
		return false
	}
	p.report(c.node, rule, name+" of "+strings.Join(reads, " and "), nil)
	return true
}

// content reports whether e is text: a string or a slice of bytes.
func (p *pass) content(e ast.Expr) bool {
	t := p.TypesInfo.TypeOf(e)
	return isString(t) || isBytes(t)
}

// under reports whether the path e is in the directory dir, a path with a
// trailing slash: a constant with that prefix, or a join whose leading
// constant elements form one.
func (p *pass) under(e ast.Expr, dir string) bool {
	var joined strings.Builder
	if n, ok := ast.Unparen(e).(*ast.CallExpr); ok && slices.Contains(joins, p.callee(n)) {
		for _, arg := range n.Args {
			value := p.TypesInfo.Types[arg].Value
			if value == nil {
				break
			}
			joined.WriteString(constant.StringVal(value))
			joined.WriteByte('/')
		}
	} else if value := p.TypesInfo.Types[e].Value; value != nil {
		joined.WriteString(constant.StringVal(value))
	}
	return strings.HasPrefix(joined.String(), dir)
}

// updateFlag reports the definition of a flag -update in a test file. A
// call of golden.MatchAt with golden.ShouldUpdate states the golden file, its
// update under -update, and the diff of a mismatch. The rule reports no
// definition in another file, where a program can define a flag of that name
// for its own use.
func updateFlag(p *pass, cursor inspector.Cursor) {
	n := cursor.Node().(*ast.CallExpr)
	fn := p.function(n)
	if fn == nil || !strings.HasSuffix(p.Fset.File(n.Pos()).Name(), "_test.go") {
		return
	}
	i, ok := flags[fn.FullName()]
	if !ok {
		return
	}
	if value := p.TypesInfo.Types[n.Args[i]].Value; value != nil && constant.StringVal(value) == "update" {
		p.report(n, "update-flag", "golden.MatchAt and golden.ShouldUpdate", nil)
	}
}
