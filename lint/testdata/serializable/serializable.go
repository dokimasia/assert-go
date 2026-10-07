// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package serializable

import (
	"testing"

	"go.dokimi.dev/assert/history"
)

func correct(t *testing.T) {
	h := history.New()
	call := h.Invoke(0, "txn", []any{[]any{"append", "x", 7}}, "x")
	call.OK([]any{[]any{"append", "x", 7}})
	history.Serializable(t, h, "the store is serializable")
}
