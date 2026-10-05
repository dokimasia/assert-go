// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"context"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/prop"
)

// The allocations of the case's methods, measured.
const (
	// reportAllocs are the allocations of a recording Report: the rendered
	// record and the message the recorder formats from it.
	reportAllocs = 2
	// errorfAllocs are the allocations of Errorf: the message, the frames
	// searched for the caller's code, the rendered record and the message
	// the recorder formats from it.
	errorfAllocs = 5
	// fatalfRunAllocs are the allocations of a run that replays one case
	// whose body calls Fatalf, on a seat that keeps the run's record: those
	// of a replay, the message and its frames, the run's record, and the two
	// closures that adapt the body and give its case a context.
	fatalfRunAllocs = 44
	// drawRunAllocs are the allocations of a run that replays one case whose
	// body draws one integer: the engine's 9 for the replay, one for the
	// token's choices, and two for the closures that adapt the body and give
	// its case a context.
	drawRunAllocs = 12
	// logfAllocs are the allocations of Logf: the message it formats.
	logfAllocs = 1
	// contextRunAllocs are the allocations of a run that replays one case
	// whose body draws one integer and calls Context: those of
	// drawRunAllocs, and the context with its cancel function.
	contextRunAllocs = 14
	// historyRunAllocs are the allocations of a run that replays one case
	// whose body draws one integer and calls History: those of
	// drawRunAllocs, and the history.
	historyRunAllocs = 13
)

// closing is the label of a draw that a cleanup makes.
const closing = "closing"

// recorded is the record that a test body reports, as an assertion would.
var recorded = assert.Failure{
	Assertion: "equal",
	Contract:  "the totals match",
	Where:     assert.Where{File: "ledger_test.go", Line: 12},
}

// ledgerKey is the key of a value in the context of a seat. The context of
// each case that a property of the seat runs derives from that context and
// returns the value.
type ledgerKey struct{}

// contextSeat is a recorder seat with a context of its own, as a
// *testing.T has.
type contextSeat struct {
	*assert.Recorder
	// ctx is the context the seat states.
	ctx context.Context
}

// Context returns the seat's context.
func (s contextSeat) Context() context.Context {
	return s.ctx
}

