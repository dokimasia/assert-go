// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package linearizable

import (
	"testing"

	"go.dokimi.dev/assert/history"
)

func next(s int, op history.Operation) []int {
	switch op.Name {
	case "write":
		return []int{op.Args[0].(int)}
	case "read":
		if op.Returned(s) {
			return []int{s}
		}
	}
	return nil
}

func correct(t *testing.T) {
	h := history.New()
	call := h.Invoke(0, "write", []any{1}, "x")
	call.OK(nil)
	history.Linearizable(t, h, history.Spec[int]{Initial: func() int { return 0 }, Next: next}, "the register is linearizable")
}
