// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package equality

import (
	"cmp"
	"reflect"
	"slices"

	"go.dokimi.dev/assert/internal/align"
)

// maxDepth is the most levels that the walk of Diff descends below the
// roots. It states the two values at that level whole.
const maxDepth = 1000

// Difference is one place where two values differ.
type Difference struct {
	// Path are the steps from the roots of the two values to the place.
	Path []Step
	// X and Y are the values at the place. A side without the place, as
	// for an element that only the other side has, is the invalid value.
	X, Y reflect.Value
}

// Step is one step of a path: a field of a struct, an element of an array
// or a slice, or the value of a map's key.
type Step struct {
	// Field is the name of a field, and empty for another step.
	Field string
	// Index is the index of an element, and -1 for another step.
	Index int
	// Key is the key of a map entry, and the invalid value for another
	// step.
	Key reflect.Value
}

// Diff returns the places where x and y differ under r, at most limit of
// them, in the order of a walk of x and y. It returns none for two equal
// values.
//
// The walk descends into the parts of two values of one type, and
// compares each part that it does not descend into as [Equal] does:
//
//   - Values of different types, scalars, functions and channels differ
//     whole, and so do a nil pointer, slice or map and a non-nil one.
//   - Pointers descend into their targets, and structs and arrays into
//     their fields and elements.
//   - Slices align their elements by their longest common subsequence. An
//     element that one side has alone differs at its index in that side,
//     with an invalid other side. A run of elements that each side has
//     alone pairs up in order, and each pair descends at its index in x.
//   - Maps pair their entries by key: through a map index when == agrees
//     with Equal on every key of both maps, and otherwise each entry of x
//     with an equal entry of y first, and then with an entry of an equal
//     key. The walk descends into the values of each pair. An entry that
//     one side has alone differs at its key, with an invalid other side.
//     The places of a map follow the order of its keys.
//
// A nil interface is a value of its own, and never an invalid side. The
// walk descends into each pair of pointers, maps or slices once, so two
// values that contain themselves differ in finitely many places. It
// descends at most 1,000 levels, and states the two values there whole.
// Outside the alignment of slices and the pairing of map entries without a
// map index, the walk visits each part once.
//
// # Allocation contract
//
// Diff allocates its differences and their paths, a step of a path for
// each part that it descends into, the alignment of each pair of slices
// and the entries of each pair of maps, and what Equal allocates for each
// part that it compares.
func Diff(x, y reflect.Value, r Rules, limit int) []Difference {
	d := differ{rules: r, limit: limit}
	d.walk(nil, x, y, 0)
	return d.found
}

// differ is one walk of Diff.
type differ struct {
	// rules are the relaxations of the comparison.
	rules Rules
	// limit is the most differences that the walk finds.
	limit int
	// found are the differences found so far.
	found []Difference
	// entered are the pairs of containers that the walk descended into.
	entered map[visit]struct{}
}

// trail is the path of a place as a chain from its last step to the roots,
// so a step adds one link and shares the steps before it. The roots have
// the nil trail.
type trail struct {
	// up is the trail of the place one step above, and nil below the roots.
	up *trail
	// step is the last step of the path.
	step Step
}

// to returns the trail of the place one step s below t.
func (t *trail) to(s Step) *trail {
	return &trail{up: t, step: s}
}

// steps returns the steps of t from the roots.
func (t *trail) steps() []Step {
	n := 0
	for u := t; u != nil; u = u.up {
		n++
	}
	out := make([]Step, n)
	for u := t; u != nil; u = u.up {
		n--
		out[n] = u.step
	}
	return out
}

// walk adds the differences of x and y at path, depth levels below the
// roots.
func (d *differ) walk(path *trail, x, y reflect.Value, depth int) {
	if len(d.found) == d.limit {
		return
	}
	ix, iy := inside(x), inside(y)
	if depth == maxDepth || !ix.IsValid() || !iy.IsValid() || ix.Type() != iy.Type() {
		d.whole(path, x, y)
		return
	}
	switch ix.Kind() {
	case reflect.Pointer:
		d.pointers(path, ix, iy, depth)
	case reflect.Struct:
		for i := range ix.NumField() {
			d.walk(path.to(Step{Field: ix.Type().Field(i).Name, Index: -1}), ix.Field(i), iy.Field(i), depth+1)
		}
	case reflect.Array:
		for i := range ix.Len() {
			d.walk(path.to(Step{Index: i}), ix.Index(i), iy.Index(i), depth+1)
		}
	case reflect.Slice:
		d.slices(path, ix, iy, depth)
	case reflect.Map:
		d.maps(path, ix, iy, depth)
	default:
		d.whole(path, x, y)
	}
}

