// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"context"

	"go.dokimi.dev/assert/internal/matcher"
)

// TB is the seat that an assertion reports through.
// [testing.T], [testing.B] and [Recorder] satisfy it.
//
// [testing.TB] declares an unexported method, so no type outside the
// standard library implements it. This package declares TB so that any
// type with these three methods is a seat, such as the seat of a
// generated check body.
type TB interface {
	// Helper marks the calling function as a test helper, so a
	// failure reports its caller's line.
	Helper()
	// Fatalf records a failure and stops the test.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}

// Context returns the context of tb: the one that its method Context
// returns, as *testing.T, *testing.B, *testing.F, a *prop.Case and a
// [Recorder] state one, and context.Background() for a seat without that
// method. The seat that [Rejects] hands its check, and the seat of each
// attempt of [Eventually], state a context that derives from the context
// of the assertion's seat and ends with the body.
//
// A helper that takes a TB passes the context to the code under test, so
// that code receives the end of the test through any seat:
//
//	func loaded(tb assert.TB, path string) *catalog.Catalog {
//	    tb.Helper()
//	    c, err := catalog.Load(assert.Context(tb), path)
//	    assert.NoError(tb, err, "the catalog at "+path+" loads")
//	    return c
//	}
//
// # Allocation contract
//
// Context allocates nothing besides what the method Context of tb
// allocates.
func Context(tb TB) context.Context {
	return matcher.ContextOf(tb)
}
