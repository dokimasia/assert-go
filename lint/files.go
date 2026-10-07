// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import "go/ast"

// kinds maps each method that reports the kind of a file to the rule that
// reports a check of it and the assertion that states the check.
var kinds = map[string][2]string{
	"(io/fs.FileInfo).IsDir":     {"is-dir", "files.IsDir"},
	"(io/fs.FileMode).IsDir":     {"is-dir", "files.IsDir"},
	"(io/fs.FileMode).IsRegular": {"is-file", "files.IsFile"},
}

// pathAbsent reports True of os.IsNotExist(err), and a match of an error with
// fs.ErrNotExist or os.ErrNotExist, where the error comes from os.Stat in the
// check or in an earlier statement of its list. files.Absent states that
// nothing is at a path of the operating system. The error of a store or of
// an fs.FS that reports a missing name is none.
func pathAbsent(p *pass, c check) bool {
	err, target, matched := p.match(c)
	if n, notExist := p.callOf(c.cond, "os", "IsNotExist"); notExist && c.holds {
		err, matched = n.Args[0], true
	} else if !p.refers(target, "io/fs", "ErrNotExist") && !p.refers(target, "os", "ErrNotExist") {
		matched = false
	}
	stmt, listed := c.statement()
	if !matched || !listed {
		return false
	}
	n, _, ok := p.localSource(stmt, err)
	if !ok || p.callee(n) != "os.Stat" {
		return false
	}
	p.report(c.node, "path-absent", "files.Absent of "+p.brief(n), nil)
	return true
}

// fileKind reports True of IsDir of an fs.FileInfo or an fs.FileMode, and of
// IsRegular of an fs.FileMode. files.IsDir and files.IsFile state the kind of
// the entry at a path.
func fileKind(p *pass, c check) bool {
	n, ok := ast.Unparen(c.cond).(*ast.CallExpr)
	if !ok || !c.holds {
		return false
	}
	kind, ok := kinds[p.callee(n)]
	if ok {
		p.report(c.node, kind[0], kind[1], nil)
	}
	return ok
}

// hasMode reports an equality of Perm of an fs.FileMode. files.HasMode states
// the permission bits of the entry at a path.
func hasMode(p *pass, c check) bool {
	return p.reportEquality(c, "(io/fs.FileMode).Perm", "has-mode", "files.HasMode")
}

// linksTo reports an equality of a value that os.Readlink produces.
// files.LinksTo states the target of the link at a path.
func linksTo(p *pass, c check) bool {
	return p.reportEquality(c, "os.Readlink", "links-to", "files.LinksTo")
}

// reportEquality reports, under the rule, an equality of a value that the
// function name produces, which the assertion states.
func (p *pass) reportEquality(c check, name, rule, assertion string) bool {
	e, ok := p.equality(c)
	if !ok {
		return false
	}
	n := p.produced(e.x, name)
	if n == nil {
		n = p.produced(e.y, name)
	}
	if n == nil {
		return false
	}
	p.report(c.node, rule, assertion+" of "+p.brief(n), nil)
	return true
}
