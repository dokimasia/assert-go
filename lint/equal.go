// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// collectionEquals are the functions that report whether two slices or maps
// are equal, counting a nil value equal to an empty one.
var collectionEquals = [][2]string{{"bytes", "Equal"}, {"slices", "Equal"}, {"maps", "Equal"}}

// compare reports True or False of == or != between two values of basic
// types: booleans, numbers and strings. It suggests Equal or NotEqual, which
// compare such values as == does: a NaN equals no number, and -0 equals +0.
// A constant operand becomes the value that the test wants.
func compare(p *pass, c check) bool {
	x, y, op, ok := c.comparison()
	if !ok || op != token.EQL && op != token.NEQ || !basic(p.TypesInfo.TypeOf(x)) || !basic(p.TypesInfo.TypeOf(y)) {
		return false
	}
	if p.constant(x) {
		x, y = y, x
	}
	name := "Equal"
	if op == token.NEQ {
		name = "NotEqual"
	}
	p.report(c.node, "compare", name, c.rewrite(name, p.source(x)+", "+p.source(y)))
	return true
}

// equalFunc reports True or False of bytes.Equal, or of slices.Equal or
// maps.Equal over basic elements and keys, and suggests Equal or NotEqual
// with EquateEmpty. The three functions count a nil value equal to an empty
// one, and Equal counts them equal under EquateEmpty alone. It suggests no
// fix for two values of different types, such as a json.RawMessage and a
// []byte, because Equal compares two values of one type.
func equalFunc(p *pass, c check) bool {
	for _, f := range collectionEquals {
		n, ok := p.callOf(c.cond, f[0], f[1])
		if !ok || !basicElements(p.TypesInfo.TypeOf(n.Fun).(*types.Signature).Params().At(0).Type()) {
			continue
		}
		name := "Equal"
		if !c.holds {
			name = "NotEqual"
		}
		var fixes []analysis.SuggestedFix
		if types.Identical(p.TypesInfo.TypeOf(n.Args[0]), p.TypesInfo.TypeOf(n.Args[1])) {
			fixes = c.rewrite(name, p.source(n.Args[0])+", "+p.source(n.Args[1]), "EquateEmpty")
		}
		p.report(c.node, "equal-func", name+" and EquateEmpty", fixes)
		return true
	}
	return false
}

// deepEqual reports True or False of reflect.DeepEqual, and suggests Equal or
// NotEqual. It suggests no fix where the two values have different types, or
// a type on which the two comparisons can disagree: see deepSafe.
func deepEqual(p *pass, c check) bool {
	n, ok := p.callOf(c.cond, "reflect", "DeepEqual")
	if !ok {
		return false
	}
	name := "Equal"
	if !c.holds {
		name = "NotEqual"
	}
	var fixes []analysis.SuggestedFix
	t := p.TypesInfo.TypeOf(n.Args[0])
	if types.Identical(t, p.TypesInfo.TypeOf(n.Args[1])) && deepSafe(t, false, make(map[types.Type]bool)) {
		fixes = c.rewrite(name, p.source(n.Args[0])+", "+p.source(n.Args[1]))
	}
	p.report(c.node, "deep-equal", name, fixes)
	return true
}

// deepSafe reports whether reflect.DeepEqual and Equal agree on every two
// values of type t. They disagree on a float, through a pointer that
// DeepEqual counts equal to itself whatever it points to, on a function,
// which DeepEqual counts unequal to every function, on an interface, whose
// values can be of any type, and on a pointer in a map key, which DeepEqual
// finds by address. inKey reports whether t is part of a map key. seen
// contains the types that the walk entered.
func deepSafe(t types.Type, inKey bool, seen map[types.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return u.Info()&(types.IsFloat|types.IsComplex|types.IsUntyped) == 0 && u.Kind() != types.UnsafePointer
	case *types.Pointer:
		return !inKey && deepSafe(u.Elem(), inKey, seen)
	case *types.Slice:
		return deepSafe(u.Elem(), inKey, seen)
	case *types.Array:
		return deepSafe(u.Elem(), inKey, seen)
	case *types.Map:
		return deepSafe(u.Key(), true, seen) && deepSafe(u.Elem(), inKey, seen)
	case *types.Chan:
		return true
	case *types.Struct:
		for field := range u.Fields() {
			if !deepSafe(field.Type(), inKey, seen) {
				return false
			}
		}
		return true
	}
	return false
}

// basic reports whether t is a boolean, numeric or string type.
func basic(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&(types.IsBoolean|types.IsNumeric|types.IsString) != 0
}

// basicElements reports whether t is a slice of basic elements, or a map of
// basic keys and values.
func basicElements(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Slice:
		return basic(u.Elem())
	case *types.Map:
		return basic(u.Key()) && basic(u.Elem())
	}
	return false
}
