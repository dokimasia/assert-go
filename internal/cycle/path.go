// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package cycle

import (
	"reflect"
	"slices"
)

// Path is the pointers, maps and slices that enclose the position of a
// walk, outermost first. The zero value is an empty path.
type Path struct {
	inside []container
}

// container is one pointer, map or slice: its type, the address of its
// data, and for a slice its length, so a slice and a shorter slice of the
// same array are two containers.
type container struct {
	typ  reflect.Type
	data uintptr
	len  int
}

// Enter adds v, a pointer, a map or a slice, to the path, and reports true.
// When the path contains v already, Enter adds nothing, and returns the
// number of steps back to v and false: 1 for the container that the walk
// entered last.
//
// # Allocation contract
//
// Enter allocates only when the path grows past the most containers that
// it has contained at once.
//
// # Panics
//
// Enter panics for a v of another kind, as [reflect.Value.Pointer] does.
func (p *Path) Enter(v reflect.Value) (back int, entered bool) {
	c := container{typ: v.Type(), data: v.Pointer()}
	if v.Kind() == reflect.Slice {
		c.len = v.Len()
	}
	for i, inner := range slices.Backward(p.inside) {
		if inner == c {
			return len(p.inside) - i, false
		}
	}
	p.inside = append(p.inside, c)
	return 0, true
}

// Leave removes the container that the walk entered last.
//
// # Allocation contract
//
// Leave allocates nothing.
//
// # Panics
//
// Leave panics on an empty path. A walk that calls Leave once for each
// Enter that reports true never meets one.
func (p *Path) Leave() {
	p.inside = p.inside[:len(p.inside)-1]
}
