// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package snapshotisolation

import (
	"testing"

	"go.dokimi.dev/assert/history"
)

func correct(t *testing.T) {
	h := history.New()
	call := h.Invoke(0, "txn", []any{[]any{"read", "y", nil}}, "y")
	call.OK([]any{[]any{"read", "y", []any{}}})
	history.HasSnapshotIsolation(t, h, "the store has snapshot isolation")
}
