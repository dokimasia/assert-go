// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/token"
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
	// whose body calls Fatalf: those of a replay, the message and its frames,
	// and the run's record with the sentence the recorder formats from it.
	fatalfRunAllocs = 48
	// drawRunAllocs are the allocations of a run that replays one case whose
	// body draws one integer: the engine's 13 for the replay, one for the
	// token's choices, and one for the adapter of the body.
	drawRunAllocs = 15
)

// recorded is the record that a test body reports, as an assertion would.
var recorded = assert.Failure{
	Assertion: "equal",
	Contract:  "the totals match",
	Where:     assert.Where{File: "ledger_test.go", Line: 12},
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

	t.Run("Note", func(t *testing.T) {
		t.Parallel()

		t.Run("reports nothing for a passing case", func(t *testing.T) {
			t.Parallel()
			seat := &sentences{}
			prop.ForAll(seat, contract, func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Note("opened the ledger")
			}, prop.Seed(7))
			assert.Empty(t, seat.all(), "no sentence")
		})

		t.Run("reports the notes of a failing case in order", func(t *testing.T) {
			t.Parallel()
			seat := &sentences{}
			prop.ForAll(seat, contract, func(c *prop.Case) {
				c.Note("opened the ledger")
				c.Note("appended twice")
				c.Fatalf("stop")
			}, prop.Replay(tokenOf(7)))
			assert.Length(t, seat.all(), 1, "one sentence")
			assert.ContainsInOrder(t, seat.all()[0], []string{"opened the ledger", "appended twice"},
				"the notes in the order the body attached them")
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
			assert.Equal(t, detail[counterexampleField], any([]prop.Drawn{{Label: drawn, Value: 7}}), "the draw")
		})

		t.Run("records two draws under one label", func(t *testing.T) {
			t.Parallel()
			detail := replayed(func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Draw(prop.Integer(0, 9), drawn)
				fail(c, every)
			}, 7, 3)
			want := []prop.Drawn{{Label: drawn, Value: 7}, {Label: drawn, Value: 3}}
			assert.Equal(t, detail[counterexampleField], any(want), "both draws in order")
		})
	})
}

// TestCaseZeroAlloc checks the allocation ceilings of the case's methods. A
// method that ends the calling goroutine, and a draw, are measured on a
// whole run that replays one case.
func TestCaseZeroAlloc(t *testing.T) {
	c := leaked()
	rec := assert.NewRecorder()
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
	assert.MaxAllocs(t, func() { c.Note("seen") }, 0, "Note allocates only to grow the record")
	assert.MaxAllocs(t, func() { c.Observe(42) }, 0, "Observe allocates only to grow the record")
	assert.MaxAllocs(t, func() { _ = c.Rand() }, 0, "Rand allocates nothing")
	assert.MaxAllocs(t, func() { prop.ForAll(rec, contract, draw, prop.Replay(seven)) }, drawRunAllocs,
		"a run of a case that draws one integer")
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
		rec, seven := assert.NewRecorder(), tokenOf(7)
		stop := func(c *prop.Case) { c.Fatalf("stop") }
		c := bench.Start(b).MaxAllocs(fatalfRunAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, stop, prop.Replay(seven))
		}
		assert.Equal(b, rec.Failures()[0].Detail[outcomeField], any(prop.Counterexample), "the case fails")
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

	b.Run("Note", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Note("seen")
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

	b.Run("Draw", func(b *testing.B) {
		rec, seven := assert.NewRecorder(), tokenOf(7)
		draw := draws(prop.Integer(0, 9))
		c := bench.Start(b).MaxAllocs(drawRunAllocs)
		defer c.End()
		for c.Loop() {
			prop.ForAll(rec, contract, draw, prop.Replay(seven))
		}
		assert.False(b, rec.Failed(), "every run passes")
	})
}

// leaked returns the case of a run that replayed one case, which drew 7
// from the digits and classified itself as small, for a caller to call
// after the body returned.
func leaked() *prop.Case {
	var c *prop.Case
	prop.ForAll(assert.NewRecorder(), contract, func(cs *prop.Case) {
		c = cs
		cs.Draw(prop.Integer(0, 9), drawn)
		cs.Classify(small)
	}, prop.Replay(tokenOf(7)))
	return c
}

// replays runs body as the property contract on rec, replaying the case of
// one integer choice for each value.
func replays(rec *assert.Recorder, body func(*prop.Case), values ...uint64) {
	prop.ForAll(rec, contract, body, prop.Replay(tokenOf(values...)))
}

// replayed runs body on a recorder, replaying the case of one integer
// choice for each value, and returns the detail of the run's record, or
// nil for a run that passed.
func replayed(body func(*prop.Case), values ...uint64) map[string]any {
	return detailOf(body, prop.Replay(tokenOf(values...)))
}

// tokenOf returns the replay token of one integer choice for each value.
func tokenOf(values ...uint64) string {
	choices := make([]choice.Choice, len(values))
	for i, v := range values {
		choices[i] = choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)}
	}
	return token.Encode(choices)
}
