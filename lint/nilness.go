// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/token"
	"go/types"
)

// errorType is the interface error.
var errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// nilCheck reports True or False of a comparison of a value with nil.
//
// The rule nil suggests Nil or NotNil for a value of a type whose nil is no
// interface: a pointer, a slice, a map, a channel, a function or an
// unsafe.Pointer. The rule error-nil suggests NoError or HasError for a value
// of an interface type that implements error. Both compare such a value with
// nil as == does. Neither reports a value of another interface type: ==
// counts a typed nil in an interface as present, and Nil counts it as nil.
func nilCheck(p *pass, c check) bool {
	x, y, op, ok := c.comparison()
	if ok && p.TypesInfo.Types[x].IsNil() {
		x, y = y, x
	}
	if !ok || !p.TypesInfo.Types[y].IsNil() {
		return false
	}
	rule, name, ok := nilAssertion(p.TypesInfo.TypeOf(x), op == token.EQL)
	if !ok {
		return false
	}
	p.report(c.node, rule, name, c.rewrite(name, p.source(x)))
	return true
}

// equalNil reports Equal or NotEqual of a value and nil, without options, and
// suggests the assertion that nilCheck suggests for the value's type. Equal
// compares such a value with nil as == does.
func equalNil(p *pass, c check) bool {
	if !c.is(equalName, notEqualName) || len(c.call.Args) != 4 {
		return false
	}
	x, y := c.call.Args[1], c.call.Args[2]
	if p.TypesInfo.Types[x].IsNil() {
		x, y = y, x
	}
	if !p.TypesInfo.Types[y].IsNil() {
		return false
	}
	_, name, ok := nilAssertion(p.TypesInfo.TypeOf(x), c.is(equalName))
	if !ok {
		return false
	}
	p.report(c.node, "equal-nil", name, c.rewrite(name, p.source(x)))
	return true
}

// nilAssertion returns the rule and the assertion of a check that a value
// of type t is nil, or that it is not where isNil is false, and false for a
// type of neither rule.
func nilAssertion(t types.Type, isNil bool) (rule, name string, ok bool) {
	names := [2]string{"Nil", "NotNil"}
	rule = "nil"
	switch {
	case isError(t):
		names, rule = [2]string{"NoError", "HasError"}, "error-nil"
	case !nilable(t):
		return "", "", false
	}
	if isNil {
		return rule, names[0], true
	}
	return rule, names[1], true
}

// nilable reports whether t is a type that has nil and is no interface: a
// pointer, a slice, a map, a channel, a function or an unsafe.Pointer.
func nilable(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature:
		return true
	case *types.Basic:
		return u.Kind() == types.UnsafePointer
	}
	return false
}

// isError reports whether t is an interface type that implements error. A
// type parameter is none, although its constraint is an interface.
func isError(t types.Type) bool {
	_, param := t.(*types.TypeParam)
	return !param && types.IsInterface(t) && types.Implements(t, errorType)
}