// TestCase checks the case as the seat of a body's assertions and the
// source of its inputs: how each kind of failure ends it, and what it
// records.
func TestCase(t *testing.T) {
	t.Parallel()

	t.Run("Helper", func(t *testing.T) {
		t.Parallel()

		t.Run("returns without ending the case", func(t *testing.T) {
			t.Parallel()
			var after bool
			rec := assert.NewRecorder()
			replays(rec, func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Helper()
				after = true
			}, 7)
			assert.True(t, after, "the body runs on")
			assert.False(t, rec.Failed(), "the case passes")
		})
	})

	t.Run("Fatalf", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a record of the message at the caller's line", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			got := replayed(func(c *prop.Case) { c.Fatalf("stop%s", here(&at)) }, 7)
			want := assert.Failure{Contract: "stop", Where: at}
			assert.Equal(t, got[failureField], any(want), "a record without an assertion")
		})

		t.Run("ends the calling goroutine", func(t *testing.T) {
			t.Parallel()
			var after bool
			got := replayed(func(c *prop.Case) {
				c.Fatalf("stop")
				after = true
			}, 7)
			assert.False(t, after, "the body ends at Fatalf")
			assert.Equal(t, got[outcomeField], any(prop.Counterexample), "the case fails")
		})

		t.Run("ends only the goroutine that calls it", func(t *testing.T) {
			t.Parallel()
			var escaped, returned bool
			got := replayed(func(c *prop.Case) {
				var wg sync.WaitGroup
				wg.Go(func() {
					c.Fatalf("stop")
					escaped = true
				})
				wg.Wait()
				returned = true
			}, 7)
			assert.False(t, escaped, "the other goroutine ends at Fatalf")
			assert.True(t, returned, "the body runs on")
			assert.Equal(t, got[outcomeField], any(prop.Counterexample), "the case fails when its body returns")
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a record of the message at the caller's line and lets the body run on", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			var after bool
			got := replayed(func(c *prop.Case) {
				c.Errorf("late%s", here(&at))
				after = true
			}, 7)
			want := assert.Failure{Contract: "late", Where: at}
			assert.True(t, after, "the body runs on")
			assert.Equal(t, got[failureField], any(want), "a record without an assertion")
		})
	})

	t.Run("Report", func(t *testing.T) {
		t.Parallel()

		t.Run("ends the case at an aborting record", func(t *testing.T) {
			t.Parallel()
			var after bool
			got := replayed(func(c *prop.Case) {
				c.Report(recorded, true)
				after = true
			}, 7)
			assert.False(t, after, "the body ends at the record")
			assert.Equal(t, got[failureField], any(recorded), "the record")
		})

		t.Run("lets the body run on after a recording record", func(t *testing.T) {
			t.Parallel()
			var after bool
			got := replayed(func(c *prop.Case) {
				c.Report(recorded, false)
				after = true
			}, 7)
			assert.True(t, after, "the body runs on")
			assert.Equal(t, got[failureField], any(recorded), "the case fails when its body returns")
		})
	})

	t.Run("Clock", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the clock of the property's seat", func(t *testing.T) {
			t.Parallel()
			start := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
			var got time.Time
			rec := assert.NewRecorder().WithClock(assert.NewControlled(start))
			replays(rec, func(c *prop.Case) { got = c.Clock().Now() }, 7)
			assert.Equal(t, got, start, "the controlled clock's time")
		})

		t.Run("returns the system clock for a seat without a clock", func(t *testing.T) {
			t.Parallel()
			var got assert.Clock
			prop.ForAll(&sentences{}, contract, func(c *prop.Case) { got = c.Clock() }, prop.Replay(tokenOf(7)))
			assert.Equal[assert.Clock](t, got, assert.System{}, "the runtime clock")
		})
	})

	t.Run("Assume", func(t *testing.T) {
		t.Parallel()

		t.Run("rejects the case for false", func(t *testing.T) {
			t.Parallel()
			var after bool
			got := replayed(func(c *prop.Case) {
				c.Assume(false)
				after = true
			}, 7)
			assert.False(t, after, "the body ends at the rejection")
			assert.Equal(t, counts(got), []any{prop.Rejected, 0, 1}, "the one case is rejected")
		})

		t.Run("returns for true", func(t *testing.T) {
			t.Parallel()
			var after bool
			rec := assert.NewRecorder()
			replays(rec, func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Assume(true)
				after = true
			}, 7)
			assert.True(t, after, "the body runs on")
			assert.False(t, rec.Failed(), "the case passes")
		})
	})

	t.Run("Classify", func(t *testing.T) {
		t.Parallel()

		t.Run("counts a label once in a case", func(t *testing.T) {
			t.Parallel()
			twice := func(c *prop.Case) {
				if isEven(c.Draw(prop.Integer(0, 1000), drawn)) {
					c.Classify(even)
					c.Classify(even)
				}
			}
			got := detailOf(twice, prop.Seed(7), prop.Require(even, 0.9))
			want := &prop.Shortfall{Label: even, Share: 0.9, Counted: 45, Valid: 100, Verdict: prop.Refuted}
			assert.Equal(t, got[coverageField], any(want), "the count of a single classification")
		})
	})

	t.Run("Logf", func(t *testing.T) {
		t.Parallel()

		t.Run("reports nothing for a passing case", func(t *testing.T) {
			t.Parallel()
			seat := &sentences{}
			prop.ForAll(seat, contract, func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Logf("opened the ledger")
			}, prop.Seed(7))
			assert.Empty(t, seat.all(), "no sentence")
		})

		t.Run("logs the formatted messages of a failing case in order before its failure", func(t *testing.T) {
			t.Parallel()
			seat := &sentences{}
			prop.ForAll(seat, contract, func(c *prop.Case) {
				c.Logf("opened the %s", "ledger")
				c.Logf("appended %d entries", 2)
				c.Fatalf("stop")
			}, prop.Replay(tokenOf(7)))
			got := seat.all()
			assert.Length(t, got, 3, "the two messages and the sentence")
			assert.Equal(t, got[:2], []string{"opened the ledger", "appended 2 entries"},
				"the messages in the order the body attached them, before the sentence")
		})
	})

	t.Run("Rand", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a source that math/rand/v2 draws the case's choices from", func(t *testing.T) {
			t.Parallel()
			var got uint64
			rec := assert.NewRecorder()
			replays(rec, func(c *prop.Case) { got = rand.New(c.Rand()).Uint64() }, 5)
			assert.Equal(t, got, uint64(5), "the replayed choice")
		})

		t.Run("returns a source whose values are inputs of the run", func(t *testing.T) {
			t.Parallel()
			got := detailOf(func(c *prop.Case) { rand.New(c.Rand()).Uint64() }, prop.Seed(7))
			assert.Nil(t, got, "a run that is not vacuous")
		})
	})

	t.Run("Observe", func(t *testing.T) {
		t.Parallel()

		t.Run("records fingerprints that the replay of a failing case repeats", func(t *testing.T) {
			t.Parallel()
			got := detailOf(func(c *prop.Case) {
				c.Observe(42)
				c.Draw(prop.Integer(0, 9), drawn)
				fail(c, every)
			}, prop.Seed(7))
			assert.Equal(t, got[outcomeField], any(prop.Counterexample), "the same fingerprint on replay")
		})
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the cleanups after the body, the last registered first", func(t *testing.T) {
			t.Parallel()
			var order []string
			replays(assert.NewRecorder(), func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Cleanup(func() { order = append(order, "first") })
				c.Cleanup(func() { order = append(order, "second") })
				order = append(order, "body")
			}, 7)
			assert.Equal(t, order, []string{"body", "second", "first"}, "the body, then the cleanups in reverse")
		})

		t.Run("runs the cleanups of a case that Fatalf ended", func(t *testing.T) {
			t.Parallel()
			var cleaned bool
			got := replayed(func(c *prop.Case) {
				c.Cleanup(func() { cleaned = true })
				c.Fatalf("stop")
			}, 7)
			assert.True(t, cleaned, "the cleanup ran")
			assert.Equal(t, got[outcomeField], any(prop.Counterexample), "the case fails")
		})

		t.Run("runs the cleanups of every run of a shrink", func(t *testing.T) {
			t.Parallel()
			var bodies, cleanups int
			got := detailOf(func(c *prop.Case) {
				bodies++
				c.Cleanup(func() { cleanups++ })
				if c.Draw(prop.Integer(0, 1000), drawn) > 500 {
					fail(c, every)
				}
			}, prop.Seed(7))
			assert.Equal(t, got[outcomeField], any(prop.Counterexample), "a shrunk counterexample")
			assert.Equal(t, cleanups, bodies, "one run of the cleanup for every call of the body")
		})

		t.Run("fails the case at a failure in a cleanup", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			got := replayed(func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Cleanup(func() { c.Fatalf("leaked%s", here(&at)) })
			}, 7)
			want := assert.Failure{Contract: "leaked", Where: at}
			assert.Equal(t, got[failureField], any(want), "the cleanup's record")
		})

		t.Run("records a draw in a cleanup as a draw of the case", func(t *testing.T) {
			t.Parallel()
			detail := replayed(func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Cleanup(func() {
					c.Draw(prop.Integer(0, 9), closing)
					fail(c, every)
				})
			}, 7, 3)
			want := []prop.Entry{prop.Drawn{Label: drawn, Value: 7}, prop.Drawn{Label: closing, Value: 3}}
			assert.Equal(t, detail[counterexampleField], any(want), "the body's draw, then the cleanup's")
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a context that is live while the body runs", func(t *testing.T) {
			t.Parallel()
			var err error
			replays(assert.NewRecorder(), func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				err = c.Context().Err()
			}, 7)
			assert.NoError(t, err, "a live context")
		})

		t.Run("returns a context that the case cancels before its cleanups run", func(t *testing.T) {
			t.Parallel()
			var err error
			replays(assert.NewRecorder(), func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				ctx := c.Context()
				c.Cleanup(func() { err = ctx.Err() })
			}, 7)
			assert.ErrorIs(t, err, context.Canceled, "a cancelled context")
		})

		t.Run("returns a context that derives from the context of the seat", func(t *testing.T) {
			t.Parallel()
			var got any
			ctx := context.WithValue(t.Context(), ledgerKey{}, "ledger")
			seat := contextSeat{Recorder: assert.NewRecorder(), ctx: ctx}
			prop.ForAll(seat, contract, func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				got = c.Context().Value(ledgerKey{})
			}, prop.Replay(tokenOf(7)))
			assert.Equal(t, got, any("ledger"), "the seat's value")
		})
	})

	t.Run("History", func(t *testing.T) {
		t.Parallel()

		t.Run("returns one history to every call of the case, which records the body's calls", func(t *testing.T) {
			t.Parallel()
			var same bool
			var events int
			replays(assert.NewRecorder(), func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				first := c.History()
				first.Invoke(0, "read", nil).OK(nil)
				same, events = c.History() == first, len(c.History().Events())
			}, 7)
			assert.Equal(t, []any{same, events}, []any{true, 2}, "the invocation and the completion of read")
		})
	})

	t.Run("Target", func(t *testing.T) {
		t.Parallel()

		t.Run("changes nothing that a run generates or reports", func(t *testing.T) {
			t.Parallel()
			fails := failsAtLeast(10000, 1001, big)
			scored := func(c *prop.Case) {
				c.Target("depth", 1)
				fails(c)
			}
			assert.Equal(t, detailOf(scored, prop.Seed(7)), detailOf(fails, prop.Seed(7)), "the record of the run")
		})
	})

	t.Run("Draw", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the replayed value and records it under its label", func(t *testing.T) {
			t.Parallel()
			var got int
			detail := replayed(func(c *prop.Case) {
				got = c.Draw(prop.Integer(0, 9), drawn)
				fail(c, every)
			}, 7)
			assert.Equal(t, got, 7, "the replayed value")
			assert.Equal(t, detail[counterexampleField], any([]prop.Entry{prop.Drawn{Label: drawn, Value: 7}}),
				"the draw")
		})

		t.Run("records two draws under one label", func(t *testing.T) {
			t.Parallel()
			detail := replayed(func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Draw(prop.Integer(0, 9), drawn)
				fail(c, every)
			}, 7, 3)
			want := []prop.Entry{prop.Drawn{Label: drawn, Value: 7}, prop.Drawn{Label: drawn, Value: 3}}
			assert.Equal(t, detail[counterexampleField], any(want), "both draws in order")
		})
	})
}

