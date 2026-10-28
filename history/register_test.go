// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
)

// registerCase is a register history, and the anomaly and the kinds that a
// check at each level reports, none for a check that passes.
type registerCase struct {
	name                               string
	give                               func(h *history.History)
	wantSerializable, wantSI           history.Anomaly
	wantSerializableKinds, wantSIKinds []history.Anomaly
}

// registerCases are the register histories of the specs: each anomaly that a
// register history can exhibit, and the histories that pass.
var registerCases = []registerCase{
	{
		name: "passes a serial history of read-modify-writes",
		give: func(h *history.History) {
			transact(h, 0, history.OK, loadOf("x", nil), writeOf("x", 1))
			transact(h, 0, history.OK, loadOf("x", 1), writeOf("x", 2))
			transact(h, 0, history.OK, loadOf("x", 2))
		},
	},
	{
		name: "reports a read of a value that no transaction wrote",
		give: func(h *history.History) {
			transact(h, 0, history.OK, writeOf("x", 1))
			transact(h, 1, history.OK, loadOf("x", 9))
		},
		wantSerializable: history.GarbageRead, wantSerializableKinds: []history.Anomaly{history.GarbageRead},
		wantSI: history.GarbageRead, wantSIKinds: []history.Anomaly{history.GarbageRead},
	},
	{
		name: "reports a read that differs from the transaction's own earlier write",
		give: func(h *history.History) {
			transact(h, 0, history.OK, writeOf("x", 2))
			transact(h, 1, history.OK, writeOf("x", 1), loadOf("x", 2))
		},
		wantSerializable:      history.InternalInconsistency,
		wantSerializableKinds: []history.Anomaly{history.InternalInconsistency},
		wantSI:                history.InternalInconsistency,
		wantSIKinds:           []history.Anomaly{history.InternalInconsistency},
	},
	{
		name:                  "reports a read of the transaction's own later write",
		give:                  func(h *history.History) { transact(h, 0, history.OK, loadOf("x", 1), writeOf("x", 1)) },
		wantSerializable:      history.InternalInconsistency,
		wantSerializableKinds: []history.Anomaly{history.InternalInconsistency},
		wantSI:                history.InternalInconsistency,
		wantSIKinds:           []history.Anomaly{history.InternalInconsistency},
	},
	{
		name: "reports a read of a write of an aborted transaction",
		give: func(h *history.History) {
			transact(h, 0, history.Fail, writeOf("x", 1))
			transact(h, 1, history.OK, loadOf("x", 1))
		},
		wantSerializable: history.AbortedRead, wantSerializableKinds: []history.Anomaly{history.AbortedRead},
		wantSI: history.AbortedRead, wantSIKinds: []history.Anomaly{history.AbortedRead},
	},
	{
		name: "reports a read of a value that its writer overwrote",
		give: func(h *history.History) {
			transact(h, 0, history.OK, writeOf("x", 1), writeOf("x", 2))
			transact(h, 1, history.OK, loadOf("x", 1))
		},
		wantSerializable: history.IntermediateRead, wantSerializableKinds: []history.Anomaly{history.IntermediateRead},
		wantSI: history.IntermediateRead, wantSIKinds: []history.Anomaly{history.IntermediateRead},
	},
	{
		name: "passes a read of the transaction's own overwritten write",
		give: func(h *history.History) {
			transact(h, 0, history.OK, writeOf("x", 1), loadOf("x", 1), writeOf("x", 2))
		},
	},
	{
		name: "reports two transactions that overwrite each other's reads as G0",
		give: func(h *history.History) {
			first := []any{writeOf("y", 3), loadOf("x", 1), writeOf("x", 2)}
			second := []any{writeOf("x", 1), loadOf("y", 3), writeOf("y", 4)}
			a := h.Invoke(0, "txn", invokedOf(first), "x", "y")
			b := h.Invoke(1, "txn", invokedOf(second), "x", "y")
			a.OK(first)
			b.OK(second)
		},
		wantSerializable: history.G0, wantSerializableKinds: []history.Anomaly{history.G0, history.G1c},
		wantSI: history.G0, wantSIKinds: []history.Anomaly{history.G0, history.G1c},
	},
	{
		name: "reports a read skew as G-single",
		give: func(h *history.History) {
			transact(h, 0, history.OK, writeOf("x", 1), writeOf("y", 1))
			reads := []any{loadOf("x", 1), loadOf("y", 2)}
			writes := []any{loadOf("x", 1), writeOf("x", 2), writeOf("y", 2)}
			reader := h.Invoke(1, "txn", invokedOf(reads), "x", "y")
			writer := h.Invoke(2, "txn", invokedOf(writes), "x", "y")
			writer.OK(writes)
			reader.OK(reads)
		},
		wantSerializable: history.GSingle, wantSerializableKinds: []history.Anomaly{history.GSingle, history.G2},
		wantSI: history.GSingle, wantSIKinds: []history.Anomaly{history.GSingle},
	},
	{
		name: "reports a long fork as G-nonadjacent",
		give: func(h *history.History) {
			transact(h, 0, history.OK, writeOf("x", 1))
			transact(h, 1, history.OK, writeOf("y", 1))
			transact(h, 2, history.OK, loadOf("x", 1), loadOf("y", nil))
			transact(h, 3, history.OK, loadOf("y", 1), loadOf("x", nil))
		},
		wantSerializable:      history.GNonadjacent,
		wantSerializableKinds: []history.Anomaly{history.GNonadjacent, history.G2},
		wantSI:                history.GNonadjacent, wantSIKinds: []history.Anomaly{history.GNonadjacent},
	},
	{
		name: "fails a write skew only as serializable",
		give: func(h *history.History) {
			first := []any{loadOf("x", nil), loadOf("y", nil), writeOf("x", 1)}
			second := []any{loadOf("x", nil), loadOf("y", nil), writeOf("y", 2)}
			a := h.Invoke(0, "txn", invokedOf(first), "x", "y")
			b := h.Invoke(1, "txn", invokedOf(second), "x", "y")
			a.OK(first)
			b.OK(second)
		},
		wantSerializable: history.G2, wantSerializableKinds: []history.Anomaly{history.G2},
	},
	{
		name: "fails a lost update only as serializable",
		give: func(h *history.History) {
			transact(h, 0, history.OK, writeOf("x", 1))
			first := []any{loadOf("x", 1), writeOf("x", 2)}
			second := []any{loadOf("x", 1), writeOf("x", 3)}
			a := h.Invoke(1, "txn", invokedOf(first), "x")
			b := h.Invoke(2, "txn", invokedOf(second), "x")
			a.OK(first)
			b.OK(second)
		},
		wantSerializable: history.G2, wantSerializableKinds: []history.Anomaly{history.G2},
	},
	{
		name: "takes an observed write of unknown outcome as committed",
		give: func(h *history.History) {
			transact(h, 0, history.Unknown, writeOf("x", 1), writeOf("y", 1))
			transact(h, 1, history.OK, loadOf("x", 1), loadOf("y", nil))
		},
		wantSerializable: history.GSingle, wantSerializableKinds: []history.Anomaly{history.GSingle, history.G2},
		wantSI: history.GSingle, wantSIKinds: []history.Anomaly{history.GSingle},
	},
	{
		name: "leaves out an unobserved write of unknown outcome",
		give: func(h *history.History) {
			transact(h, 0, history.Unknown, writeOf("x", 1), writeOf("y", 1))
			transact(h, 1, history.OK, loadOf("x", nil), loadOf("y", nil))
		},
	},
}

