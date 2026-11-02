// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import "go.dokimi.dev/assert/internal/record"

// confirm replays a case once from its choices, a failing case or a valid
// case, and returns the replay and how it differed, or nil when it did not.
// The comparison takes the requests' bounds first, then the observed
// fingerprints, then the way the replay ended. A replay of a valid case
// differs in its verdict only when it fails, so a replay that is rejected or
// skips agrees with it. A request or a fingerprint that differs takes where
// the replay made it. A replay that ends with a fault is not compared, and
// the caller reads its fault. The slot of s takes the calls of every run of
// the replay under phase.
func confirm(body Body, failing Execution, s Settings, phase record.Phase) (Execution, *Divergence) {
	choices := failing.Case.Choices()
	c := newCase(replaying{choices: choices}, s, nil)
	c.keepsWheres, c.wheres = true, make([]Where, 0, len(choices))
	replay := finish(c, body)
	replay.take(s.Slot, phase)
	if replay.Status == CaseFaulted {
		return replay, nil
	}
	recorded, replayed := nodesOf(failing.Case), nodesOf(replay.Case)
	for index := range max(len(recorded), len(replayed)) {
		before, after := requestAt(recorded, index), requestAt(replayed, index)
		if before != after {
			return replay, &Divergence{
				What: RequestDifference, Index: index, Recorded: before, Replayed: after,
				Where: replay.Case.requestWhere(index),
			}
		}
	}
	prints, again := failing.Case.Fingerprints(), replay.Case.Fingerprints()
	for index := range max(len(prints), len(again)) {
		before, after := fingerprintAt(prints, index), fingerprintAt(again, index)
		if before != after {
			return replay, &Divergence{
				What: FingerprintDifference, Index: index, Recorded: before, Replayed: after,
				Where: replay.Case.fingerprintWhere(index),
			}
		}
	}
	failed := failing.Status == CaseFailed
	if (replay.Status == CaseFailed) != failed || replay.Identity != failing.Identity {
		d := &Divergence{What: VerdictDifference, Index: len(recorded)}
		if failed {
			d.Recorded = failing.Identity
		}
		if replay.Status == CaseFailed {
			d.Replayed = replay.Identity
		}
		return replay, d
	}
	return replay, nil
}

// requestAt returns the bounds of the request at index, and nil past the
// last request.
func requestAt(nodes []node, index int) any {
	if index >= len(nodes) {
		return nil
	}
	return nodes[index].r.bounds
}

// fingerprintAt returns the fingerprint at index, and nil past the last
// fingerprint.
func fingerprintAt(prints []uint64, index int) any {
	if index >= len(prints) {
		return nil
	}
	return prints[index]
}
