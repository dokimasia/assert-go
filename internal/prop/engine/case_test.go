// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocation ceilings of the case's methods.
const (
	// reportAllocs is the ceiling of the allocations of a recording Report:
	// the rendered record and the message the recorder formats from it.
	reportAllocs = 3
	// errorfAllocs is the ceiling of the allocations of Errorf: the
	// message, the frames searched for the caller's code, the rendered
	// record and the message the recorder formats from it.
	errorfAllocs = 7
	// fatalCaseAllocs is the ceiling of the allocations of a whole replayed
	// case whose body calls Fatalf.
	fatalCaseAllocs = 13
	// historyCaseAllocs is the ceiling of the allocations of a whole
	// replayed case whose body calls History: the case's own three, and the
	// history.
	historyCaseAllocs = 5
	// targetsAllocs is the ceiling of the allocations of Targets: the map
	// of the copy and its storage.
	targetsAllocs = 3
	// contextCaseAllocs is the ceiling of the allocations of a whole
	// replayed case whose body calls Context: the case's own three, and the
	// context with its cancel function.
	contextCaseAllocs = 7
	// countedCaseAllocs is the ceiling of the allocations of a whole
	// replayed case whose body counts one value: the case's own three, the
	// owner, the case's map of counts with its storage, the owner's stack,
	// and the function that ends the count.
	countedCaseAllocs = 10
)

// owner is the owner of a count of a test body. Its field gives it a size,
// so two owners have two addresses.
type owner struct {
	_ byte
}

// wideMax is the upper bound of wideRange, which a random case of the
// reference seed draws in none of its first cases.
const wideMax = 1 << 40

// wideRange are the bounds [0, wideMax] of a choice that a test body makes
// on the case itself.
var wideRange = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(wideMax))