// TestCaseAllocs checks the allocation ceilings of the case's methods. A
// method that ends the calling goroutine, and a draw, are measured on a
// whole run that replays one case.
func TestCaseAllocs(t *testing.T) {
	c := leaked()
	rec := &matchertest.Seat{}
	seven := tokenOf(7)
	stop := func(c *prop.Case) { c.Fatalf("stop") }
	draw := draws(prop.Integer(0, 9))
	assert.MaxAllocs(t, func() { c.Helper() }, 0, "Helper allocates nothing")
	assert.MaxAllocs(t, func() { c.Report(recorded, false) }, reportAllocs, "Report allocates its message")
	assert.MaxAllocs(t, func() { prop.ForAll(rec, contract, stop, prop.Replay(seven)) }, fatalfRunAllocs,
		"a run of a case that calls Fatalf")
	assert.MaxAllocs(t, func() { c.Errorf("late") }, errorfAllocs, "Errorf allocates its message and its frames")
	assert.MaxAllocs(t, func() { _ = c.Clock() }, 0, "Clock allocates nothing")
	assert.MaxAllocs(t, func() { c.Assume(true) }, 0, "Assume allocates nothing")
	assert.MaxAllocs(t, func() { c.Classify(small) }, 0, "Classify allocates nothing for a counted label")
	assert.MaxAllocs(t, func() { c.Logf("seen") }, logfAllocs, "Logf allocates its message")
	assert.MaxAllocs(t, func() { c.Observe(42) }, 0, "Observe allocates only to grow the record")
	assert.MaxAllocs(t, func() { _ = c.Rand() }, 0, "Rand allocates nothing")
	assert.MaxAllocs(t, func() { c.Cleanup(nothing) }, 0, "Cleanup allocates only to grow the record")
	assert.MaxAllocs(t, func() { prop.ForAll(rec, contract, draw, prop.Replay(seven)) }, drawRunAllocs,
		"a run of a case that draws one integer")
	digit := prop.Integer(0, 9)
	contextual := func(c *prop.Case) {
		c.Draw(digit, drawn)
		_ = c.Context()
	}
	assert.MaxAllocs(t, func() { prop.ForAll(rec, contract, contextual, prop.Replay(seven)) }, contextRunAllocs,
		"a run of a case that calls Context")
	assert.MaxAllocs(t, func() { prop.ForAll(rec, contract, historical, prop.Replay(seven)) }, historyRunAllocs,
		"a run of a case that calls History")
	assert.MaxAllocs(t, func() { c.Target("depth", 1) }, 0, "Target allocates nothing for a scored label")
}

