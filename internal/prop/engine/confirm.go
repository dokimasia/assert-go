// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

// confirm replays a failing case once from its choices, and returns how
// the replay differed, or nil when it did not. The comparison takes the
// requests' bounds first, then the observed fingerprints, then the way the
// replay ended.
func confirm(body Body, failing Execution, s Settings) *Divergence {
	replay := execute(body, replaying{choices: failing.Case.Choices()}, s.MaxChoices, s.Clock)
	recorded, replayed := nodesOf(failing.Case), nodesOf(replay.Case)
	for index := range max(len(recorded), len(replayed)) {
		before, after := requestAt(recorded, index), requestAt(replayed, index)
		if before != after {
			return &Divergence{What: RequestDifference, Index: index, Recorded: before, Replayed: after}
		}
	}
	prints, again := failing.Case.Fingerprints(), replay.Case.Fingerprints()
	for index := range max(len(prints), len(again)) {
		before, after := fingerprintAt(prints, index), fingerprintAt(again, index)
		if before != after {
			return &Divergence{What: FingerprintDifference, Index: index, Recorded: before, Replayed: after}
		}
	}
	if replay.Status != CaseFailed || replay.Identity != failing.Identity {
		d := &Divergence{What: VerdictDifference, Index: len(recorded), Recorded: failing.Identity}
		if replay.Status == CaseFailed {
			d.Replayed = replay.Identity
		}
		return d
	}
	return nil
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
