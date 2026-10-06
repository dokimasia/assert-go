// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package equality

import (
	"math"
	"reflect"
)

// Rules are the relaxations of one comparison. The zero value relaxes
// nothing, which is the comparison that equal states by default.
type Rules struct {
	// EquateEmpty makes a nil slice or map equal an empty one of its type.
	EquateEmpty bool
	// EquateNaNs makes a NaN equal a NaN of its type.
	EquateNaNs bool
	// ByIdentity makes a pointer, a map and a slice equal another only when
	// both are the same object: the same address, and for a slice the same
	// length as well. The walk does not enter them, so EquateEmpty does not
	// apply to them. A channel and a function compare by their address
	// under every rule.
	ByIdentity bool
}

// stackBuffer is the most frames that a walk keeps on the stack of its
// goroutine before its own stack moves to the heap.
const stackBuffer = 16

// logBuffer is the room for visits that the log of a comparison takes at
// its first visit.
const logBuffer = 8

// Equal reports whether x and y are equal under r. A value of an interface
// type compares by the value inside it, and the invalid value compares as a
// nil interface.
//
// The walk compares the elements of arrays and slices, the fields of
// structs and the targets of pointers from a stack of its own, so the depth
// of a value costs heap memory and not the goroutine's stack. A map compares
// its entries in a nested walk.
//
// # Allocation contract
//
// Equal allocates nothing for two values without a map, nested at most 16
// levels deep, whose containers contain no container. A pair of containers
// whose parts can contain a container allocates the set of pairs that the
// walk entered, once.
func Equal(x, y reflect.Value, r Rules) bool {
	c := comparison{rules: r}
	return c.equal(x, y)
}

// frame is a pair of values whose parts the walk compares one at a time:
// the elements of two arrays or slices, the fields of two structs, or the
// targets of two pointers.
type frame struct {
	// x and y are the pair.
	x, y reflect.Value
	// next is the part that the walk compares next.
	next int
	// parts is the number of parts of each, and 0 for a pair that the walk
	// does not enter.
	parts int
}

// visit is a pair of pointers, maps or slices that the walk entered: the
// addresses of both, their type, and for slices their length.
type visit struct {
	x, y uintptr
	typ  reflect.Type
	len  int
}

// comparison is one comparison under its rules.
type comparison struct {
	// rules are the relaxations of the comparison.
	rules Rules
	// visited are the pairs of containers that the walk entered, which it
	// takes as equal when it meets them again.
	visited map[visit]struct{}
	// log are the visits in the order that the walk recorded them, so a
	// trial that fails forgets the ones that it recorded.
	log []visit
}

// equal reports whether x and y are equal, walking their parts with a stack
// of frames of its own. A frame leaves the stack when its last part begins,
// so the parts of a list in its last field add no frame for each node.
func (c *comparison) equal(x, y reflect.Value) bool {
	var buf [stackBuffer]frame
	stack := buf[:0]
	f, same := c.begin(x, y)
	for {
		if f.parts > 0 {
			stack = append(stack, f)
		}
		if !same || len(stack) == 0 {
			return same
		}
		top := len(stack) - 1
		f = stack[top]
		if f.next+1 == f.parts {
			stack = stack[:top]
		} else {
			stack[top].next++
		}
		f, same = c.begin(part(f.x, f.next), part(f.y, f.next))
	}
}

// begin compares x and y themselves, and returns the frame of their parts,
// which has none for a pair without parts or that the walk entered before.
// It reports false when x and y differ in themselves.
func (c *comparison) begin(x, y reflect.Value) (frame, bool) {
	x, y = inside(x), inside(y)
	if !x.IsValid() || !y.IsValid() {
		return frame{}, x.IsValid() == y.IsValid()
	}
	if x.Type() != y.Type() {
		return frame{}, false
	}
	if c.rules.ByIdentity && reference(x.Kind()) {
		return frame{}, sameObject(x, y)
	}
	switch x.Kind() {
	case reflect.Bool:
		return frame{}, x.Bool() == y.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return frame{}, x.Int() == y.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return frame{}, x.Uint() == y.Uint()
	case reflect.Float32, reflect.Float64:
		return frame{}, c.floats(x.Float(), y.Float())
	case reflect.Complex64, reflect.Complex128:
		a, b := x.Complex(), y.Complex()
		return frame{}, c.floats(real(a), real(b)) && c.floats(imag(a), imag(b))
	case reflect.String:
		return frame{}, x.String() == y.String()
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return frame{}, x.Pointer() == y.Pointer()
	case reflect.Pointer:
		return c.pointers(x, y)
	case reflect.Array:
		return frame{x: x, y: y, parts: x.Len()}, true
	case reflect.Struct:
		return frame{x: x, y: y, parts: x.NumField()}, true
	case reflect.Slice:
		return c.slices(x, y)
	}
	// A map is the one kind left: inside unwraps every interface.
	return frame{}, c.maps(x, y)
}

