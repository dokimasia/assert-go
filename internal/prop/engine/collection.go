// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"reflect"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
)

// The labels of the spans of a collection's elements.
const (
	// ElementLabel labels the span of one element of a collection.
	ElementLabel = "element"
	// EntryLabel labels the span of one entry of a dict.
	EntryLabel = "entry"
)

// The ids and the rules of the collections.
const (
	// listID is the id of a list.
	listID = "list"
	// dictID is the id of a dict.
	dictID = "dict"
	// maxDiscards is the number of duplicates in a row after which a
	// collection stops. A collection still below its minimum length then
	// rejects the case.
	maxDiscards = 10
)

// entry is one key and its value of a dict.
type entry[K comparable, V any] struct {
	// key is the entry's key.
	key K
	// value is the entry's value.
	value V
}

// List returns a generator of lists of of's values with lengths that sizes
// admits: per element a continue flag that decides structure, then the
// element. Its simplest value is the minimum number of simplest elements.
// It runs backwards from a slice or an array, through each element in
// order.
func List[T any](of Generator[T], sizes choice.Sizes) Generator[[]T] {
	decode := func(c *Case) []T {
		span := c.openSpan(listID)
		defer c.closeSpan(span)
		return collect(c, sizes, ElementLabel, of.decode, nil, nil)
	}
	return NewInvertible(listID, decode, func(v any) ([]Step, []T, error) {
		return invertList(of, sizes, v, false)
	})
}

// UniqueList returns a generator of lists as [List] does, which discards an
// element equal to an earlier one. Two values are equal when they have the
// same type and the same value, with floats compared by their bits. It runs
// backwards from a list of distinct elements, through each in order.
func UniqueList[T any](of Generator[T], sizes choice.Sizes) Generator[[]T] {
	decode := func(c *Case) []T {
		span := c.openSpan(listID)
		defer c.closeSpan(span)
		return collect(c, sizes, ElementLabel, of.decode, func(v T) string { return canonicalKey(v) }, nil)
	}
	return NewInvertible(listID, decode, func(v any) ([]Step, []T, error) {
		return invertList(of, sizes, v, true)
	})
}

// invertList returns the steps of a list of of's values under sizes that
// decode to v, a slice or an array, and the list they decode to. A unique
// list refuses two elements that decode to equal values. The fault of an
// element is at the element's index.
func invertList[T any](of Generator[T], sizes choice.Sizes, v any, unique bool) ([]Step, []T, error) {
	items, ok := ListItems(v)
	if !ok {
		return nil, nil, uninvertible("%v is no list", v)
	}
	parts := make([][]Step, len(items))
	list := make([]T, len(items))
	seen := make(map[string]int, len(items))
	for i, item := range items {
		steps, t, err := of.inverse(item)
		if err != nil {
			return nil, nil, fault.At(err, fault.Index(i))
		}
		parts[i], list[i] = steps, t
		if !unique {
			continue
		}
		key := canonicalKey(list[i])
		if earlier, repeated := seen[key]; repeated {
			return nil, nil, fault.At(uninvertible("the element repeats element %d", earlier), fault.Index(i))
		}
		seen[key] = i
	}
	steps, err := CollectionSteps(sizes, parts)
	return steps, list, err
}

// Dict returns a generator of maps with entries of a key of keys and a
// value of values, with lengths that sizes admits, decoded as a unique
// list of entries compared by key. A map with float keys stores what Go's
// equality allows: one entry for the two zeros, and one for each NaN.
//
// It runs backwards from a map, through each entry's key and then its
// value. The entries take the shortlex order of their own choices, so a
// map, whose order of iteration varies, has one sequence of choices.
func Dict[K comparable, V any](keys Generator[K], values Generator[V], sizes choice.Sizes) Generator[map[K]V] {
	decodeEntry := func(c *Case) entry[K, V] {
		key := keys.decode(c)
		return entry[K, V]{key: key, value: values.decode(c)}
	}
	unique := func(e entry[K, V]) string { return canonicalKey(e.key) }
	decode := func(c *Case) map[K]V {
		span := c.openSpan(dictID)
		defer c.closeSpan(span)
		entries := collect(c, sizes, EntryLabel, decodeEntry, unique, nil)
		m := make(map[K]V, len(entries))
		for _, e := range entries {
			m[e.key] = e.value
		}
		return m
	}
	return NewInvertible(dictID, decode, func(v any) ([]Step, map[K]V, error) {
		return invertDict(keys, values, sizes, v)
	})
}

