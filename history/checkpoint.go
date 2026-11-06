// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"cmp"
	"slices"
)

// Checkpoint keeps the searches of the last check of a history that passed
// with it, so that the next check of the history continues those searches
// instead of searching from the first call. A check with Whole keeps one
// search of the whole history, and a check without Whole keeps one search
// for each partition. [Resume] passes it to a check. The zero Checkpoint
// does not keep a search.
//
// A check after each step of a case keeps its searches in a Checkpoint, so
// each check searches the calls since the check before it. Over a case of n
// sequential calls, the spec's Next runs once for each call.
//
// # Concurrency
//
// A Checkpoint is safe for one check at a time.
type Checkpoint[S any] struct {
	// history is the history of the kept searches, and nil when the
	// checkpoint does not keep a search.
	history *History
	// whole reports whether the kept check searched every call as one
	// partition.
	whole bool
	// events is the number of events that the kept check read.
	events int
	// parts are the kept partitions. A check with Whole keeps one partition
	// of every call, also of no call.
	parts []*kept[S]
	// every is the partition of every key, which every new call joins: the
	// partition of a check with Whole, and for a check without Whole the one
	// partition of a history with a call without keys. It is nil otherwise.
	every *kept[S]
	// keyOf maps the identity of each key of a check without Whole to the
	// partition of the key.
	keyOf map[string]*kept[S]
	// pending maps the invocation of each call that the kept check read as
	// pending to the partition of the call.
	pending map[int]*kept[S]
	// partitions and searches are the partitions of the last check and the
	// search of each, in the order of their first invocation. Each check
	// reuses their storage.
	partitions []partition
	searches   []*search[S]
}

// kept is a partition of a checkpoint, and the search of its calls.
type kept[S any] struct {
	partition
	// search is the kept search of the partition's calls before its new
	// calls, and nil for a partition that searches from its first call. The
	// calls of the search are the first calls of the partition.
	search *search[S]
}

// resume returns the partitions of h for a check through ops under c and the
// deadline d, in the order of their first invocation, and the search of
// each. The search of a partition is the kept search grown by the
// partition's new calls where it continues as a search of every call of the
// partition does, and a new search otherwise.
func (cp *Checkpoint[S]) resume(h *History, ops operations[S], c config, d *deadline) ([]partition, []*search[S]) {
	if cp.history != h || cp.whole != c.whole {
		cp.restart(h, c.whole)
	}
	events, ids := h.recordedFrom(cp.events)
	calls, completions := callsOf(events, ids, cp.events)
	cp.events += len(events)
	for _, e := range completions {
		cp.complete(e)
	}
	for _, added := range calls {
		cp.join(added)
	}
	// The partitions of a check without Whole have calls, and a check with
	// Whole keeps one partition, which the sort compares with none.
	slices.SortFunc(cp.parts, func(a, b *kept[S]) int {
		return cmp.Compare(a.calls[0].span.Call, b.calls[0].span.Call)
	})
	//dokimi:mutate-skip sbr-delete: the check overwrites every partition that it returns, so the clear only releases the partitions past them
	clear(cp.partitions)
	//dokimi:mutate-skip sbr-delete: the check overwrites every search that it returns, so the clear only releases the searches past them
	clear(cp.searches)
	cp.partitions, cp.searches = cp.partitions[:0], cp.searches[:0]
	for _, p := range cp.parts {
		s := p.search
		if s != nil && s.fits(len(p.calls)-len(s.calls), c) {
			s.grow(ops, p.calls, c, d)
		} else {
			s = newSearch(ops, p.partition, c, d)
		}
		p.search = s
		cp.partitions, cp.searches = append(cp.partitions, p.partition), append(cp.searches, s)
	}
	return cp.partitions, cp.searches
}

// restart clears the searches of cp, so that the check reads h from its
// first event: into one partition of every call for a check that states
// whole, and into the partitions of the keys otherwise.
func (cp *Checkpoint[S]) restart(h *History, whole bool) {
	*cp = Checkpoint[S]{
		history: h, whole: whole, pending: map[int]*kept[S]{}, partitions: cp.partitions, searches: cp.searches,
	}
	if whole {
		cp.every = &kept[S]{keys: []any{}}
		cp.parts = []*kept[S]{cp.every}
		return
	}
	cp.keyOf = map[string]*kept[S]{}
}