// TestCase checks the case as the TB of a body's assertions: how each kind
// of failure ends it, and what it records.
func TestCase(t *testing.T) {
	t.Parallel()

	t.Run("Helper", func(t *testing.T) {
		t.Parallel()

		t.Run("returns without ending the case", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Helper()
				after = true
			}, nil, nil)
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CasePassed, "the case passes")
		})
	})

	t.Run("Report", func(t *testing.T) {
		t.Parallel()

		t.Run("ends the case at an aborting record", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, true)
				after = true
			}, nil, nil)
			assert.False(t, after, "the body ends at the record")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
			assert.Equal(t, e.Identity, engine.Identity{Assertion: "equal", Where: reported.Where},
				"the assertion and its location")
		})

		t.Run("lets the body run on after a recording record", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, false)
				after = true
			}, nil, nil)
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails when its body returns")
		})

		t.Run("keeps every record in call order", func(t *testing.T) {
			t.Parallel()
			second := assert.Failure{
				Assertion: "true",
				Contract:  "the ledger balances",
				Where:     assert.Where{File: "ledger_test.go", Line: 20},
			}
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, false)
				c.Report(second, false)
			}, nil, nil)
			assert.Equal(t, e.Case.Failures(), []assert.Failure{reported, second}, "both records")
			assert.Equal(t, e.Identity.Assertion, "equal", "the first record is the case's failure")
		})
	})

	t.Run("Fatalf", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a record of the message at the caller's line", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			e := engine.Replay(func(c *engine.Case) { c.Fatalf("stop at %d", site(&at)) }, nil, nil)
			want := assert.Failure{Contract: fmt.Sprintf("stop at %d", at.Line), Where: at}
			assert.Equal(t, e.Case.Failures(), []assert.Failure{want}, "a record without an assertion")
		})

		t.Run("ends the calling goroutine", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Fatalf("stop")
				after = true
			}, nil, nil)
			assert.False(t, after, "the body ends at Fatalf")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
		})

		t.Run("ends only the goroutine that calls it", func(t *testing.T) {
			t.Parallel()
			var escaped, returned bool
			e := engine.Replay(func(c *engine.Case) {
				var wg sync.WaitGroup
				wg.Go(func() {
					c.Fatalf("stop")
					escaped = true
				})
				wg.Wait()
				returned = true
			}, nil, nil)
			assert.False(t, escaped, "the other goroutine ends at Fatalf")
			assert.True(t, returned, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails when its body returns")
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a record of the message at the caller's line", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Errorf("late at %d", site(&at))
				after = true
			}, nil, nil)
			want := assert.Failure{Contract: fmt.Sprintf("late at %d", at.Line), Where: at}
			assert.Equal(t, e.Case.Failures(), []assert.Failure{want}, "a record without an assertion")
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails when its body returns")
		})

		t.Run("keeps the zero location for a goroutine without the caller's code", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) { go c.Errorf("detached") }, nil, nil)
			arrived := func() bool { return len(e.Case.Failures()) == 1 }
			assert.EventuallyTrue(t, 10*time.Second, arrived, "the message arrives")
			assert.Equal(t, e.Case.Failures()[0].Where, assert.Where{}, "no frame of the caller's code")
		})
	})

	t.Run("Clock", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the clock of the test", func(t *testing.T) {
			t.Parallel()
			start := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
			var got assert.Clock
			engine.Replay(func(c *engine.Case) { got = c.Clock() }, nil, assert.NewControlled(start))
			assert.Equal(t, got.Now(), start, "the controlled clock's time")
		})

		t.Run("returns the system clock for no clock", func(t *testing.T) {
			t.Parallel()
			var got assert.Clock
			engine.Replay(func(c *engine.Case) { got = c.Clock() }, nil, nil)
			assert.Equal[assert.Clock](t, got, assert.System{}, "the runtime clock")
		})
	})

	t.Run("Assume", func(t *testing.T) {
		t.Parallel()

		t.Run("rejects the case for false", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Assume(false)
				after = true
			}, nil, nil)
			assert.False(t, after, "the body ends at the rejection")
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
		})

		t.Run("returns for true", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Assume(true)
				after = true
			}, nil, nil)
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CasePassed, "the case passes")
		})

		t.Run("rejects the case from another goroutine when the body returns", func(t *testing.T) {
			t.Parallel()
			var returned bool
			e := engine.Replay(func(c *engine.Case) {
				var wg sync.WaitGroup
				wg.Go(func() { c.Assume(false) })
				wg.Wait()
				returned = true
			}, nil, nil)
			assert.True(t, returned, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
		})
	})

	t.Run("Classify", func(t *testing.T) {
		t.Parallel()

		t.Run("counts a label once in a case", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Classify("small")
				c.Classify("even")
				c.Classify("small")
			}, nil, nil)
			assert.Equal(t, e.Case.Labels(), []string{"even", "small"}, "each label once, sorted")
		})
	})

	t.Run("Note", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the notes in order", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Note("opened the ledger")
				c.Note("appended two entries")
			}, nil, nil)
			assert.Equal(t, e.Case.Notes(), []string{"opened the ledger", "appended two entries"}, "both notes")
		})
	})

	t.Run("Observe", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the fingerprints in order", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Observe(3)
				c.Observe(1)
			}, nil, nil)
			assert.Equal(t, e.Case.Fingerprints(), []uint64{3, 1}, "both fingerprints")
		})
	})

	t.Run("History", func(t *testing.T) {
		t.Parallel()

		t.Run("returns one empty history to every call of the case", func(t *testing.T) {
			t.Parallel()
			var first, second *history.History
			engine.Replay(func(c *engine.Case) { first, second = c.History(), c.History() }, nil, nil)
			assert.True(t, first == second, "the same history")
			assert.Empty(t, first.Events(), "no event")
		})
	})

	t.Run("Target", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the highest score of each label", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Target("depth", 3)
				c.Target("depth", 1)
				c.Target("loss", -2)
				c.Target("width", 2)
				c.Target("width", 4)
			}, nil, nil)
			assert.Equal(t, e.Case.Targets(), map[string]float64{"depth": 3, "loss": -2, "width": 4},
				"the higher of two scores, and a first score below 0")
		})
	})

	t.Run("Rand", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a source that math/rand/v2 draws the case's choices from", func(t *testing.T) {
			t.Parallel()
			var got uint64
			body := func(c *engine.Case) { got = rand.New(c.Rand()).Uint64() }
			e := engine.Replay(body, []choice.Choice{unsigned(5)}, nil)
			assert.Equal(t, got, uint64(5), "the replayed choice")
			assert.True(t, sameChoices(e.Case.Choices(), []choice.Choice{unsigned(5)}), "one choice")
		})
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the cleanups after the body, the last registered first", func(t *testing.T) {
			t.Parallel()
			var order []string
			engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { order = append(order, "first") })
				c.Cleanup(func() { order = append(order, "second") })
				order = append(order, "body")
			}, nil, nil)
			assert.Equal(t, order, []string{"body", "second", "first"}, "the body, then the cleanups in reverse")
		})

		t.Run("runs the cleanups of a case whose body panicked", func(t *testing.T) {
			t.Parallel()
			var cleaned bool
			e := engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { cleaned = true })
				panic("broken ledger")
			}, nil, nil)
			assert.True(t, cleaned, "the cleanup ran")
			assert.Equal[any](t, e.Panic, "broken ledger", "the body's panic")
		})

		t.Run("runs the cleanups of a rejected case", func(t *testing.T) {
			t.Parallel()
			var cleaned bool
			e := engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { cleaned = true })
				c.Assume(false)
			}, nil, nil)
			assert.True(t, cleaned, "the cleanup ran")
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
		})

		t.Run("fails the case at a failure in a cleanup", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			e := engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { c.Fatalf("leaked at %d", site(&at)) })
			}, nil, nil)
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
			assert.Equal(t, e.Identity, engine.Identity{Where: at}, "the cleanup's location")
		})

		t.Run("keeps the body's failure as the case's failure", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { c.Fatalf("leaked") })
				c.Report(reported, false)
			}, nil, nil)
			assert.Equal(t, e.Identity, engine.Identity{Assertion: "equal", Where: reported.Where}, "the body's record")
		})

		t.Run("keeps the body's panic as the case's failure", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { panic("closing") })
				panic("broken ledger")
			}, nil, nil)
			assert.Equal[any](t, e.Panic, "broken ledger", "the body's panic")
		})

		t.Run("runs the later cleanups after a cleanup that ends the goroutine", func(t *testing.T) {
			t.Parallel()
			var first bool
			engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { first = true })
				c.Cleanup(func() { c.Fatalf("closing") })
				c.Fatalf("stop")
			}, nil, nil)
			assert.True(t, first, "the first cleanup ran")
		})

		t.Run("runs the later cleanups after a cleanup that panics", func(t *testing.T) {
			t.Parallel()
			var first bool
			e := engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { first = true })
				c.Cleanup(func() { panic("closing") })
			}, nil, nil)
			assert.True(t, first, "the first cleanup ran")
			assert.Equal[any](t, e.Panic, "closing", "the cleanup's panic")
		})

		t.Run("runs a cleanup that a cleanup registers before the earlier ones", func(t *testing.T) {
			t.Parallel()
			var order []string
			engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { order = append(order, "first") })
				c.Cleanup(func() {
					order = append(order, "second")
					c.Cleanup(func() { order = append(order, "registered") })
				})
			}, nil, nil)
			assert.Equal(t, order, []string{"second", "registered", "first"}, "the registered cleanup next")
		})

		t.Run("records a choice of a cleanup as a choice of the case", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Integer(digitRange)
				c.Cleanup(func() { c.Integer(digitRange) })
			}, integers(7, 3), nil)
			assert.True(t, sameChoices(e.Case.Choices(), integers(7, 3)), "the body's choice, then the cleanup's")
		})

		t.Run("ends a cleanup at a choice once the case has stopped", func(t *testing.T) {
			t.Parallel()
			var chose, first bool
			e := engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { first = true })
				c.Cleanup(func() {
					c.Integer(digitRange)
					chose = true
				})
				c.Assume(false)
			}, integers(7), nil)
			assert.False(t, chose, "the cleanup ends at its choice")
			assert.True(t, first, "the earlier cleanup runs")
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
			assert.Empty(t, e.Case.Choices(), "no choice recorded")
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a context that is live while the body runs", func(t *testing.T) {
			t.Parallel()
			var err error
			engine.Replay(func(c *engine.Case) { err = c.Context().Err() }, nil, nil)
			assert.NoError(t, err, "a live context")
		})

		t.Run("returns a context that the case cancels before its cleanups run", func(t *testing.T) {
			t.Parallel()
			var err error
			engine.Replay(func(c *engine.Case) {
				ctx := c.Context()
				c.Cleanup(func() { err = ctx.Err() })
			}, nil, nil)
			assert.ErrorIs(t, err, context.Canceled, "a cancelled context")
		})

		t.Run("returns a cancelled context to a cleanup that asks first", func(t *testing.T) {
			t.Parallel()
			var err error
			engine.Replay(func(c *engine.Case) {
				c.Cleanup(func() { err = c.Context().Err() })
			}, nil, nil)
			assert.ErrorIs(t, err, context.Canceled, "a cancelled context")
		})

		t.Run("returns one context to every call", func(t *testing.T) {
			t.Parallel()
			reads := make(chan context.Context, 2)
			engine.Replay(func(c *engine.Case) { reads <- c.Context(); reads <- c.Context() }, nil, nil)
			first, second := <-reads, <-reads
			assert.True(t, first == second, "the same context")
		})
	})

	t.Run("Uint64", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the case's next choice over the unsigned range", func(t *testing.T) {
			t.Parallel()
			var got []uint64
			recorded := []choice.Choice{unsigned(math.MaxUint64), unsigned(0)}
			e := engine.Replay(func(c *engine.Case) {
				source := c.Rand()
				got = []uint64{source.Uint64(), source.Uint64()}
			}, recorded, nil)
			assert.Equal(t, got, []uint64{math.MaxUint64, 0}, "the replayed choices")
			assert.True(t, sameChoices(e.Case.Choices(), recorded), "one choice for each value")
		})

		t.Run("returns 0 for a negative choice", func(t *testing.T) {
			t.Parallel()
			var got uint64
			e := engine.Replay(func(c *engine.Case) { got = c.Rand().Uint64() }, integers(-5), nil)
			assert.Equal(t, got, uint64(0), "the target of the unsigned range")
			assert.True(t, sameChoices(e.Case.Choices(), []choice.Choice{unsigned(0)}), "the coerced choice")
		})
	})

	t.Run("Choices", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Choices()[0] = unsigned(1)
			assert.True(t, sameChoices(c.Choices(), integers(7)), "the recorded choice")
		})
	})

	t.Run("Spans", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Spans()[0].Label = "changed"
			assert.Equal(t, labels(c.Spans()), []string{"integer"}, "the recorded span")
		})
	})

	t.Run("Draws", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Draws()[0].Label = "changed"
			assert.Equal(t, drawLabels(c.Draws()), []string{drawn}, "the recorded draw")
		})
	})

	t.Run("Labels", func(t *testing.T) {
		t.Parallel()

		t.Run("returns no label for an unclassified case", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(*engine.Case) {}, nil, nil)
			assert.Empty(t, e.Case.Labels(), "no label")
		})
	})

	t.Run("Notes", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Notes()[0] = "changed"
			assert.Equal(t, c.Notes(), []string{"seen"}, "the recorded note")
		})
	})

	t.Run("Fingerprints", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Fingerprints()[0] = 0
			assert.Equal(t, c.Fingerprints(), []uint64{42}, "the recorded fingerprint")
		})
	})

	t.Run("Targets", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Targets()["depth"] = 0
			assert.Equal(t, c.Targets(), map[string]float64{"depth": 3}, "the recorded score")
		})
	})

	t.Run("Failures", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Failures()[0].Assertion = "changed"
			assert.Equal(t, c.Failures(), []assert.Failure{reported}, "the recorded failure")
		})
	})

	t.Run("Position", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the index of the case's next choice", func(t *testing.T) {
			t.Parallel()
			var got []int
			engine.Replay(func(c *engine.Case) {
				got = append(got, c.Position())
				c.Integer(digitRange)
				got = append(got, c.Position())
			}, nil, nil)
			assert.Equal(t, got, []int{0, 1}, "before and after one choice")
		})
	})

	t.Run("Integer", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			give     []choice.Choice
			want     uint64
			recorded []choice.Choice
		}{
			{name: "returns a replayed choice inside its bounds", give: integers(7), want: 7, recorded: integers(7)},
			{
				name:     "returns the target for a replayed choice outside its bounds",
				give:     integers(12),
				want:     0,
				recorded: integers(0),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got uint64
				e := engine.Replay(func(c *engine.Case) { got = c.Integer(digitRange).Magnitude() }, tt.give, nil)
				assert.Equal(t, got, tt.want, "the value")
				assert.True(t, sameChoices(e.Case.Choices(), tt.recorded), "the recorded choice")
			})
		}

		t.Run("returns the upper bound on the edge case at the maximum", func(t *testing.T) {
			t.Parallel()
			got := failingAt(func(c *engine.Case) uint64 { return c.Integer(wideRange).Magnitude() }, wideMax)
			assert.Equal(t, got.Outcome, engine.Counterexample, "a case fails at the maximum")
			assert.Equal(t, got.Cases, 3, "the simplest case and random cases 0 and 1, as the definition pins")
		})
	})

	t.Run("Structure", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a replayed choice inside its bounds", func(t *testing.T) {
			t.Parallel()
			var got uint64
			e := engine.Replay(func(c *engine.Case) { got = c.Structure(digitRange, 0).Magnitude() }, integers(4), nil)
			assert.Equal(t, got, uint64(4), "the value")
			assert.True(t, sameChoices(e.Case.Choices(), integers(4)), "the recorded choice")
		})

		t.Run("returns its edge on the first edge case", func(t *testing.T) {
			t.Parallel()
			got := failingAt(func(c *engine.Case) uint64 { return c.Structure(wideRange, 123).Magnitude() }, 123)
			assert.Equal(t, got.Outcome, engine.Counterexample, "a case fails at the edge")
			assert.Equal(t, got.Cases, 2, "the simplest case and random case 0, as the definition pins")
		})
	})

	t.Run("Reusable", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a replayed choice inside its bounds", func(t *testing.T) {
			t.Parallel()
			var got uint64
			e := engine.Replay(func(c *engine.Case) { got = c.Reusable(digitRange).Magnitude() }, integers(4), nil)
			assert.Equal(t, got, uint64(4), "the value")
			assert.True(t, sameChoices(e.Case.Choices(), integers(4)), "the recorded choice")
		})

		t.Run("returns an earlier value of the case in some random cases", func(t *testing.T) {
			t.Parallel()
			repeats := 0
			for index := range uint64(100) {
				var first, second choice.Int
				engine.Generate(func(c *engine.Case) {
					first, second = c.Reusable(wideRange), c.Reusable(wideRange)
				}, 7, index, nil)
				if first == second {
					repeats++
				}
			}
			assert.True(t, repeats >= 25, "at least one case in four repeats the earlier value")
		})
	})

	t.Run("Coin", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []choice.Choice
			want bool
		}{
			{name: "reports true for a replayed 1", give: integers(1), want: true},
			{name: "reports false for a replayed 0", give: integers(0), want: false},
			{name: "reports false for the target", want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got bool
				engine.Replay(func(c *engine.Case) { got = c.Coin(1, 4) }, tt.give, nil)
				assert.Equal(t, got, tt.want, "the coin")
			})
		}

		t.Run("reports true in about one random case in four of odds 1 in 4", func(t *testing.T) {
			t.Parallel()
			heads := 0
			for index := range uint64(4000) {
				var got bool
				engine.Generate(func(c *engine.Case) { got = c.Coin(1, 4) }, 3, index, nil)
				if got {
					heads++
				}
			}
			assert.InRange(t, float64(heads)/4000, 0.22, 0.28, "about a quarter")
		})
	})

	t.Run("Count", func(t *testing.T) {
		t.Parallel()

		t.Run("returns 0 for an owner without a value in progress", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, leaked().Count(&owner{}), 0, "no count")
		})

		t.Run("returns the count of the innermost value in progress", func(t *testing.T) {
			t.Parallel()
			var outer, inner, after []int
			o := &owner{}
			engine.Replay(func(c *engine.Case) {
				end := c.Enter(o)
				c.Add(o)
				outer = append(outer, c.Count(o))
				endInner := c.Enter(o)
				c.Add(o)
				c.Add(o)
				inner = append(inner, c.Count(o))
				endInner()
				after = append(after, c.Count(o))
				end()
				after = append(after, c.Count(o))
			}, nil, nil)
			assert.Equal(t, [3][]int{outer, inner, after}, [3][]int{{1}, {2}, {1, 0}},
				"each value counts apart, and an ended one leaves the one around it")
		})
	})

	t.Run("Span", func(t *testing.T) {
		t.Parallel()

		t.Run("records a span around the choices that f makes", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Integer(digitRange)
				c.Span("outer", func() {
					c.Integer(digitRange)
					c.Span("inner", func() { c.Integer(digitRange) })
				})
			}, integers(1, 2, 3), nil)
			assert.Equal(t, e.Case.Spans(), []engine.Span{
				{Label: "outer", Start: 1, End: 3, Depth: 0, Parent: -1},
				{Label: "inner", Start: 2, End: 3, Depth: 1, Parent: 0},
			}, "the outer span from the second choice, and the inner one inside it")
		})

		t.Run("closes the span when f ends the goroutine", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Span("outer", func() {
					c.Integer(digitRange)
					c.Assume(false)
				})
			}, integers(1), nil)
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
			assert.Equal(t, e.Case.Spans(), []engine.Span{{Label: "outer", Start: 0, End: 1, Parent: -1}},
				"the span ends after the choice f made")
		})
	})
}

