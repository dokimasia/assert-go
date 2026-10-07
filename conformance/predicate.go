// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"bytes"
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"strings"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
)

// The predicates that the corpus's bodies and filters state, by kind.
const (
	alwaysKind        = "always"
	neverKind         = "never"
	equalsKind        = "equals"
	atLeastKind       = "at-least"
	divisibleByKind   = "divisible-by"
	sumAboveKind      = "sum-above"
	lengthAtLeastKind = "length-at-least"
	containsKind      = "contains"
	notSortedKind     = "not-sorted"
	hasDuplicateKind  = "has-duplicate"
	indexedAboveKind  = "indexed-above"
)

// predicateSpec is a predicate as the corpus states it.
type predicateSpec struct {
	// Kind names the predicate.
	Kind string `json:"kind"`
	// N is the number of at-least, divisible-by, sum-above,
	// length-at-least and indexed-above.
	N json.RawMessage `json:"n"`
	// Value is the typed literal of equals and contains.
	Value json.RawMessage `json:"value"`
	// Not negates the predicate.
	Not bool `json:"not"`
}

// predicateOf returns the predicate that a corpus spec states. Equal means
// the same canonical text, and a number compares exactly whatever its type,
// as the definition's executable reference compares it.
//
// It returns a fault for a spec that names no predicate of the vocabulary
// or misstates a parameter, whose path leads through the spec to the part
// at fault.
func predicateOf(raw json.RawMessage) (func(any) bool, error) {
	var spec predicateSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fault.New("the predicate does not parse").Because(err)
	}
	holds, err := holdsOf(spec)
	if err != nil {
		return nil, err
	}
	if spec.Not {
		return func(v any) bool { return !holds(v) }, nil
	}
	return holds, nil
}

// holdsOf returns the predicate of spec without its negation.
func holdsOf(spec predicateSpec) (func(any) bool, error) {
	switch spec.Kind {
	case alwaysKind:
		return func(any) bool { return true }, nil
	case neverKind:
		return func(any) bool { return false }, nil
	case notSortedKind:
		return notSorted, nil
	case hasDuplicateKind:
		return hasDuplicate, nil
	case equalsKind, containsKind:
		return valuePredicate(spec)
	case atLeastKind, divisibleByKind, sumAboveKind, lengthAtLeastKind, indexedAboveKind:
		return numberPredicate(spec)
	}
	return nil, fault.At(fault.New("%q names no predicate", spec.Kind), fault.Field(kindMember))
}

// valuePredicate returns equals or contains, of the value that spec
// states.
func valuePredicate(spec predicateSpec) (func(any) bool, error) {
	if spec.Value == nil {
		return nil, fault.New("%s states no value", spec.Kind)
	}
	wanted, err := literal.Decode(spec.Value)
	if err != nil {
		return nil, fault.At(err, fault.Field(valueMember))
	}
	key := literal.Canonical(wanted)
	if spec.Kind == equalsKind {
		return func(v any) bool { return same(v, key) }, nil
	}
	return func(v any) bool { return contains(v, wanted, key) }, nil
}

// numberPredicate returns at-least, divisible-by, sum-above,
// length-at-least or indexed-above, of the number that spec states.
func numberPredicate(spec predicateSpec) (func(any) bool, error) {
	if spec.N == nil {
		return nil, fault.New("%s states no n", spec.Kind)
	}
	n, err := numberParameter(spec.N)
	if err != nil {
		return nil, fault.At(err, fault.Field(nMember))
	}
	switch spec.Kind {
	case atLeastKind:
		return func(v any) bool { return atLeast(v, n) }, nil
	case sumAboveKind:
		return func(v any) bool { return sumAbove(v, n) }, nil
	case indexedAboveKind:
		return func(v any) bool { return indexedAbove(v, n) }, nil
	}
	if !n.IsInt() {
		return nil, fault.At(fault.New("%v is no integer", n), fault.Field(nMember))
	}
	if spec.Kind == divisibleByKind {
		return func(v any) bool { return divisibleBy(v, n) }, nil
	}
	return func(v any) bool { return lengthAtLeast(v, n) }, nil
}

// numberParameter returns the number of a predicate: an integer, or a
// float when the JSON states a fraction or an exponent.
func numberParameter(raw json.RawMessage) (*big.Float, error) {
	if i, err := literal.Int(raw); err == nil {
		f, _ := numeric(reflect.ValueOf(i))
		return f, nil
	}
	f, err := literal.Float(raw)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(f) {
		return nil, fault.New("NaN is no number of a predicate")
	}
	return new(big.Float).SetFloat64(f), nil
}

// numeric returns v as an exact number, for an integer and a float that is
// not NaN, and reports false for any other value, a bool included.
func numeric(v reflect.Value) (*big.Float, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return new(big.Float).SetInt64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return new(big.Float).SetUint64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) {
			return nil, false
		}
		return new(big.Float).SetFloat64(f), true
	}
	return nil, false
}

// same reports whether v has the canonical text key.
func same(v any, key string) bool {
	return literal.Canonical(v) == key
}

// atLeast reports whether v is a number of n or more.
func atLeast(v any, n *big.Float) bool {
	f, ok := numeric(reflect.ValueOf(v))
	return ok && f.Cmp(n) >= 0
}