// reference reports whether a value of kind k is a pointer, a map or a
// slice, which ByIdentity compares by the object it refers to.
func reference(k reflect.Kind) bool {
	return k == reflect.Pointer || k == reflect.Map || k == reflect.Slice
}

// sameObject reports whether x and y, two pointers, maps or slices of one
// type, refer to the same object: the same address, and for slices the
// same length.
func sameObject(x, y reflect.Value) bool {
	return x.Pointer() == y.Pointer() && (x.Kind() != reflect.Slice || x.Len() == y.Len())
}

// floats reports whether a equals b: by value, and a NaN equals a NaN under
// EquateNaNs alone.
func (c *comparison) floats(a, b float64) bool {
	return a == b || c.rules.EquateNaNs && math.IsNaN(a) && math.IsNaN(b)
}

// pointers compares two pointers: both nil, or targets that compare equal.
func (c *comparison) pointers(x, y reflect.Value) (frame, bool) {
	if x.IsNil() || y.IsNil() {
		return frame{}, x.IsNil() && y.IsNil()
	}
	if c.met(x, y) {
		return frame{}, true
	}
	return frame{x: x, y: y, parts: 1}, true
}

// slices compares two slices: both nil, or neither nil with equal elements.
// Under EquateEmpty, nil equals empty.
func (c *comparison) slices(x, y reflect.Value) (frame, bool) {
	if x.IsNil() != y.IsNil() {
		return frame{}, c.rules.EquateEmpty && x.Len() == 0 && y.Len() == 0
	}
	if x.Len() != y.Len() {
		return frame{}, false
	}
	if x.Len() == 0 || c.met(x, y) {
		return frame{}, true
	}
	return frame{x: x, y: y, parts: x.Len()}, true
}

// met reports whether the walk entered the pair of containers x and y
// before, and records the pair when it did not. A pair whose parts cannot
// contain a container cannot be met again inside itself, and is not
// recorded.
func (c *comparison) met(x, y reflect.Value) bool {
	if !partsRepeat(x.Type()) {
		return false
	}
	v := visit{x: x.Pointer(), y: y.Pointer(), typ: x.Type()}
	if x.Kind() == reflect.Slice {
		v.len = x.Len()
	}
	if _, seen := c.visited[v]; seen {
		return true
	}
	if c.visited == nil {
		c.visited = make(map[visit]struct{})
		c.log = make([]visit, 0, logBuffer)
	}
	c.visited[v] = struct{}{}
	c.log = append(c.log, v)
	return false
}

// try runs compare as a trial, and reports what compare reports. A trial
// that fails forgets the pairs of containers that it recorded, which it
// assumed equal and which can differ.
func (c *comparison) try(compare func() bool) bool {
	mark := len(c.log)
	if compare() {
		return true
	}
	for _, v := range c.log[mark:] {
		delete(c.visited, v)
	}
	c.log = c.log[:mark]
	return false
}

// part returns the part i of v: the element of an array or a slice, the
// field of a struct, or the target of a pointer.
func part(v reflect.Value, i int) reflect.Value {
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		return v.Index(i)
	case reflect.Struct:
		return v.Field(i)
	}
	return v.Elem()
}

// inside returns the value inside v when v is of an interface type, which
// is the invalid value for a nil interface, and v otherwise.
func inside(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

// partsRepeat reports whether the parts of a container of type t, a
// pointer's target, a slice's elements, or a map's keys and values, can
// contain a container.
func partsRepeat(t reflect.Type) bool {
	if t.Kind() == reflect.Map {
		return canRepeat(t.Key()) || canRepeat(t.Elem())
	}
	return canRepeat(t.Elem())
}

// canRepeat reports whether a value of type t can contain a pointer, a map,
// a slice or an interface, through which a walk can meet a container that
// encloses it.
func canRepeat(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return true
	case reflect.Array:
		return canRepeat(t.Elem())
	case reflect.Struct:
		zero := reflect.Zero(t)
		for i := range t.NumField() {
			if canRepeat(zero.Field(i).Type()) {
				return true
			}
		}
	}
	return false
}
