// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package honourscancellation

import (
	"context"
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type holder struct {
	ctx context.Context
	err error
}

func fetch(ctx context.Context) error { return ctx.Err() }

func count(ctx context.Context) (int, error) { return 0, ctx.Err() }

func ended(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func live(t *testing.T) context.Context { return t.Context() }

func broken() context.Context { panic("no context") }

func forever(t *testing.T) context.Context { return forever(t) }

func prepare(ctx *context.Context) {}

func correct(t *testing.T) {
	assert.HonoursCancellation(t, fetch, "Fetch stops when its context is cancelled")
}

func cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := fetch(ctx)
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context is cancelled")         // want `honours-cancellation: state the check with HonoursCancellation`
	expect.True(t, errors.Is(err, context.Canceled), "Fetch stops when its context is cancelled") // want `honours-cancellation: state the check with HonoursCancellation`
	assert.Equal(t, err, context.Canceled, "Fetch stops when its context is cancelled")           // want `honours-cancellation: state the check with HonoursCancellation`
	if err != context.Canceled {                                                                  // want `honours-cancellation: state the check with HonoursCancellation`
		t.Fatalf("Fetch returns %v", err)
	}
}

func stepped(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	name := "Fetch"
	cancel()
	assert.ErrorIs(t, fetch(ctx), context.Canceled, name+" stops when its context is cancelled") // want `honours-cancellation: state the check with HonoursCancellation of fetch\(ctx\)`
}

func helped(t *testing.T) {
	assert.ErrorIs(t, fetch(ended(t)), context.Canceled, "Fetch stops when its context is cancelled") // want `honours-cancellation: state the check with HonoursCancellation`
	ctx := ended(t)
	assert.ErrorIs(t, fetch(ctx), context.Canceled, "Fetch stops when its context is cancelled") // want `honours-cancellation: state the check with HonoursCancellation`
}

func measured(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var err error
	put := func() { _, err = count(ctx) }
	put()
	assert.MaxAllocs(t, put, 0, "Count allocates nothing")
	assert.ErrorIs(t, err, context.Canceled, "Count stops when its context is cancelled") // want `honours-cancellation: state the check with HonoursCancellation of count\(ctx\)`
}

func measuredHelped(t *testing.T) {
	var err error
	put := func() { err = fetch(ended(t)) }
	put()
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context is cancelled") // want `honours-cancellation: state the check with HonoursCancellation`
}

func running(t *testing.T, h holder, given context.Context) {
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- fetch(ctx) }()
	cancel()
	assert.ErrorIs(t, <-done, context.Canceled, "Fetch stops when its context ends")
	later, stop := context.WithCancel(t.Context())
	err := fetch(later)
	stop()
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context ends")
	assert.ErrorIs(t, ctx.Err(), context.Canceled, "the context ends")
	kept, _ := context.WithCancel(t.Context())
	assert.ErrorIs(t, fetch(kept), context.Canceled, "Fetch stops when its context ends")
	assert.ErrorIs(t, fetch(h.ctx), context.Canceled, "Fetch stops when its context ends")
	assert.ErrorIs(t, fetch(given), context.Canceled, "Fetch stops when its context ends")
	assert.ErrorIs(t, fetch(live(t)), context.Canceled, "Fetch stops when its context ends")
	assert.ErrorIs(t, fetch(broken()), context.Canceled, "Fetch stops when its context ends")
	assert.ErrorIs(t, fetch(forever(t)), context.Canceled, "Fetch stops when its context ends")
	var addressed context.Context
	prepare(&addressed)
	assert.ErrorIs(t, fetch(addressed), context.Canceled, "Fetch stops when its context ends")
	assert.ErrorIs(t, h.err, context.Canceled, "the holder keeps the error of its context")
}

func measuredLate(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var err error
	put := func() { err = fetch(ctx) }
	cancel()
	put()
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context ends")
}

func measuredReplaced(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var err error
	put := func() { err = fetch(ctx) }
	ctx = t.Context()
	put()
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context ends")
}

func measuredRenewed(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var err error
	put := func() {
		ctx = t.Context()
		err = fetch(ctx)
	}
	put()
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context ends")
}

func measuredAssigned(t *testing.T) {
	var err error
	put := func() { err = context.Canceled }
	put()
	assert.ErrorIs(t, err, context.Canceled, "the literal assigns the error itself")
}

func measuredTwice(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var err error
	put := func() {
		err = fetch(ctx)
		err = fetch(t.Context())
	}
	put()
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context ends")
}

func looped(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var err error
	for range 2 {
		err = fetch(ctx)
	}
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context ends")
}

func overwritten(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := fetch(ctx)
	put := func() { err = fetch(t.Context()) }
	put()
	assert.ErrorIs(t, err, context.Canceled, "Fetch stops when its context ends")
}