// historicalDigits is the generator of the draw of historical.
var historicalDigits = prop.Integer(0, 9)

// historical is the body of a case that draws one integer and asks for its
// history.
func historical(c *prop.Case) {
	c.Draw(historicalDigits, drawn)
	c.History()
}

// BenchmarkCase measures each method of the case. A method that ends the
// calling goroutine, and a draw, are measured on a whole run that replays
// one case.
func BenchmarkCase(b *testing.B) {
	b.Run("Helper", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Helper()
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Fatalf", func(b *testing.B) {
		rec, seven := &matchertest.Seat{}, tokenOf(7)
		stop := func(c *prop.Case) { c.Fatalf("stop") }
		c := bench.Start(b).MaxAllocs(fatalfRunAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, stop, prop.Replay(seven))
		}
		assert.Equal(b, rec.Records()[0].Detail[outcomeField], any(prop.Counterexample), "the case fails")
	})

	b.Run("Errorf", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(errorfAllocs)
		defer c.End()
		for c.Loop() {
			cs.Errorf("late")
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Report", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(reportAllocs)
		defer c.End()
		for c.Loop() {
			cs.Report(recorded, false)
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Clock", func(b *testing.B) {
		var got assert.Clock
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = cs.Clock()
		}
		assert.Equal[assert.Clock](b, got, assert.System{}, "the runtime clock")
	})

	b.Run("Assume", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Assume(true)
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Classify", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Classify(small)
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Logf", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(logfAllocs)
		defer c.End()
		for c.Loop() {
			cs.Logf("seen")
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Rand", func(b *testing.B) {
		var got rand.Source
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = cs.Rand()
		}
		assert.NotNil(b, got, "the case's source")
	})

	b.Run("Observe", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Observe(42)
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Cleanup", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Cleanup(nothing)
		}
		assert.NotNil(b, cs, "the case")
	})

	b.Run("Context", func(b *testing.B) {
		rec, seven, digit := &matchertest.Seat{}, tokenOf(7), prop.Integer(0, 9)
		contextual := func(c *prop.Case) {
			c.Draw(digit, drawn)
			_ = c.Context()
		}
		c := bench.Start(b).MaxAllocs(contextRunAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, contextual, prop.Replay(seven))
		}
		assert.False(b, rec.Failed(), "every run passes")
	})

	b.Run("Draw", func(b *testing.B) {
		rec, seven := &matchertest.Seat{}, tokenOf(7)
		draw := draws(prop.Integer(0, 9))
		c := bench.Start(b).MaxAllocs(drawRunAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, draw, prop.Replay(seven))
		}
		assert.False(b, rec.Failed(), "every run passes")
	})

	b.Run("History", func(b *testing.B) {
		rec, seven := &matchertest.Seat{}, tokenOf(7)
		c := bench.Start(b).MaxAllocs(historyRunAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, historical, prop.Replay(seven))
		}
		assert.False(b, rec.Failed(), "every run passes")
	})

	b.Run("Target", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Target("depth", 1)
		}
		assert.NotNil(b, cs, "the case")
	})
}

// nothing is a cleanup that does nothing.
func nothing() {}

// leaked returns the case of a run that replayed one case, which drew 7
// from the digits, classified itself as small and scored 3 under depth, for
// a caller to call after the body returned.
func leaked() *prop.Case {
	var c *prop.Case
	prop.ForAll(assert.NewRecorder(), contract, func(cs *prop.Case) {
		c = cs
		cs.Draw(prop.Integer(0, 9), drawn)
		cs.Classify(small)
		cs.Target("depth", 3)
	}, prop.Replay(tokenOf(7)))
	return c
}

// replays runs body as the property contract on rec, replaying the case of
// one integer choice for each value.
func replays(rec *assert.Recorder, body func(*prop.Case), values ...uint64) {
	prop.ForAll(rec, contract, body, prop.Replay(tokenOf(values...)))
}