// whole adds x and y as one difference at path when [Equal] reports them
// unequal.
func (d *differ) whole(path *trail, x, y reflect.Value) {
	if !Equal(x, y, d.rules) {
		d.add(path, x, y)
	}
}

// pointers adds the differences of two pointers: the two whole when one is
// nil, and else the differences of their targets.
func (d *differ) pointers(path *trail, x, y reflect.Value, depth int) {
	if x.IsNil() || y.IsNil() {
		d.whole(path, x, y)
		return
	}
	if d.descended(x, y) {
		return
	}
	d.walk(path, x.Elem(), y.Elem(), depth+1)
}

// slices adds the differences of two slices: the two whole when one is
// nil, and else the differences of their aligned elements.
func (d *differ) slices(path *trail, x, y reflect.Value, depth int) {
	if x.IsNil() || y.IsNil() {
		d.whole(path, x, y)
		return
	}
	if d.descended(x, y) {
		return
	}
	edits := align.Edits(x.Len(), y.Len(), func(i, j int) bool { return Equal(x.Index(i), y.Index(j), d.rules) })
	var deleted, inserted []int
	for k, e := range edits {
		switch e.Op {
		case align.Delete:
			deleted = append(deleted, e.X)
		case align.Insert:
			inserted = append(inserted, e.Y)
		}
		if k+1 == len(edits) || edits[k+1].Op == align.Keep {
			d.run(path, x, y, deleted, inserted, depth)
			deleted, inserted = deleted[:0], inserted[:0]
		}
	}
}

// run adds the differences of a run of the elements deleted of x and
// inserted of y between two aligned elements: each pair in order, and then
// the elements left without a pair.
func (d *differ) run(path *trail, x, y reflect.Value, deleted, inserted []int, depth int) {
	pairs := min(len(deleted), len(inserted))
	for k := range pairs {
		d.walk(path.to(Step{Index: deleted[k]}), x.Index(deleted[k]), y.Index(inserted[k]), depth+1)
	}
	for _, i := range deleted[pairs:] {
		d.add(path.to(Step{Index: i}), x.Index(i), reflect.Value{})
	}
	for _, j := range inserted[pairs:] {
		d.add(path.to(Step{Index: j}), reflect.Value{}, y.Index(j))
	}
}

// entry is a key of one of two maps, and its value in each map, which is
// the invalid value in a map without the key.
type entry struct {
	key, x, y reflect.Value
}

// maps adds the differences of two maps: the two whole when one is nil,
// and else the differences of their paired entries, in the order of their
// keys.
func (d *differ) maps(path *trail, x, y reflect.Value, depth int) {
	if x.IsNil() || y.IsNil() {
		d.whole(path, x, y)
		return
	}
	if d.descended(x, y) {
		return
	}
	entries := d.entries(x, y)
	slices.SortStableFunc(entries, func(a, b entry) int { return order(a.key, b.key) })
	for _, e := range entries {
		where := path.to(Step{Index: -1, Key: e.key})
		if e.x.IsValid() && e.y.IsValid() {
			d.walk(where, e.x, e.y, depth+1)
			continue
		}
		d.add(where, e.x, e.y)
	}
}