// divisibleBy reports whether v is an integer that n divides.
func divisibleBy(v any, n *big.Float) bool {
	rv := reflect.ValueOf(v)
	if !rv.CanInt() && !rv.CanUint() {
		return false
	}
	f, _ := numeric(rv)
	i, _ := f.Int(nil)
	d, _ := n.Int(nil)
	return d.Sign() != 0 && new(big.Int).Rem(i, d).Sign() == 0
}

// sumAbove reports whether v is a list whose numbers sum above n. As in
// the executable reference, a bool counts as an integer, the sum is exact
// while every term is an integer and a float from the first float term on,
// and a NaN sum is above nothing.
func sumAbove(v any, n *big.Float) bool {
	items := reflect.ValueOf(v)
	if items.Kind() != reflect.Slice || literal.IsBytes(items.Type()) {
		return false
	}
	exact, inexact, floating := new(big.Int), 0.0, false
	for k := range items.Len() {
		i, f, isFloat, ok := term(elem(items.Index(k)))
		if !ok {
			continue
		}
		if isFloat && !floating {
			inexact, floating = toFloat(exact), true
		}
		if !floating {
			exact.Add(exact, i)
		} else if isFloat {
			inexact += f
		} else {
			inexact += toFloat(i)
		}
	}
	if !floating {
		return new(big.Float).SetInt(exact).Cmp(n) > 0
	}
	total, ok := numeric(reflect.ValueOf(inexact))
	return ok && total.Cmp(n) > 0
}

// term returns an element of a list as a term of a sum: an integer, which
// a bool is, or a float. It reports false for any other value.
func term(v reflect.Value) (i *big.Int, f float64, isFloat, ok bool) {
	if v.Kind() == reflect.Bool {
		if v.Bool() {
			return big.NewInt(1), 0, false, true
		}
		return new(big.Int), 0, false, true
	}
	if v.CanFloat() {
		return nil, v.Float(), true, true
	}
	exact, ok := numeric(v)
	if !ok {
		return nil, 0, false, false
	}
	i, _ = exact.Int(nil)
	return i, 0, false, true
}

// toFloat returns the float nearest to i, ties to even, as Python converts
// an int it adds to a float.
func toFloat(i *big.Int) float64 {
	f, _ := new(big.Float).SetInt(i).Float64()
	return f
}

// lengthAtLeast reports whether v is a list, a string, a byte string or a
// map of n or more elements, a string counted in characters.
func lengthAtLeast(v any, n *big.Float) bool {
	rv := reflect.ValueOf(v)
	length := -1
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		length = rv.Len()
	case reflect.String:
		length = utf8.RuneCountInString(rv.String())
	}
	return length >= 0 && new(big.Float).SetInt64(int64(length)).Cmp(n) >= 0
}

// contains reports whether v is a list with an element of the canonical
// text key, or a string or a byte string that contains wanted.
func contains(v, wanted any, key string) bool {
	switch v := v.(type) {
	case string:
		w, ok := wanted.(string)
		return ok && strings.Contains(v, w)
	case []byte:
		w, ok := wanted.([]byte)
		return ok && bytes.Contains(v, w)
	}
	items := reflect.ValueOf(v)
	if items.Kind() != reflect.Slice {
		return false
	}
	for i := range items.Len() {
		if same(items.Index(i).Interface(), key) {
			return true
		}
	}
	return false
}

// notSorted reports whether v is a list with an element below the one
// before it, of numbers compared exactly or of strings.
func notSorted(v any) bool {
	items := reflect.ValueOf(v)
	if items.Kind() != reflect.Slice {
		return false
	}
	for i := 1; i < items.Len(); i++ {
		if below(elem(items.Index(i)), elem(items.Index(i-1))) {
			return true
		}
	}
	return false
}

// below reports whether a is below b: two numbers, two strings, or false
// below true.
func below(a, b reflect.Value) bool {
	if a.Kind() == reflect.String && b.Kind() == reflect.String {
		return a.String() < b.String()
	}
	if a.Kind() == reflect.Bool && b.Kind() == reflect.Bool {
		return !a.Bool() && b.Bool()
	}
	x, ok := numeric(a)
	y, ok2 := numeric(b)
	return ok && ok2 && x.Cmp(y) < 0
}

// hasDuplicate reports whether v is a list with two elements of one
// canonical text.
func hasDuplicate(v any) bool {
	items := reflect.ValueOf(v)
	if items.Kind() != reflect.Slice {
		return false
	}
	seen := make(map[string]bool, items.Len())
	for i := range items.Len() {
		text := literal.Canonical(items.Index(i).Interface())
		if seen[text] {
			return true
		}
		seen[text] = true
	}
	return false
}

// indexedAbove reports whether v is a list whose last element is an index,
// from 0, into the elements before it, where the element at the index is a
// number above n. A float or a bool is no index, and a bool is no number.
func indexedAbove(v any, n *big.Float) bool {
	items := reflect.ValueOf(v)
	if items.Kind() != reflect.Slice || literal.IsBytes(items.Type()) || items.Len() == 0 {
		return false
	}
	last := items.Len() - 1
	index := elem(items.Index(last))
	if !index.CanInt() && !index.CanUint() {
		return false
	}
	at, _ := numeric(index)
	if at.Sign() < 0 || at.Cmp(new(big.Float).SetInt64(int64(last))) >= 0 {
		return false
	}
	position, _ := at.Int64()
	element, ok := numeric(elem(items.Index(int(position))))
	return ok && element.Cmp(n) > 0
}

// elem returns the value inside an interface value, and v otherwise.
func elem(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}
