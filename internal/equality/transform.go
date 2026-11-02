// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality

import (
	"reflect"
	"slices"
)

// Transform is a transform of the values of one type. A comparison under it
// compares those values by their results.
type Transform struct {
	// In is the type of the values that the transform takes. A value of type
	// In matches it, and so does a value whose type implements In when In is
	// an interface.
	In reflect.Type
	// Apply returns the result of a value that matches. The value is one that
	// Go lets a function receive, also when it is in an unexported field.
	Apply func(reflect.Value) reflect.Value
}

// WithTransform returns r with t, in place of a transform of r of the same
// type. A value compares by the last transform of r that it matches.
//
// # Allocation contract
//
// WithTransform allocates the list of transforms when it grows.
func (r Rules) WithTransform(t Transform) Rules {
	kept := r.Transforms[:0:0]
	for _, u := range r.Transforms {
		if u.In != t.In {
			kept = append(kept, u)
		}
	}
	kept = append(kept, t)
	r.Transforms = kept
	return r
}

// Result returns the result of the last transform of r that v matches, and v
// when v matches none. A value of an interface type matches as the value
// inside it.
//
// # Allocation contract
//
// Result allocates what the transform allocates, and nothing for a value
// that matches no transform.
func Result(v reflect.Value, r Rules) reflect.Value {
	inner := inside(v)
	if !inner.IsValid() {
		return v
	}
	if u, ok := r.matching(inner.Type()); ok {
		return u.Apply(inner)
	}
	return v
}

// matching returns the last transform of r that a value of type t matches,
// and false when none does.
func (r Rules) matching(t reflect.Type) (Transform, bool) {
	for _, u := range slices.Backward(r.Transforms) {
		if t == u.In || u.In.Kind() == reflect.Interface && t.Implements(u.In) {
			return u, true
		}
	}
	return Transform{}, false
}

// expose returns v under r as a value that Go lets a function receive. Under
// a transform, a value of an unexported field becomes a value of the same
// memory without the restriction of its field. Such a value has an address,
// because the walk reads the fields of a struct from memory that address
// returns. An element, a target and an entry of a value without the
// restriction carry none either, so they need no address.
func (r Rules) expose(v reflect.Value) reflect.Value {
	if len(r.Transforms) == 0 || !v.IsValid() || v.CanInterface() {
		return v
	}
	return reflect.NewAt(v.Type(), v.Addr().UnsafePointer()).Elem()
}

// address returns v under r as the walk reads the fields of a struct: under
// a transform, in memory that the walk can address, so expose can hand the
// value of an unexported field to the transform.
func (r Rules) address(v reflect.Value) reflect.Value {
	//dokimi:mutate-skip sbr-delete,lcr-left,lcr-right,ror-false: a copy that no transform reads changes only the allocations
	if len(r.Transforms) == 0 || v.CanAddr() {
		return v
	}
	c := reflect.New(v.Type()).Elem()
	c.Set(v)
	return c
}