// entries returns the entries of the maps x and y paired by key, the
// entries of x first. When == agrees with Equal on every key of both maps,
// a map index pairs them. Otherwise each entry of x pairs with an equal
// entry of y first, and then with an entry of an equal key.
func (d *differ) entries(x, y reflect.Value) []entry {
	var entries []entry
	if c := (comparison{rules: d.rules}); c.keysLookUp(x) && c.keysLookUp(y) {
		for it := x.MapRange(); it.Next(); {
			entries = append(entries, entry{key: it.Key(), x: it.Value(), y: y.MapIndex(it.Key())})
		}
		for it := y.MapRange(); it.Next(); {
			if !x.MapIndex(it.Key()).IsValid() {
				entries = append(entries, entry{key: it.Key(), y: it.Value()})
			}
		}
		return entries
	}
	var rest []entry
	for it := x.MapRange(); it.Next(); {
		entries = append(entries, entry{key: it.Key(), x: it.Value()})
	}
	for it := y.MapRange(); it.Next(); {
		rest = append(rest, entry{key: it.Key(), y: it.Value()})
	}
	rest = d.pair(entries, rest, true)
	return append(entries, d.pair(entries, rest, false)...)
}

// pair gives each entry of ys to the first entry of entries without a
// value of y whose key equals its key, and whose value equals its value as
// well when whole is true. It returns the entries of ys that it gave to
// none.
func (d *differ) pair(entries, ys []entry, whole bool) []entry {
	rest := ys[:0]
	for _, e := range ys {
		i := slices.IndexFunc(entries, func(f entry) bool {
			return !f.y.IsValid() && Equal(f.key, e.key, d.rules) && (!whole || Equal(f.x, e.y, d.rules))
		})
		if i < 0 {
			rest = append(rest, e)
			continue
		}
		entries[i].y = e.y
	}
	return rest
}

// descended reports whether the walk descended into the pair of containers
// x and y before, and records the pair when it did not.
func (d *differ) descended(x, y reflect.Value) bool {
	v := visit{x: x.Pointer(), y: y.Pointer(), typ: x.Type()}
	if x.Kind() == reflect.Slice {
		v.len = x.Len()
	}
	if _, seen := d.entered[v]; seen {
		return true
	}
	if d.entered == nil {
		d.entered = make(map[visit]struct{})
	}
	d.entered[v] = struct{}{}
	return false
}

// add adds the difference of x and y at path, unless the walk found as many
// as its limit.
func (d *differ) add(path *trail, x, y reflect.Value) {
	if len(d.found) != d.limit {
		d.found = append(d.found, Difference{Path: path.steps(), X: x, Y: y})
	}
}

// order returns -1, 0 or +1 as the map key x orders before, with or after
// the map key y: by type first, then by value for numbers, strings and
// bools, by address for pointers and channels, and part by part for
// arrays, structs and interfaces. A nil interface orders first.
func order(x, y reflect.Value) int {
	x, y = inside(x), inside(y)
	if !x.IsValid() || !y.IsValid() {
		return cmp.Compare(valid(x), valid(y))
	}
	if x.Type() != y.Type() {
		return cmp.Compare(x.Type().String(), y.Type().String())
	}
	switch x.Kind() {
	case reflect.Bool:
		return cmp.Compare(bit(x.Bool()), bit(y.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmp.Compare(x.Int(), y.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return cmp.Compare(x.Uint(), y.Uint())
	case reflect.Float32, reflect.Float64:
		return cmp.Compare(x.Float(), y.Float())
	case reflect.Complex64, reflect.Complex128:
		a, b := x.Complex(), y.Complex()
		return cmp.Or(cmp.Compare(real(a), real(b)), cmp.Compare(imag(a), imag(b)))
	case reflect.String:
		return cmp.Compare(x.String(), y.String())
	case reflect.Array, reflect.Struct:
		for i := range parts(x) {
			if c := order(part(x, i), part(y, i)); c != 0 {
				return c
			}
		}
		return 0
	}
	// A pointer, a channel and an unsafe pointer are the kinds left that a
	// map key can be.
	return cmp.Compare(x.Pointer(), y.Pointer())
}

// parts returns the number of parts of an array or a struct.
func parts(v reflect.Value) int {
	if v.Kind() == reflect.Array {
		return v.Len()
	}
	return v.NumField()
}

// valid returns 1 for a valid value, and 0 for the invalid value, which
// orders first.
func valid(v reflect.Value) int {
	if v.IsValid() {
		return 1
	}
	return 0
}

// bit returns 1 for true and 0 for false, which orders first.
func bit(b bool) int {
	if b {
		return 1
	}
	return 0
}
