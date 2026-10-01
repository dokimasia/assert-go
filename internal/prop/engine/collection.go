// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import "go.dokimi.dev/assert/internal/prop/choice"

// The ids and the rules of the collections.
const (
	// listID is the id of a list.
	listID = "list"
	// dictID is the id of a dict.
	dictID = "dict"
	// elementLabel labels the span of one element of a collection.
	elementLabel = "element"
	// entryLabel labels the span of one entry of a dict.
	entryLabel = "entry"
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
func List[T any](of Generator[T], sizes choice.Sizes) Generator[[]T] {
	return newGenerator(listID, func(c *Case) []T {
		span := c.openSpan(listID)
		defer c.closeSpan(span)
		return collect(c, sizes, elementLabel, of.decode, nil)
	})
}

// UniqueList returns a generator of lists as [List] does, which discards an
// element equal to an earlier one. Two values are equal when they have the
// same type and the same value, with floats compared by their bits.
func UniqueList[T any](of Generator[T], sizes choice.Sizes) Generator[[]T] {
	return newGenerator(listID, func(c *Case) []T {
		span := c.openSpan(listID)
		defer c.closeSpan(span)
		return collect(c, sizes, elementLabel, of.decode, func(v T) string { return canonicalKey(v) })
	})
}

// Dict returns a generator of maps with entries of a key of keys and a
// value of values, with lengths that sizes admits, decoded as a unique
// list of entries compared by key. A map with float keys stores what Go's
// equality allows: one entry for the two zeros, and one for each NaN.
func Dict[K comparable, V any](keys Generator[K], values Generator[V], sizes choice.Sizes) Generator[map[K]V] {
	decode := func(c *Case) entry[K, V] {
		key := keys.decode(c)
		return entry[K, V]{key: key, value: values.decode(c)}
	}
	unique := func(e entry[K, V]) string { return canonicalKey(e.key) }
	return newGenerator(dictID, func(c *Case) map[K]V {
		span := c.openSpan(dictID)
		defer c.closeSpan(span)
		entries := collect(c, sizes, entryLabel, decode, unique)
		m := make(map[K]V, len(entries))
		for _, e := range entries {
			m[e.key] = e.value
		}
		return m
	})
}

// collect decodes a collection: per element a continue flag, then the
// element in a span labelled label that starts at its flag.
//
// With a key function, an element whose key repeats an earlier key is
// discarded, and the next flag is decided for the same count. After ten
// discards in a row the collection stops, and rejects the case when it is
// still below its minimum length.
func collect[T any](c *Case, sizes choice.Sizes, label string, decode func(*Case) T, key func(T) string) []T {
	var items []T
	seen := make(map[string]struct{})
	discards := 0
	for {
		start := c.position()
		if !c.more(sizes, len(items)) {
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
