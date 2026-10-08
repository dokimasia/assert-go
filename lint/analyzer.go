// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/ast"
	"go/types"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer reports a check that a test writes by hand and that an assertion
// of go.dokimi.dev/assert states, and suggests the rewrite where the rewrite
// keeps the check's meaning in every case.
var Analyzer = &analysis.Analyzer{
	Name:     "assertlint",
	Doc:      "report hand-written checks that an assertion states",
	URL:      "https://pkg.go.dev/go.dokimi.dev/assert/lint",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// checkRules are the rules over a check, in the order in which they claim
// it. The first rule that reports a check ends the search, so no check has
// two diagnostics of these rules. conjunction names the assertion of each
// operand through the rules. It reads them from the pass, because a rule
// that referred to checkRules would make its initializer depend on itself.
var checkRules = []func(p *pass, c check) bool{
	panics, nilContextSafe, inRange, conjunction,
	honoursCancellation, honoursDeadline, pathAbsent, afterClose, errorsIs, errorsAs, sentinel,
	fileKind, hasMode, linksTo, fileContent,
	measurement,
	commutative, associative, roundTrip, repetition, permutation,
	rejects,
	equalFunc, deepEqual, nilCheck, equalNil, length, contains, membership, containsInOrder, prefix, matches,
	closeTo, pairwise,
	order, compare,
	condition,
}

// loopNodes are the types of a loop.
var loopNodes = []ast.Node{(*ast.ForStmt)(nil), (*ast.RangeStmt)(nil)}

// statementRules are the rules over a node that is no check, each with the
// types of the nodes that it reads.
var statementRules = []struct {
	nodes []ast.Node
	rule  func(p *pass, cursor inspector.Cursor)
}{
	{nodes: []ast.Node{(*ast.ExprStmt)(nil)}, rule: chain},
	{nodes: []ast.Node{(*ast.RangeStmt)(nil)}, rule: total},
	{nodes: loopNodes, rule: poisoned},
	{nodes: loopNodes, rule: noDuplicates},
	{nodes: loopNodes, rule: monotonic},
	{nodes: loopNodes, rule: eventually},
	{nodes: loopNodes, rule: forAll},
	{nodes: []ast.Node{(*ast.SelectStmt)(nil)}, rule: completesWithin},
	{nodes: []ast.Node{(*ast.CallExpr)(nil)}, rule: updateFlag},
	{nodes: []ast.Node{(*ast.CallExpr)(nil)}, rule: quick},
	{nodes: []ast.Node{(*ast.CallExpr)(nil)}, rule: propertyForm},
	{nodes: []ast.Node{(*ast.CallExpr)(nil)}, rule: machine},
}

// pass is one run of the analyzer over a package.
type pass struct {
	*analysis.Pass
	// rules are the rules over a check, checkRules, which named asks.
	rules []func(p *pass, c check) bool
	// origins maps each variable to the calls that assign it or receive its
	// address.
	origins map[types.Object][]*ast.CallExpr
	// lookups maps each variable to the lookup of a map whose ok it is, as
	// ok in _, ok := m[k].
	lookups map[types.Object]*ast.IndexExpr
	// fixed are the calls of assertions that a rule over checks fixes.
	fixed map[*ast.CallExpr]bool
	// skips are the annotations of the package.
	skips []*skip
	// naming, where it is not nil, receives what a rule states in place of a
	// report, while named asks the rules which assertion states a check.
	naming *string
}

// run reports the hand-written checks of the package of ap. The rules over
// checks run first, so a rule over statements knows which calls another fix
// rewrites. An annotation leaves out the reports that it covers, and the run
// reports each rule of an annotation that left out none.
func run(ap *analysis.Pass) (any, error) {
	in := ap.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	p := &pass{Pass: ap, rules: checkRules, fixed: make(map[*ast.CallExpr]bool)}
	p.skips = p.annotations()
	p.origins, p.lookups = origins(p, in)
	for cursor := range in.Root().Preorder((*ast.CallExpr)(nil), (*ast.IfStmt)(nil)) {
		if c, ok := p.check(cursor); ok {
			for _, rule := range p.rules {
				if rule(p, c) {
					break
				}
			}
		}
	}
	var nodes []ast.Node
	for _, r := range statementRules {
		nodes = append(nodes, r.nodes...)
	}
	for cursor := range in.Root().Preorder(nodes...) {
		kind := reflect.TypeOf(cursor.Node())
		for _, r := range statementRules {
			if slices.ContainsFunc(r.nodes, func(n ast.Node) bool { return reflect.TypeOf(n) == kind }) {
				r.rule(p, cursor)
			}
		}
	}
	p.reportUnused()
	return nil, nil //nolint:nilnil // the analyzer reports diagnostics and returns no result
}

// briefLength is the most characters of a node's text that a diagnostic
// quotes.
const briefLength = 60

// report reports a hand-written check of the rule at the node n, which the
// assertion states, with the fixes. A fix of a call of an assertion marks
// the call fixed. The assertion's text names the calls that the rule matched
// outside n, as in "Pure of r.Len(), around r.Next()".
func (p *pass) report(n ast.Node, rule, assertion string, fixes []analysis.SuggestedFix) {
	p.reportf(n, rule, "state the check with "+assertion, fixes)
}

// brief returns the text of the node n as a diagnostic quotes it, on one line
// with each run of white space as one space. A statement that only makes a
// call, such as _ = s.Put(k), quotes the call, and a call longer than
// briefLength characters quotes its function and an ellipsis, as
// sort.Slice(…). Any other text longer than that is cut with an ellipsis.
func (p *pass) brief(n ast.Node) string {
	switch s := n.(type) {
	case *ast.ExprStmt:
		return p.brief(s.X)
	case *ast.AssignStmt:
		if len(s.Rhs) == 1 && !slices.ContainsFunc(s.Lhs, func(lhs ast.Expr) bool {
			id, ok := lhs.(*ast.Ident)
			return !ok || id.Name != "_"
		}) {

			return p.brief(s.Rhs[0])
		}
	}
	text := []rune(strings.Join(strings.Fields(p.source(n)), " "))
	if len(text) <= briefLength {
		return string(text)
	}
	if call, ok := n.(*ast.CallExpr); ok {
		return p.brief(call.Fun) + "(…)"
	}
	return string(text[:briefLength-1]) + "…"
}

// named returns the assertion that the first rule over checks names for c,
// without a report, and True or False of c's condition where no rule names
// one.
func (p *pass) named(c check) string {
	name := falseName
	if c.holds {
		name = trueName
	}
	saved := p.naming
	p.naming = &name
	defer func() { p.naming = saved }()
	for _, rule := range p.rules {
		if rule(p, c) {
			break
		}
	}
	return name
}

// reportf reports a hand-written check of the rule at the node n with the
// message and the fixes, unless an annotation leaves the report out. While
// named asks the rules, it passes what the message states to named instead.
func (p *pass) reportf(n ast.Node, rule, message string, fixes []analysis.SuggestedFix) {
	if p.naming != nil {
		*p.naming = strings.TrimPrefix(message, "state the check with ")
		return
	}
	if p.skipped(n.Pos(), rule) {
		return
	}
	if call, ok := n.(*ast.CallExpr); ok && fixes != nil {
		p.fixed[call] = true
	}
	p.Report(analysis.Diagnostic{
		Pos:            n.Pos(),
		End:            n.End(),
		Message:        rule + ": " + message,
		SuggestedFixes: fixes,
	})
}
