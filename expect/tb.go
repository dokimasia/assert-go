// Copyright Dokimasia B.V. 2026
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

// Cleanup registers f to run when the test of tb ends, as
// [go.dokimi.dev/assert.Cleanup] does: through the method Cleanup of tb, or of
// the first seat that it wraps with the method. A chain without the method
// ends the call with a fault, and f does not run.
//
// # Allocation contract
//
// Cleanup allocates nothing besides what the method Cleanup of tb
// allocates.
func Cleanup(tb TB, f func()) {
	tb.Helper()
	assert.Cleanup(tb, f)
}

// WriteRecord writes record into the output of the test of tb, as
// [go.dokimi.dev/assert.WriteRecord] does. record is the JSON text of one
// record of kind. For a refused kind or record, WriteRecord ends the call with
// a fault and does not write the record.
//
// # Allocation contract
//
// WriteRecord allocates what [go.dokimi.dev/assert.WriteRecord] allocates.
func WriteRecord(tb TB, kind, record string) {
	tb.Helper()
	assert.WriteRecord(tb, kind, record)
}
