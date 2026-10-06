// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"fmt"
	"hash/fnv"
	"slices"
	"strings"

	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/literal"
)

// emptyValue is the value of a key that the key-value spec does not store.
const emptyValue = ""

// Specs builds each spec of the definition's specs section, by its name. A
// state is a decoded value: the register's value, the key-value map as
// [literal.Pairs] in the order its keys were stored, and the queue and the
// set as a []any in order. Two values compare by their canonical texts.
//
// Every spec accepts a call whose outcome is unknown in every state, checks
// no output of a write, a put, an append, an enqueue or an add, and panics on
// an operation that it does not define, which the check reports as a fault.
// The key-value map and the set compare their states in any order, and hash
// them alike.
var Specs = map[string]history.Spec[any]{
	"register":     {Initial: func() any { return nil }, Next: register},
	"cas-register": {Initial: func() any { return nil }, Next: casRegister},
	"key-value": {
		Initial: func() any { return literal.Pairs{} }, Next: keyValue, Equal: sameCanonical, Hash: hashCanonical,
	},
	"queue":          {Initial: func() any { return []any{} }, Next: queue},
	"set":            {Initial: func() any { return []any{} }, Next: set, Equal: sameSet, Hash: hashSet},
	"lossy-register": {Initial: func() any { return nil }, Next: lossyRegister},
}

// undefined returns the panic value of op, an operation that the spec of
// the name does not define.
func undefined(name string, op history.Operation) string {
	return fmt.Sprintf("conformance: the %s spec has no operation %q", name, op.Name)
}

// sameCanonical reports whether a and b, two decoded values, have one
// canonical text. Two states of the key-value map that store the same values
// under the same keys have one text, whatever the order of their keys.
func sameCanonical(a, b any) bool {
	return literal.Canonical(a) == literal.Canonical(b)
}

// readRegister steps a register's read, which outputs the state and changes
// nothing. It panics on any other operation of the spec of the name.
func readRegister(state any, op history.Operation, name string) []any {
	if op.Name != "read" {
		panic(undefined(name, op))
	}
	if !op.Known || sameCanonical(op.Output, state) {
		return []any{state}
	}
	return nil
}

// register steps the register: a write stores its value, and a read outputs
// it.
func register(state any, op history.Operation) []any {
	if op.Name == "write" {
		return []any{op.Args[0]}
	}
	return readRegister(state, op, "register")
}

// casRegister steps the register with compare-and-set. cas(from, to) outputs
// whether the state equals from, and stores to when it does. A cas whose
// outcome is unknown takes effect when the state equals from, and leaves the
// state otherwise.
func casRegister(state any, op history.Operation) []any {
	switch op.Name {
	case "write":
		return register(state, op)
	case "cas":
		matches := sameCanonical(state, op.Args[0])
		if op.Known && op.Output != matches {
			return nil
		}
		if matches {
			return []any{op.Args[1]}
		}
		return []any{state}
	}
	return readRegister(state, op, "cas-register")
}

// lossyRegister steps the register whose write may be lost: a write leaves
// the new value, then the old one.
func lossyRegister(state any, op history.Operation) []any {
	if op.Name == "write" {
		return []any{op.Args[0], state}
	}
	return readRegister(state, op, "lossy-register")
}

// keyValue steps the map from keys to strings, each of which starts empty.
// get(k) outputs the value under k, put(k, v) stores v under k, and
// append(k, s) stores the value under k followed by s.
func keyValue(state any, op history.Operation) []any {
	pairs := state.(literal.Pairs)
	name := op.Args[0]
	held := lookup(pairs, name)
	switch op.Name {
	case "get":
		if !op.Known || sameCanonical(op.Output, held) {
			return []any{state}
		}
		return nil
	case "put":
		return []any{withValue(pairs, name, op.Args[1])}
	case "append":
		return []any{withValue(pairs, name, held.(string)+op.Args[1].(string))}
	}
	panic(undefined("key-value", op))
}

// lookup returns the value that pairs stores under name, or the empty value.
func lookup(pairs literal.Pairs, name any) any {
	for _, e := range pairs.Entries {
		if sameCanonical(e.Key, name) {
			return e.Value
		}
	}
	return emptyValue
}

// withValue returns pairs with value under name: in the place of name when
// pairs stores it, and last otherwise. The map stores no empty value, so two
// maps are equal exactly when every key reads the same value from both.
func withValue(pairs literal.Pairs, name, value any) literal.Pairs {
	entries := slices.Clone(pairs.Entries)
	i := slices.IndexFunc(entries, func(e literal.Entry) bool { return sameCanonical(e.Key, name) })
	if i < 0 {
		if value != emptyValue {
			entries = append(entries, literal.Entry{Key: name, Value: value})
		}
		return literal.Pairs{Entries: entries}
	}
	if value == emptyValue {
		return literal.Pairs{Entries: slices.Delete(entries, i, i+1)}
	}
	entries[i].Value = value
	return literal.Pairs{Entries: entries}
}

// queue steps the queue: enqueue(v) adds v at the tail, and dequeue() removes
// the head and outputs it, or outputs null when the queue is empty.
func queue(state any, op history.Operation) []any {
	items := state.([]any)
	switch op.Name {
	case "enqueue":
		return []any{append(slices.Clip(items), op.Args[0])}
	case "dequeue":
		if len(items) == 0 {
			if !op.Known || op.Output == nil {
				return []any{state}
			}
			return nil
		}
		if !op.Known || sameCanonical(op.Output, items[0]) {
			return []any{items[1:]}
		}
		return nil
	}
	panic(undefined("queue", op))
}

// set steps the set, which lists its values in the order they were added.
// add(v) adds v when it is absent, and remove(v) and contains(v) output
// whether v is present. A remove whose outcome is unknown removes v when it
// is present.
func set(state any, op history.Operation) []any {
	items := state.([]any)
	value := op.Args[0]
	present := slices.ContainsFunc(items, func(held any) bool { return sameCanonical(value, held) })
	switch op.Name {
	case "add":
		if present {
			return []any{state}
		}
		return []any{append(slices.Clip(items), value)}
	case "contains":
		if !op.Known || op.Output == present {
			return []any{state}
		}
		return nil
	case "remove":
		if op.Known && op.Output != present {
			return nil
		}
		return []any{slices.DeleteFunc(slices.Clone(items), func(held any) bool { return sameCanonical(held, value) })}
	}
	panic(undefined("set", op))
}

// hashCanonical returns the hash of the canonical text of a state, which
// every state that [sameCanonical] reports equal to it shares.
func hashCanonical(state any) uint64 {
	return hashText(literal.Canonical(state))
}

// sameSet reports whether two states of the set contain equal values, in any
// order.
func sameSet(a, b any) bool {
	return setText(a) == setText(b)
}

// hashSet returns the hash of a state of the set, which every state with
// equal values shares.
func hashSet(state any) uint64 {
	return hashText(setText(state))
}

// setText returns the canonical texts of the values of a state of the set,
// sorted, so that two states of equal values in any order have one text.
func setText(state any) string {
	items := state.([]any)
	texts := make([]string, len(items))
	for i, item := range items {
		texts[i] = literal.Canonical(item)
	}
	slices.Sort(texts)
	return strings.Join(texts, ",")
}

// hashText returns the 64-bit FNV-1a hash of text.
func hashText(text string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	return h.Sum64()
}