// invokedOf returns the arguments of the invocation of a transaction whose
// micro-operations ran as operations: each read with nil.
func invokedOf(operations []any) []any {
	args := make([]any, len(operations))
	for i, m := range operations {
		parts := m.([]any)
		args[i] = parts
		if parts[0] == "read" {
			args[i] = []any{"read", parts[1], nil}
		}
	}
	return args
}

// TestRegister checks the verdicts of the two isolation checks over register
// histories.
func TestRegister(t *testing.T) {
	t.Parallel()

	levels := []struct {
		name  string
		check func(assert.TB, *history.History, string)
		want  func(registerCase) (history.Anomaly, []history.Anomaly)
	}{
		{
			name:  "Serializable",
			check: history.Serializable,
			want: func(c registerCase) (history.Anomaly, []history.Anomaly) {
				return c.wantSerializable, c.wantSerializableKinds
			},
		},
		{
			name:  "HasSnapshotIsolation",
			check: history.HasSnapshotIsolation,
			want:  func(c registerCase) (history.Anomaly, []history.Anomaly) { return c.wantSI, c.wantSIKinds },
		},
	}
	for _, level := range levels {
		t.Run(level.name, func(t *testing.T) {
			t.Parallel()
			for _, tt := range registerCases {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					h := history.New()
					tt.give(h)
					got := isolationOf(level.check, h)
					wantAnomaly, wantKinds := level.want(tt)
					if wantAnomaly == 0 {
						assert.Nil(t, got, "the check passes")
						return
					}
					assert.Equal(t, got[anomalyField], any(wantAnomaly), "the first anomaly")
					assert.Equal(t, got[kindsField], any(wantKinds), "every kind that the history exhibits")
				})
			}
		})
	}
}
