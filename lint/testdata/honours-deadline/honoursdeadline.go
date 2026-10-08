// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package honoursdeadline

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.dokimi.dev/assert"
)

func fetch(ctx context.Context) error { return ctx.Err() }

func correct(t *testing.T) {
	assert.HonoursDeadline(t, fetch, "Fetch stops when its deadline passes")
}

func expired(t *testing.T) {
	ctx, cancel := context.WithDeadline(t.Context(), time.Unix(0, 0))
	defer cancel()
	err := fetch(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "Fetch stops when its deadline passes") // want `honours-deadline: state the check with HonoursDeadline`
	if !errors.Is(err, context.DeadlineExceeded) {                                           // want `honours-deadline: state the check with HonoursDeadline`
		t.Fatal(err)
	}
	timedOut, stop := context.WithTimeout(t.Context(), 0)
	defer stop()
	assert.ErrorIs(t, fetch(timedOut), context.DeadlineExceeded, "Fetch stops when its deadline passes") // want `honours-deadline: state the check with HonoursDeadline`
	past, done := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer done()
	assert.ErrorIs(t, fetch(past), context.DeadlineExceeded, "Fetch stops when its deadline passes") // want `honours-deadline: state the check with HonoursDeadline`
}

func measured(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 0)
	defer cancel()
	var err error
	put := func() { err = fetch(ctx) }
	put()
	assert.MaxAllocs(t, put, 0, "Fetch allocates nothing")
	assert.ErrorIs(t, err, context.DeadlineExceeded, "Fetch stops when its deadline passes") // want `honours-deadline: state the check with HonoursDeadline of fetch\(ctx\)`
}

func running(t *testing.T, deadline time.Time) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	assert.ErrorIs(t, fetch(ctx), context.DeadlineExceeded, "Fetch stops when its deadline passes")
	later, stop := context.WithDeadline(t.Context(), time.Now().Add(time.Hour))
	defer stop()
	assert.ErrorIs(t, fetch(later), context.DeadlineExceeded, "Fetch stops when its deadline passes")
	given, done := context.WithDeadline(t.Context(), deadline)
	defer done()
	assert.ErrorIs(t, fetch(given), context.DeadlineExceeded, "Fetch stops when its deadline passes")
	start := time.Now()
	stamped, finish := context.WithDeadline(t.Context(), start.Add(-time.Second))
	defer finish()
	assert.ErrorIs(t, fetch(stamped), context.DeadlineExceeded, "Fetch stops when its deadline passes")
	dated, end := context.WithDeadline(t.Context(), time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
	defer end()
	assert.ErrorIs(t, fetch(dated), context.DeadlineExceeded, "Fetch stops when its deadline passes")
	cancelled, release := context.WithCancel(t.Context())
	release()
	assert.ErrorIs(t, fetch(cancelled), context.DeadlineExceeded, "Fetch stops when its deadline passes")
}
