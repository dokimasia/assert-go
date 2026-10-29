// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"context"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/wrap"
)

// The packages of this module look for the methods of a seat along the
// seats that it wraps, and none of them can name TB. This package registers
// the function that calls Unwrap while it initialises.
func init() {
	wrap.Register(unwrap)
}

// TB is the seat that an assertion reports through.
// [testing.T], [testing.B] and [Recorder] satisfy it.
//
// [testing.TB] declares an unexported method, so no type outside the
// standard library implements it. This package declares TB so that any
// type with these three methods is a seat, such as the seat of a
// generated check body.
//
// A seat that wraps another seat declares a method Unwrap() TB, which
// returns the seat that it wraps, or nil. An assertion calls Helper,
// Fatalf and Errorf of the seat itself. It looks for every other method,
// such as Report, Logf, Skipf, Clock, Context and Cleanup, on the seat and
// then on each seat that Unwrap returns, in order. The first seat that has the
// method provides it, and the call records are those of the first seat that
// keeps call records. A wrapper declares each method that it changes:
//
//	// handle is the seat of a check, with the clock that the check controls.
//	type handle struct {
//	    assert.TB
//	    clock *assert.Controlled
//	}
//
//	func (h *handle) Clock() assert.Clock { return h.clock }
//	func (h *handle) Unwrap() assert.TB   { return h.TB }
//
// The seats that Unwrap returns must not lead back to the wrapper.
type TB interface {
	// Helper marks the calling function as a test helper, so a
	// failure reports its caller's line.
	Helper()
	// Fatalf records a failure and stops the test.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}

// unwrap returns the seat that seat wraps: the one that its method Unwrap
// returns, and nil for a seat without Unwrap.
func unwrap(seat any) any {
	if w, ok := seat.(interface{ Unwrap() TB }); ok {
		return w.Unwrap()
	}
	return nil
}

// Context returns the context that the method Context of tb returns, as
// *testing.T, *testing.B, *testing.F, a *prop.Case and a [Recorder] state
// one. A seat without the method gives the context of the first seat that
// it wraps with the method, as [TB] states, and context.Background() when
// it wraps none. The seat that [Rejects] hands its check, and the seat of
// each attempt of [Eventually], state a context that derives from the
// context of the assertion's seat and ends with the body.
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

// Cleanup registers f to run when the test of tb ends, through the method
// Cleanup of tb. *testing.T, *testing.B, *testing.F, a *prop.Case and a
// [Recorder] have the method. A seat without it registers f with the first
// seat that it wraps with the method, as [TB] states. The seat that [Rejects]
// hands its check, and the seat of each attempt of [Eventually], run their
// cleanups when the body ends, before the assertion returns. A chain without
// the method ends the call with a fault, as [Fault] ends one, and f does not
// run.
//
// A helper that takes a TB registers the release of what it starts:
//
//	func witness(tb assert.TB) *httptest.Server {
//	    tb.Helper()
//	    srv := httptest.NewServer(newWitness())
//	    assert.Cleanup(tb, srv.Close)
//	    return srv
//	}
//
// # Allocation contract
//
// Cleanup allocates nothing besides what the method Cleanup of tb
// allocates.
func Cleanup(tb TB, f func()) {
	tb.Helper()
	matcher.Cleanup(tb, "assert.Cleanup", f)
}