// TestCaseAllocs checks the allocation ceilings of the case's methods.
// A method that ends the calling goroutine, and a value that adds a choice,
// are measured on a whole replayed case.
func TestCaseAllocs(t *testing.T) {
	c := leaked()
	stop := func(c *engine.Case) { c.Fatalf("stop") }
	value := func(c *engine.Case) { c.Rand().Uint64() }
	assert.MaxAllocs(t, func() { c.Helper() }, 0, "Helper allocates nothing")
	assert.MaxAllocs(t, func() { c.Report(reported, false) }, reportAllocs, "Report allocates its message")
	assert.MaxAllocs(t, func() { engine.Replay(stop, nil, nil) }, fatalCaseAllocs, "a case that calls Fatalf")
	assert.MaxAllocs(t, func() { c.Errorf("late") }, errorfAllocs, "Errorf allocates its message and its frames")
	assert.MaxAllocs(t, func() { _ = c.Clock() }, 0, "Clock allocates nothing")
	assert.MaxAllocs(t, func() { c.Assume(true) }, 0, "Assume allocates nothing")
	assert.MaxAllocs(t, func() { c.Classify("small") }, 0, "Classify allocates nothing for a known label")
	assert.MaxAllocs(t, func() { c.Note("seen") }, 0, "Note allocates only to grow the record")
	assert.MaxAllocs(t, func() { c.Observe(42) }, 0, "Observe allocates only to grow the record")
	historical := func(c *engine.Case) { c.History() }
	assert.MaxAllocs(t, func() { engine.Replay(historical, nil, nil) }, historyCaseAllocs,
		"a case that calls History")
	assert.MaxAllocs(t, func() { c.Target("depth", 1) }, 0, "Target allocates nothing for a known label")
	assert.MaxAllocs(t, func() { _ = c.Targets() }, targetsAllocs, "Targets allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Position() }, 0, "Position allocates nothing")
	assert.MaxAllocs(t, func() { _ = c.Rand() }, 0, "Rand allocates nothing")
	assert.MaxAllocs(t, func() { c.Cleanup(nothing) }, 0, "Cleanup allocates only to grow its list")
	contextual := func(c *engine.Case) { _ = c.Context() }
	assert.MaxAllocs(t, func() { engine.Replay(contextual, nil, nil) }, contextCaseAllocs,
		"a case that calls Context")
	assert.MaxAllocs(t, func() { engine.Replay(value, nil, nil) }, valueCaseAllocs, "a case that draws one value")
	assert.MaxAllocs(t, func() { _ = c.Choices() }, copyAllocs, "Choices allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Spans() }, copyAllocs, "Spans allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Draws() }, copyAllocs, "Draws allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Labels() }, copyAllocs, "Labels allocates its list")
	assert.MaxAllocs(t, func() { _ = c.Notes() }, copyAllocs, "Notes allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Fingerprints() }, copyAllocs, "Fingerprints allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Failures() }, copyAllocs, "Failures allocates its copy")
	integer := func(c *engine.Case) { c.Integer(digitRange) }
	reusable := func(c *engine.Case) { c.Reusable(digitRange) }
	coin := func(c *engine.Case) { c.Coin(1, 4) }
	structure := func(c *engine.Case) { c.Structure(digitRange, 0) }
	spanned := func(c *engine.Case) { c.Span("outer", func() { c.Integer(digitRange) }) }
	counted := func(c *engine.Case) {
		o := &owner{}
		defer c.Enter(o)()
		c.Add(o)
		c.Count(o)
	}
	assert.MaxAllocs(t, func() { engine.Replay(integer, nil, nil) }, valueCaseAllocs,
		"a case that makes a value choice")
	assert.MaxAllocs(t, func() { engine.Replay(reusable, nil, nil) }, valueCaseAllocs,
		"a case that makes a reusable value choice")
	assert.MaxAllocs(t, func() { engine.Replay(coin, nil, nil) }, valueCaseAllocs, "a case that tosses a coin")
	assert.MaxAllocs(t, func() { _ = c.Count(c) }, 0, "Count allocates nothing")
	assert.MaxAllocs(t, func() { engine.Replay(counted, nil, nil) }, countedCaseAllocs,
		"a case that counts a value")
	assert.MaxAllocs(t, func() { engine.Replay(structure, nil, nil) }, valueCaseAllocs,
		"a case that makes a structure choice")
	assert.MaxAllocs(t, func() { engine.Replay(spanned, nil, nil) }, spanCaseAllocs,
		"a case that makes a choice in a span")
}

