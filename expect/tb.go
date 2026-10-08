// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"context"

	"go.dokimi.dev/assert"
)

// TB is the seat that an assertion reports through, the seat of
// [go.dokimi.dev/assert]. [testing.T], [testing.B] and [Recorder] satisfy
// it.
type TB = assert.TB

// Context returns the context of tb, as [go.dokimi.dev/assert.Context]
// does: the one that its method Context returns, and context.Background()
// for a seat without that method.
//
// # Allocation contract
//
// Context allocates nothing besides what the method Context of tb
// allocates.
func Context(tb TB) context.Context {
	return assert.Context(tb)
}
