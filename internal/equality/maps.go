// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package equality

import "reflect"

// HasKey reports whether the map m has a key that equals key under r, as
// [Equal] compares them.
//
// A needle that no key of m can equal, such as an int64 in a map[int]bool,
// is not found at once. A needle whose type's == agrees with equal is
// looked up with a map index. Every other needle compares with each key, so
// a needle that Go cannot hash, such as a slice, is never hashed.
//
// # Allocation contract
//
// HasKey allocates nothing for a needle that a map index looks up.
//
// # Panics
//
// HasKey panics for an m that is no map, as [reflect.Value.MapIndex] does.
func HasKey(m, key reflect.Value, r Rules) bool {
	key = inside(key)
	if !accepts(m.Type().Key(), key) {
		return false
	}
	if key.IsValid() && lookup(key.Type(), r) {
		return m.MapIndex(key).IsValid()
	}
	c := comparison{rules: r}
	for it := m.MapRange(); it.Next(); {
		if c.try(func() bool { return c.equal(it.Key(), key) }) {
			return true
		}
	}
	return false
}

// maps compares two maps of one type: both nil, or neither nil with a
// one-to-one matching of their entries. Under EquateEmpty, nil equals
// empty.
func (c *comparison) maps(x, y reflect.Value) bool {
	if x.IsNil() != y.IsNil() {
		return c.rules.EquateEmpty && x.Len() == 0 && y.Len() == 0
	}
	if x.Len() != y.Len() {
		return false
	}
	if x.Len() == 0 || c.met(x, y) {
		return true
	}
	if c.keysLookUp(x) {
		return c.lookUp(x, y)
	}
	return c.match(x, y)
}

// keysLookUp reports whether every key of x is of a type whose == agrees
// with equal, so a map index of y finds each key of y that equals it.
func (c *comparison) keysLookUp(x reflect.Value) bool {
	t := x.Type().Key()
	if t.Kind() != reflect.Interface {
		return lookup(t, c.rules)
	}
	for it := x.MapRange(); it.Next(); {
		k := inside(it.Key())
		if k.IsValid() && !lookup(k.Type(), c.rules) {
			return false
		}
	}
	return true
}

// lookUp compares the entries of x and y by looking each key of x up in y.
func (c *comparison) lookUp(x, y reflect.Value) bool {
	for it := x.MapRange(); it.Next(); {
		w := y.MapIndex(it.Key())
		if !w.IsValid() || !c.equal(it.Value(), w) {
			return false
		}
	}
	return true
}

// match compares the entries of x and y by matching: each entry of x takes
// the first unmatched entry of y whose key and value equal its own. Equal
// is an equivalence on the values that equal themselves, so a greedy match
// is exact.
func (c *comparison) match(x, y reflect.Value) bool {
	keys := make([]reflect.Value, 0, y.Len())
	values := make([]reflect.Value, 0, y.Len())
	for it := y.MapRange(); it.Next(); {
		keys = append(keys, it.Key())
		values = append(values, it.Value())
	}
	matched := make([]bool, len(keys))
	for it := x.MapRange(); it.Next(); {
		if !c.matchEntry(it.Key(), it.Value(), keys, values, matched) {
			return false
		}
	}
	return true
}

// matchEntry matches the entry of the key k and the value v with the first
// unmatched entry of keys and values that equals it, and reports whether
// one does.
func (c *comparison) matchEntry(k, v reflect.Value, keys, values []reflect.Value, matched []bool) bool {
	for j := range keys {
		if !matched[j] && c.try(func() bool { return c.equal(k, keys[j]) && c.equal(v, values[j]) }) {
			matched[j] = true
			return true
		}
	}
	return false
}

// accepts reports whether a key of keyType can equal key: key is of
// keyType, or keyType is an interface that key's type implements, or whose
// nil a nil key equals.
func accepts(keyType reflect.Type, key reflect.Value) bool {
	if keyType.Kind() == reflect.Interface {
		return !key.IsValid() || key.Type().Implements(keyType)
	}
	return key.IsValid() && key.Type() == keyType
}

// lookup reports whether == on values of type t agrees with equal under r,
// so a map index finds each key of type t that equals a needle: a bool, an
// integer, a string, a channel or an unsafe pointer, a float or a complex
// number without EquateNaNs, and an array or a struct of those.
func lookup(t reflect.Type, r Rules) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.String, reflect.Chan, reflect.UnsafePointer,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return !r.EquateNaNs
	case reflect.Array:
		return lookup(t.Elem(), r)
	case reflect.Struct:
		zero := reflect.Zero(t)
		for i := range t.NumField() {
			if !lookup(zero.Field(i).Type(), r) {
				return false
			}
		}
		return true
	}
	return false
}