// BenchmarkCase measures each method of the case. A method that ends the
// calling goroutine, and a value that adds a choice, are measured on a
// whole replayed case.
func BenchmarkCase(b *testing.B) {
	b.Run("Helper", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Helper()
		}
		assert.Length(b, cs.Failures(), 1, "the case's one record")
	})

	b.Run("Report", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(reportAllocs)
		defer c.End()
		for c.Loop() {
			cs.Report(reported, false)
		}
		assert.Equal(b, cs.Failures()[0], reported, "the first record")
	})

	b.Run("Fatalf", func(b *testing.B) {
		var got engine.Execution
		stop := func(c *engine.Case) { c.Fatalf("stop") }
		c := bench.Start(b).MaxAllocs(fatalCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(stop, nil, nil)
		}
		assert.Equal(b, got.Status, engine.CaseFailed, "the case fails")
	})

	b.Run("Errorf", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(errorfAllocs)
		defer c.End()
		for c.Loop() {
			cs.Errorf("late")
		}
		assert.Equal(b, cs.Failures()[1].Contract, "late", "the message's record")
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
		assert.Length(b, cs.Choices(), 1, "no choice added")
	})

	b.Run("Classify", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Classify("small")
		}
		assert.Equal(b, cs.Labels(), []string{"small"}, "one label")
	})

	b.Run("Note", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Note("seen")
		}
		assert.Equal(b, cs.Notes()[0], "seen", "the first note")
	})

	b.Run("Observe", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Observe(42)
		}
		assert.Equal(b, cs.Fingerprints()[0], uint64(42), "the first fingerprint")
	})

	b.Run("History", func(b *testing.B) {
		var got engine.Execution
		historical := func(c *engine.Case) { c.History() }
		c := bench.Start(b).MaxAllocs(historyCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(historical, nil, nil)
		}
		assert.Equal(b, got.Status, engine.CasePassed, "the case passes")
	})

	b.Run("Target", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Target("depth", 1)
		}
		assert.Equal(b, cs.Targets(), map[string]float64{"depth": 3}, "the higher score")
	})

	b.Run("Targets", func(b *testing.B) {
		var got map[string]float64
		cs := leaked()
		c := bench.Start(b).MaxAllocs(targetsAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Targets()
		}
		assert.Equal(b, got, map[string]float64{"depth": 3}, "the recorded score")
	})

	b.Run("Position", func(b *testing.B) {
		var got int
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = cs.Position()
		}
		assert.Equal(b, got, 1, "the one choice of the case")
	})

	b.Run("Rand", func(b *testing.B) {
		var got engine.Source
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = cs.Rand()
		}
		assert.Equal(b, got, cs.Rand(), "the case's source")
	})

	b.Run("Cleanup", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Cleanup(nothing)
		}
		assert.Length(b, cs.Choices(), 1, "no choice added")
	})

	b.Run("Context", func(b *testing.B) {
		var got engine.Execution
		contextual := func(c *engine.Case) { _ = c.Context() }
		c := bench.Start(b).MaxAllocs(contextCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(contextual, nil, nil)
		}
		assert.Equal(b, got.Status, engine.CasePassed, "the case passes")
	})

	b.Run("Uint64", func(b *testing.B) {
		var got engine.Execution
		value := func(c *engine.Case) { c.Rand().Uint64() }
		c := bench.Start(b).MaxAllocs(valueCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(value, nil, nil)
		}
		assert.Length(b, got.Case.Choices(), 1, "one choice")
	})

	b.Run("Choices", func(b *testing.B) {
		var got []choice.Choice
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Choices()
		}
		assert.True(b, sameChoices(got, integers(7)), "the recorded choice")
	})

	b.Run("Spans", func(b *testing.B) {
		var got []engine.Span
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Spans()
		}
		assert.Equal(b, labels(got), []string{"integer"}, "the recorded span")
	})

	b.Run("Draws", func(b *testing.B) {
		var got []engine.Drawn
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Draws()
		}
		assert.Equal(b, drawLabels(got), []string{drawn}, "the recorded draw")
	})

	b.Run("Labels", func(b *testing.B) {
		var got []string
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Labels()
		}
		assert.Equal(b, got, []string{"small"}, "the label")
	})

	b.Run("Notes", func(b *testing.B) {
		var got []string
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Notes()
		}
		assert.Equal(b, got, []string{"seen"}, "the note")
	})

	b.Run("Fingerprints", func(b *testing.B) {
		var got []uint64
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Fingerprints()
		}
		assert.Equal(b, got, []uint64{42}, "the fingerprint")
	})

	b.Run("Failures", func(b *testing.B) {
		var got []assert.Failure
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Failures()
		}
		assert.Equal(b, got, []assert.Failure{reported}, "the record")
	})

	b.Run("Integer", func(b *testing.B) {
		var got engine.Execution
		integer := func(c *engine.Case) { c.Integer(digitRange) }
		c := bench.Start(b).MaxAllocs(valueCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(integer, nil, nil)
		}
		assert.Length(b, got.Case.Choices(), 1, "one choice")
	})

	b.Run("Reusable", func(b *testing.B) {
		var got engine.Execution
		reusable := func(c *engine.Case) { c.Reusable(digitRange) }
		c := bench.Start(b).MaxAllocs(valueCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(reusable, nil, nil)
		}
		assert.Length(b, got.Case.Choices(), 1, "one choice")
	})

	b.Run("Coin", func(b *testing.B) {
		var got engine.Execution
		coin := func(c *engine.Case) { c.Coin(1, 4) }
		c := bench.Start(b).MaxAllocs(valueCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(coin, nil, nil)
		}
		assert.Length(b, got.Case.Choices(), 1, "one choice")
	})

	b.Run("Count", func(b *testing.B) {
		var got int
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = cs.Count(cs)
		}
		assert.Equal(b, got, 0, "no value in progress")
	})

	b.Run("Enter", func(b *testing.B) {
		var got int
		counted := func(c *engine.Case) {
			o := &owner{}
			defer c.Enter(o)()
			c.Add(o)
			got = c.Count(o)
		}
		c := bench.Start(b).MaxAllocs(countedCaseAllocs)
		defer c.End()
		for c.Loop() {
			engine.Replay(counted, nil, nil)
		}
		assert.Equal(b, got, 1, "one counted")
	})

	b.Run("Structure", func(b *testing.B) {
		var got engine.Execution
		structure := func(c *engine.Case) { c.Structure(digitRange, 0) }
		c := bench.Start(b).MaxAllocs(valueCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(structure, nil, nil)
		}
		assert.Length(b, got.Case.Choices(), 1, "one choice")
	})

	b.Run("Span", func(b *testing.B) {
		var got engine.Execution
		spanned := func(c *engine.Case) { c.Span("outer", func() { c.Integer(digitRange) }) }
		c := bench.Start(b).MaxAllocs(spanCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(spanned, nil, nil)
		}
		assert.Length(b, got.Case.Spans(), 1, "one span")
	})
}

// nothing is a cleanup that does nothing.
func nothing() {}

// failingAt runs a property of the reference seed, without shrinking, whose
// body makes the one choice that choose makes and fails when the choice is
// value.
func failingAt(choose func(*engine.Case) uint64, value uint64) engine.Result {
	s := settled()
	s.Shrink = 0
	return engine.Run(func(c *engine.Case) {
		if choose(c) == value {
			c.Report(reported, false)
		}
	}, s)
}
