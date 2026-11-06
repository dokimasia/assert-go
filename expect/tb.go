// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"context"

	"go.dokimi.dev/assert"
)

// TB is the seat of [go.dokimi.dev/assert]. An assertion reports through it.
// [testing.T], [testing.B] and [Recorder] satisfy it.
type TB = assert.TB

// Context returns the same context as [go.dokimi.dev/assert.Context]. That is
// the context of the first seat of the chain of tb with the method Context.
// For a chain without the method, Context returns context.Background().
//
// # Allocation contract
//
// Context allocates nothing besides what the method Context of tb
// allocates.
func Context(tb TB) context.Context {
	return assert.Context(tb)
}

// Cleanup registers f in the same way as [go.dokimi.dev/assert.Cleanup]. f
// runs when the test of tb ends. Cleanup calls the method Cleanup of tb, or of
// the first seat that tb wraps with the method. On a chain without the method,
// Cleanup ends the call with a fault. f does not run then.
//
// # Allocation contract
//
// Cleanup allocates nothing besides what the method Cleanup of tb
// allocates.
func Cleanup(tb TB, f func()) {
	tb.Helper()
	assert.Cleanup(tb, f)
}

// Skipf skips the test of tb in the same way as [go.dokimi.dev/assert.Skipf].
// The reason is the text that fmt.Sprintf formats from format and args. On a
// chain without the method Skipf, Skipf ends the call with a fault. The test
// does not skip then.
//
// # Allocation contract
//
// Skipf allocates nothing besides what the method Skipf of tb allocates.
func Skipf(tb TB, format string, args ...any) {
	tb.Helper()
	assert.Skipf(tb, format, args...)
}

// WriteRecord writes record into the output of the test of tb in the same way
// as [go.dokimi.dev/assert.WriteRecord]. record is the JSON text of one record
// of kind. For a refused kind or record, WriteRecord ends the call with a
// fault and does not write the record.
//
// # Allocation contract
//
// WriteRecord allocates what [go.dokimi.dev/assert.WriteRecord] allocates.
func WriteRecord(tb TB, kind, record string) {
	tb.Helper()
	assert.WriteRecord(tb, kind, record)
}
