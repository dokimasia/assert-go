// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"context"
	"sync"
)

// Seat is where a matcher sends a failure or a fault.
//
// A matcher calls no test framework directly. It reports through a Seat,
// so one comparison works on a test, a benchmark and a recorder alike. The
// public TB interface declares the same three methods and satisfies this
// one structurally, so neither package imports the other.
type Seat interface {
	// Helper marks the calling frame as a helper, so a failure is
	// attributed to the caller's line rather than to the matcher.
	Helper()
	// Fatalf records a failure and stops the test. It may not return.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}

// ContextOf returns the context of seat: the one that its method Context
// returns, as testing.TB states one, and context.Background() for a seat
// without that method or whose method returns nil.
//
// # Allocation contract
//
// ContextOf allocates nothing besides what the method Context of seat
// allocates.
func ContextOf(seat Seat) context.Context {
	if s, ok := seat.(interface{ Context() context.Context }); ok {
		if ctx := s.Context(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

// bodyContext is the context of the seat that an assertion makes for a
// body: the check of [Rejects], or an attempt of [Eventually]. It derives
// from the context of the assertion's seat on its first read, and the
// assertion cancels it when the body ends, so work that the body starts
// with it stops before the assertion returns.
type bodyContext struct {
	// parent is the assertion's seat, from whose context the body's
	// context derives.
	parent Seat

	mu     sync.Mutex
	ctx    context.Context //nolint:containedctx // a body's seat has the context of the body, as testing.T has
	cancel context.CancelFunc
	// ended records that the body ended, so that a first read after the
	// end returns a cancelled context.
	ended bool
}

// Context returns the body's context, which derives from the context of
// the assertion's seat on the first call. Each call after it returns the
// same context. A first call after the body ended returns a context that
// is already cancelled.
func (b *bodyContext) Context() context.Context {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ctx == nil {
		b.ctx, b.cancel = context.WithCancel(ContextOf(b.parent))
		if b.ended {
			b.cancel()
		}
	}
	return b.ctx
}

// end marks the body ended, and cancels its context where a read derived
// one.
func (b *bodyContext) end() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ended = true
	if b.cancel != nil {
		b.cancel()
	}
}

// FaultReporter is a [Seat] that takes a fault as the error it is, rather
// than as the writer's text of it, as a [Reporter] takes a failure's record.
//
// A FaultReporter receives the fault of a call that ends without a verdict,
// such as one of [Fault], with ending true, and a fault that [NoteFault]
// notes for a call that runs on, with ending false. Any other seat receives
// the writer's text of the first through Fatalf, and of the second through
// its log.
type FaultReporter interface {
	ReportFault(err error, ending bool)
}

// Mode selects which of a [Seat]'s two failure methods a matcher uses.
// The zero value is [Fatal].
type Mode int

const (
	// Fatal reports through [Seat.Fatalf], stopping the test at the
	// first failure.
	Fatal Mode = iota
	// Soft reports through [Seat.Errorf], so the test runs on and
	// later failures are reported too.
	Soft
)