// invertDict returns the steps of a map of keys and values under sizes
// that decode to v, a map, and the map they decode to. It refuses two keys
// that decode to equal keys. The fault of a value is at its key, and so is
// the fault of a key that keys does not produce, whose cause is the fault of
// keys.
func invertDict[K comparable, V any](keys Generator[K], values Generator[V], sizes choice.Sizes, v any) (
	[]Step, map[K]V, error,
) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, nil, uninvertible("%v is no map", v)
	}
	parts := make([][]Step, 0, rv.Len())
	m := make(map[K]V, rv.Len())
	seen := make(map[string]struct{}, rv.Len())
	for key, value := range rv.Seq2() {
		stated := key.Interface()
		keySteps, k, err := keys.inverse(stated)
		if err != nil {
			return nil, nil, fault.At(uninvertible("the generator of keys produces no such key").Because(err),
				fault.Key(stated))
		}
		valueSteps, val, err := values.inverse(value.Interface())
		if err != nil {
			return nil, nil, fault.At(err, fault.Key(stated))
		}
		text := canonicalKey(k)
		if _, repeated := seen[text]; repeated {
			return nil, nil, uninvertible("two keys decode to the key %v", k)
		}
		seen[text] = struct{}{}
		parts = append(parts, slices.Concat(keySteps, valueSteps))
		m[k] = val
	}
	slices.SortFunc(parts, CompareSteps)
	steps, err := CollectionSteps(sizes, parts)
	return steps, m, err
}

// Collect decodes a collection whose elements the caller keeps itself: per
// element a continue flag that decides structure, then a call of element
// in a span labelled "element" that starts at its flag. The number of
// elements is one that sizes admits.
func Collect(c *Case, sizes choice.Sizes, element func()) {
	collect(c, sizes, ElementLabel, func(*Case) struct{} {
		element()
		return struct{}{}
	}, nil, nil)
}

// Elements decodes a collection of the values that decode returns, as List
// and Dict do: per element a continue flag that decides structure, then a
// call of decode in a span labelled label that starts at its flag. The
// number of elements is one that sizes admits.
//
// key, when not nil, returns the key that an element is unique by. An
// element whose key repeats an earlier one is discarded, and the next flag
// is decided for the same count. After ten discards in a row the
// collection stops, and rejects the case when it is still below its
// minimum length.
//
// stop, when not nil, is asked before each flag that the minimum length
// does not force. When it reports true, the flag admits only 0, so the
// collection takes no further element.
func Elements[T any](c *Case, sizes choice.Sizes, label string, decode func(*Case) T, key func(T) string,
	stop func() bool,
) []T {
	return collect(c, sizes, label, decode, key, stop)
}

// collect decodes a collection as [Elements] states.
func collect[T any](c *Case, sizes choice.Sizes, label string, decode func(*Case) T, key func(T) string,
	stop func() bool,
) []T {
	var items []T
	seen := make(map[string]struct{})
	discards := 0
	for {
		start := c.Position()
		decided := sizes
		if stop != nil && len(items) >= sizes.Min() && stop() {
			decided = stoppedAt(sizes, len(items))
		}
		if !c.more(decided, len(items), random.Average(decided)) {
			return items
		}
		span := c.openSpanAt(label, start)
		item := decode(c)
		c.closeSpan(span)
		if key == nil {
			items = append(items, item)
			continue
		}
		k := key(item)
		if _, repeated := seen[k]; !repeated {
			seen[k], discards = struct{}{}, 0
			items = append(items, item)
			continue
		}
		discards++
		if discards == maxDiscards {
			c.Assume(len(items) >= sizes.Min())
			return items
		}
	}
}

// stoppedAt returns the sizes of a collection of count elements that takes
// no further element: sizes with count as their maximum. count is at least
// the minimum of sizes, so the sizes it returns admit a length.
func stoppedAt(sizes choice.Sizes, count int) choice.Sizes {
	stopped, _ := choice.NewSizes(sizes.Min(), count)
	return stopped
}
