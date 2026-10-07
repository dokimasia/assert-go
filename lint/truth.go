// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// conjunction reports a check that a conjunction is true or that a
// disjunction is false, such as True of a && b, False of a || b, or an if
// check of a || b, whose failure does not state which operand failed. It
// names the assertion that each operand takes alone, as the rules over
// checks name it for a check of the operand. An if check in a loop that
// sleeps is part of the wait that eventually reports.
//
// It suggests a True or a False of each operand, with the call's test and
// message, for a statement of the surface that stops the test. A failed
// assertion of that surface ends the test, so each operand runs only where
// the operands before it leave the check undecided, as && and || run it. The
// surface that records a failure continues, so an operand that an earlier
// one guards, such as p.N after p != nil, would run on a nil p, and the rule
// suggests no fix for it. The fix writes the test and the message once for
// each operand, so it also requires that neither calls a function or
// receives from a channel.
func conjunction(p *pass, c check) bool {
	op, assertion := token.LOR, "False"
	if c.holds {
		op, assertion = token.LAND, "True"
	}
	terms := operands(c.cond, op)
	if len(terms) < 2 || c.call == nil && p.polls(c.cursor) {
		return false
	}
	var fixes []analysis.SuggestedFix
	if c.stmt && c.name != nil && c.surface(assertPath) && pure(c.tb) && pure(c.call.Args[2]) {
		calls := make([]string, len(terms))
		tb, msg := p.source(c.tb), p.source(c.call.Args[2])
		for i, term := range terms {
			calls[i] = c.qualifier + "." + assertion + "(" + tb + ", " + p.source(term) + ", " + msg + ")"
		}
		fixes = replace("Call "+assertion+" for each operand", c.call, strings.Join(calls, "\n"))
	}
	names := make([]string, len(terms))
	for i, term := range terms {
		cond, holds := normalize(term, c.holds)
		names[i] = p.named(check{node: c.node, cursor: c.cursor, tb: c.tb, cond: cond, holds: holds}) + " for " +
			p.brief(term)
	}
	p.reportf(c.node, "conjunction", "state each operand in an assertion of its own: "+strings.Join(names, ", "), fixes)
	return true
}

// condition reports an if check that no other rule reports, which True or
// False of its condition states. An if check in a loop that sleeps is part of
// the wait that eventually reports.
func condition(p *pass, c check) bool {
	if c.call != nil || p.polls(c.cursor) {
		return false
	}
	name := "False"
	if c.holds {
		name = "True"
	}
	p.report(c.node, "condition", name, nil)
	return true
}
