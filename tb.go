// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"context"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/wrap"
)

// The packages of this module look for the methods of a seat along the
// seats that it wraps. None of them can refer to TB, because this package
// imports them. This package registers the function that calls Unwrap while
// it initialises.
func init() {
	wrap.Register(unwrap)
}

// TB is the seat through which an assertion reports.
// [testing.T], [testing.B] and [Recorder] satisfy it.
//
// [testing.TB] declares an unexported method, so no type outside the
// standard library implements it. This package declares TB so that any
// type with these three methods is a seat. The seat of a generated check
// body is such a type.
//
// A seat that wraps another seat declares a method Unwrap() TB. The method
// returns the wrapped seat or nil. An assertion calls Helper, Fatalf and
// Errorf of the seat itself. It looks for every other method on the seat and
// then on each seat that Unwrap returns, in order. Report, Logf, Skipf,
// Clock, Context and Cleanup are such methods. The first seat that has the
// method provides it. The first seat that keeps call records receives the
// call records. A wrapper declares each method that it changes:
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
// The chain of seats that Unwrap returns must not contain the wrapper.
type TB interface {
	// Helper marks the calling function as a test helper. The seat then
	// reports a failure at the line of the caller of that function.
	Helper()
	// Fatalf records a failure and stops the test.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}

// unwrap returns the seat that the method Unwrap of seat returns. For a seat
// without the method, it returns nil.
func unwrap(seat any) any {
	if w, ok := seat.(interface{ Unwrap() TB }); ok {
		return w.Unwrap()
	}
	return nil
}

// Context returns the context that the method Context of tb returns.
// *testing.T, *testing.B, *testing.F, a *prop.Case and a [Recorder] have the
// method. For a seat without the method, Context returns the context of the
// first seat that the seat wraps with the method. It returns
// context.Background() when no seat of the chain has the method. The doc
// comment of [TB] describes the chain. The seat that [Rejects] hands its check
// and the seat of each attempt of [Eventually] have a context of their own.
// Each context derives from the context of the seat of the assertion and ends
// with the body.
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

// Cleanup registers f through the method Cleanup of tb. f runs when the test
// of tb ends. *testing.T, *testing.B, *testing.F, a *prop.Case and a
// [Recorder] have the method. For a seat without the method, Cleanup
// registers f with the first seat that the seat wraps with the method. The
// doc comment of [TB] describes the chain. The seat that [Rejects] hands its
// check and the seat of each attempt of [Eventually] run their cleanups when
// the body ends. They run them before the assertion returns. On a chain
// without the method, Cleanup ends the call with a fault as [Fault] ends one.
// f does not run then.
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

// Skipf skips the test of tb. The reason is the text that fmt.Sprintf formats
// from format and args. Skipf calls the method Skipf of tb, or of the first
// seat that tb wraps with the method. The doc comment of [TB] describes the
// chain. These seats have the method:
//
//   - A *testing.T, a *testing.B and a *testing.F end the test as skipped,
//     and Skipf does not return.
//   - A *prop.Case ends the run of its property as skipped.
//   - The seat that [Rejects] hands its check ends the check. Rejects then
//     ends its call as skipped when the check did not fail before the skip.
//
// On a chain without the method, Skipf ends the call with a fault as [Fault]
// ends one. The test does not skip. A [Recorder] does not have the method, so
// it keeps the fault among its faults.
//
// A helper that takes a TB skips the test where the environment lacks what
// the test needs:
//
//	func stream(tb assert.TB) string {
//	    tb.Helper()
//	    addr := os.Getenv("NATS_ADDR")
//	    if addr == "" {
//	        assert.Skipf(tb, "NATS_ADDR is unset: start the local stack first")
//	    }
//	    return addr
//	}
//
// # Allocation contract
//
// Skipf allocates nothing besides what the method Skipf of tb allocates. On
// a chain without the method, it allocates the fault and its text.
func Skipf(tb TB, format string, args ...any) {
	tb.Helper()
	matcher.Skip(tb, "assert.Skipf", format, args...)
}

// WriteRecord writes record into the output of the test of tb. record is the
// JSON text of one record of kind. The output receives an attr event with the
// key <kind>.<n>. WriteRecord numbers the records of each kind of a test from
// 1. An attr event is one line, so WriteRecord replaces each line break of
// record with a space. The JSON value of the record does not change. A record
// that does not fit the line buffer of test2json is split over consecutive
// attr events of the key, as a call record is. A tool of dokimasia writes its
// own records into the output of a test this way:
//
//	assert.WriteRecord(t, "dokimi.suite", `{"suite":"ledger","seed":"7"}`)
//
// WriteRecord looks for the seat of the test along the chain of tb. The doc
// comment of [TB] describes the chain. WriteRecord writes into the first seat
// that is or embeds a *testing.T, *testing.B or *testing.F. It does not write
// when a seat that keeps its call records as values comes first. A [Recorder]
// is such a seat. WriteRecord ignores DOKIMI_ASSERT_RECORD.
//
// In these cases WriteRecord ends the call with a fault as [Fault] ends one
// and does not write the record:
//
//   - kind does not match dokimi\.[a-z][a-z0-9-]*.
//   - kind is dokimi.assert, the kind of the call records of this module.
//   - record is not the JSON text of one object.
//
// # Allocation contract
//
// WriteRecord allocates a copy of record to check it. On a test's seat it also
// allocates the key of the record and what the method Attr allocates. For a
// refused kind or record it allocates the fault.
func WriteRecord(tb TB, kind, record string) {
	tb.Helper()
	matcher.WriteRecord(tb, "assert.WriteRecord", kind, record)
}