// complete records the completion e of a call that the kept check read as
// pending, and makes the partition of the call search from its first call.
// A call that failed leaves its partition. For a check without Whole, the
// other calls of the partition then form the partitions of a check of every
// call.
func (cp *Checkpoint[S]) complete(e Event) {
	p := cp.pending[e.Call]
	//dokimi:mutate-skip sbr-delete: a call completes once, so the entry of a completed call only keeps its partition reachable
	delete(cp.pending, e.Call)
	p.search = nil
	i, _ := slices.BinarySearchFunc(p.calls, e.Call, func(c call, invocation int) int {
		return cmp.Compare(c.span.Call, invocation)
	})
	if e.Kind != Fail {
		span := &p.calls[i].span
		span.Completion = e.Index
		if e.Kind == OK {
			span.Operation.Known, span.Operation.Output = true, e.Output
		}
		return
	}
	failed := p.calls[i]
	p.calls = slices.Delete(p.calls, i, i+1)
	if cp.whole {
		return
	}
	for _, id := range failed.ids {
		delete(cp.keyOf, id)
	}
	cp.replace([]*kept[S]{p})
}

// join adds the new call c to the partitions that it joins: the partition of
// every key when there is one, every partition for a call without keys, and
// the partitions of its keys otherwise. A call that joins no partition starts
// one, and a call that joins two or more partitions joins them into one,
// which searches from its first call.
func (cp *Checkpoint[S]) join(c call) {
	switch {
	case cp.every != nil:
		cp.extend(cp.every, c)
	case len(c.ids) > 0:
		cp.joinKeys(c)
	case len(cp.parts) == 1:
		cp.extend(cp.parts[0], c)
	default:
		cp.replace(slices.Clone(cp.parts), c)
	}
}

// joinKeys adds the new call c, which has keys, to the partition of its
// keys, to a new partition when its keys belong to none, and to one partition
// of the partitions of its keys when they belong to two or more.
func (cp *Checkpoint[S]) joinKeys(c call) {
	var p *kept[S]
	for _, id := range c.ids {
		switch q := cp.keyOf[id]; {
		case q == nil, q == p:
		case p == nil:
			p = q
		default:
			var joined []*kept[S]
			for _, id := range c.ids {
				if r := cp.keyOf[id]; r != nil && !slices.Contains(joined, r) {
					joined = append(joined, r)
				}
			}
			cp.replace(joined, c)
			return
		}
	}
	if p == nil {
		p = &kept[S]{}
		cp.parts = append(cp.parts, p)
	}
	cp.extend(p, c)
}

// extend adds the new call c to the partition p, and maps the keys of c and
// a pending c to p.
func (cp *Checkpoint[S]) extend(p *kept[S], c call) {
	p.calls = append(p.calls, c)
	if c.pending() {
		cp.pending[c.span.Call] = p
	}
	if cp.whole {
		return
	}
	if len(c.ids) == 0 {
		p.keys, cp.every = []any{}, p
	}
	for i, id := range c.ids {
		if cp.keyOf[id] == p {
			continue
		}
		cp.keyOf[id] = p
		if p != cp.every {
			p.keys = append(p.keys, c.keys[i])
		}
	}
}

// replace replaces the partitions old with the partitions of their calls and
// of more, which follow every call of old. Each of them searches from its
// first call. The keys and the pending calls of old map to the partitions
// that replace them.
func (cp *Checkpoint[S]) replace(old []*kept[S], more ...call) {
	var calls []call
	for _, p := range old {
		calls = append(calls, p.calls...)
	}
	// The partition of every key is the one partition of the check, so it is
	// among old when there is one.
	cp.every = nil
	cp.parts = slices.DeleteFunc(cp.parts, func(p *kept[S]) bool { return slices.Contains(old, p) })
	slices.SortFunc(calls, func(a, b call) int { return cmp.Compare(a.span.Call, b.span.Call) })
	for _, q := range partitionsOf(append(calls, more...)) {
		p := &kept[S]{partition: q}
		cp.parts = append(cp.parts, p)
		if len(p.keys) == 0 {
			cp.every = p
		}
		for _, c := range p.calls {
			for _, id := range c.ids {
				cp.keyOf[id] = p
			}
			if c.pending() {
				cp.pending[c.span.Call] = p
			}
		}
	}
}
