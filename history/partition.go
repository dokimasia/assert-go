// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"cmp"
	"slices"
)

// call is a call of a history as the checker reads it: its span, and its
// keys with the identity of each.
type call struct {
	// span is the call as the record of a check states it.
	span Span
	// keys are the keys that the call touches, and none for every key.
	keys []any
	// ids are the identities of keys, the texts of their typed literals.
	ids []string
}

// partition is a set of calls that share keys, which the checker searches
// on its own.
type partition struct {
	// keys are the keys of the calls, each once, in the order the history
	// first declares them, and none for a partition whose call touches every
	// key.
	keys []any
	// calls are the partition's calls, in event order.
	calls []call
}

// callsOf returns the calls that events invoke, in event order, without the
// calls that failed. events are the events of a history from the index from
// on, and ids are the identities of each event's keys. A call is known when
// it completed as OK, and its completion is -1 when it is pending. It
// reports false when one of events completes a call invoked before from.
func callsOf(events []Event, ids [][]string, from int) ([]call, bool) {
	completion := make([]int, len(events))
	for i, e := range events {
		completion[i] = -1
		if e.Kind != Invoke {
			if e.Call < from {
				return nil, false
			}
			completion[e.Call-from] = i
		}
	}
	var calls []call
	for i, e := range events {
		end := completion[i]
		if e.Kind != Invoke || (end >= 0 && events[end].Kind == Fail) {
			continue
		}
		op := Operation{Name: e.Operation, Args: e.Args}
		span := Span{Call: from + i, Completion: -1, Process: e.Process, Operation: op}
		if end >= 0 {
			span.Completion = from + end
			if events[end].Kind == OK {
				span.Operation.Known, span.Operation.Output = true, events[end].Output
			}
		}
		calls = append(calls, call{span: span, keys: e.Keys, ids: ids[i]})
	}
	return calls, true
}

// partitionsOf returns the partitions of calls, in the order of their first
// invocation. Two calls that share a key are in one partition, and a call
// that touches every key puts every call into one partition.
func partitionsOf(calls []call) []partition {
	if len(calls) == 0 {
		return nil
	}
	for _, c := range calls {
		if len(c.ids) == 0 {
			return []partition{{keys: []any{}, calls: calls}}
		}
	}
	parent := make([]int, len(calls))
	first := map[string]int{}
	for position, c := range calls {
		parent[position] = position
		for _, id := range c.ids {
			other, seen := first[id]
			if !seen {
				first[id], other = position, position
			}
			parent[root(parent, position)] = root(parent, other)
		}
	}
	var parts []partition
	index := map[int]int{}
	for position, c := range calls {
		r := root(parent, position)
		i, seen := index[r]
		if !seen {
			i = len(parts)
			index[r] = i
			parts = append(parts, partition{})
		}
		parts[i].calls = append(parts[i].calls, c)
	}
	for i := range parts {
		parts[i].keys = keysOf(parts[i].calls)
	}
	return parts
}

// root returns the representative of the partition of the call at position,
// and halves the path to it.
func root(parent []int, position int) int {
	for parent[position] != position {
		parent[position] = parent[parent[position]]
		position = parent[position]
	}
	return position
}

// keysOf returns the keys that calls touch, each once, in the order that
// calls first declare them.
func keysOf(calls []call) []any {
	var keys []any
	seen := map[string]bool{}
	for _, c := range calls {
		for i, id := range c.ids {
			if !seen[id] {
				seen[id] = true
				keys = append(keys, c.keys[i])
			}
		}
	}
	return keys
}

// concurrency returns the most calls that are open at one event. A call is
// open from its invocation to its completion, and a call that is not known
// is open to the end of the history.
func concurrency(calls []call) int {
	type change struct{ event, open int }
	var changes []change
	for _, c := range calls {
		changes = append(changes, change{event: c.span.Call, open: 1})
		if c.span.Operation.Known {
			changes = append(changes, change{event: c.span.Completion, open: -1})
		}
	}
	slices.SortFunc(changes, func(a, b change) int { return cmp.Compare(a.event, b.event) })
	open, most := 0, 0
	for _, c := range changes {
		open += c.open
		most = max(most, open)
	}
	return most
}
